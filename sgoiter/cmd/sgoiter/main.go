// Command sgoiter — CLI transpileur itératif (front → rules → emit go127).
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/emit"
	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/front"
	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/ir"
	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/rules"
)

func main() {
	fs := flag.NewFlagSet("sgoiter", flag.ExitOnError)
	in := fs.String("in", "", "fichier .c subset v0 ou .ir.json")
	out := fs.String("out", "", "sortie .go (emit go127)")
	splitDir := fs.String("split-dir", "", "répertoire de sortie modulaire (découpe par domaine)")
	pkgFlag := fs.String("pkg", "", "nom du package Go (défaut: nom du module C ou 'generated')")
	irOut := fs.String("ir-out", "", "optionnel : écrire IR JSON")
	opt := fs.Bool("opt", true, "appliquer règles IR")
	mode := fs.String("mode", "safe", "emit mode: safe|kernel (kernel=unsafe LE word paths)")
	preferIR := fs.Bool("prefer-ir", false, "skip body overrides (fnv/blake/…) — absorb/debug")
	rootsOnly := fs.Bool("roots-only", false, "n'émettre que les racines et leur clôture d'appels (défaut : les fonctions non-static)")
	roots := fs.String("roots", "", "racines explicites, séparées par des virgules (implique -roots-only)")
	exclude := fs.String("exclude", "", "symboles ou structures à exclure de l'émission, séparés par des virgules")
	stripDeadGlobals := fs.Bool("strip-dead-globals", false, "supprimer les globales jamais référencées dans le fichier émis (fixpoint) — opt-in : une strate hand_*.go hors émission peut en consommer")
	keepGlobals := fs.String("keep-globals", "", "globales à protéger de -strip-dead-globals, séparées par des virgules")
	_ = fs.Parse(os.Args[1:])

	if *in == "" || (*out == "" && *splitDir == "") {
		fmt.Fprintln(os.Stderr, "usage: sgoiter -in file.c|.ir.json (-out file.go | -split-dir dir/) [-mode safe|kernel] [-ir-out m.ir.json] [-opt=true]")
		os.Exit(2)
	}
	emMode := emit.ModeSafe
	switch strings.ToLower(*mode) {
	case "safe", "":
		emMode = emit.ModeSafe
	case "kernel":
		emMode = emit.ModeKernel
	default:
		fmt.Fprintf(os.Stderr, "sgoiter: unknown -mode %q (safe|kernel)\n", *mode)
		os.Exit(2)
	}

	if filepath.Ext(*in) == ".go" {
		srcBytes, err := os.ReadFile(*in)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		pkgName := *pkgFlag
		if pkgName == "" {
			pkgName = "generated"
		}
		if *splitDir != "" {
			if err := os.MkdirAll(*splitDir, 0o755); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			domainFiles, err := emit.SplitGoSource(string(srcBytes), pkgName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "sgoiter split error: %v\n", err)
				os.Exit(1)
			}
			for fname, fcontent := range domainFiles {
				destPath := filepath.Join(*splitDir, fname)
				if err := os.WriteFile(destPath, []byte(fcontent), 0o644); err != nil {
					fmt.Fprintf(os.Stderr, "sgoiter write %s error: %v\n", destPath, err)
					os.Exit(1)
				}
				fmt.Printf("sgoiter split: %s (%d L)\n", destPath, len(strings.Split(fcontent, "\n")))
			}
		}
		return
	}

	var m *ir.Module
	var err error
	switch filepath.Ext(*in) {
	case ".json":
		m, err = ir.Load(*in)
	default:
		m, err = front.ParseFile(*in)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *pkgFlag != "" {
		m.Name = *pkgFlag
	}

	if *opt {
		m, err = rules.ApplyAll(m)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	if *rootsOnly || *roots != "" {
		before := len(m.Funcs)
		var list []string
		if *roots != "" {
			list = strings.Split(*roots, ",")
		}
		m = emit.RootClosure(m, list)
		fmt.Fprintf(os.Stderr, "sgoiter: clôture des racines — %d fonction(s) sur %d retenues\n", len(m.Funcs), before)
	}

	// A stub means a symbol the module calls but the front could not harvest.
	// Naming them here keeps the gap visible instead of leaving it to a panic
	// at run time.
	if stubs := emit.FillStubs(m); len(stubs) > 0 {
		fmt.Fprintf(os.Stderr, "sgoiter: %d symbol(s) stubbed (called but not harvested): %s\n",
			len(stubs), strings.Join(stubs, ", "))
	}

	if *irOut != "" {
		b, err := ir.Marshal(m)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := os.WriteFile(*irOut, b, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	var excludedList []string
	if *exclude != "" {
		for _, s := range strings.Split(*exclude, ",") {
			if s = strings.TrimSpace(s); s != "" {
				excludedList = append(excludedList, s)
			}
		}
	}

	var keepList []string
	if *keepGlobals != "" {
		for _, s := range strings.Split(*keepGlobals, ",") {
			if s = strings.TrimSpace(s); s != "" {
				keepList = append(keepList, s)
			}
		}
	}

	src, err := emit.EmitOpts(m, emit.Options{
		Profile:          emit.ProfileGo127,
		Mode:             emMode,
		PreferIR:         *preferIR,
		ExcludeSymbols:   excludedList,
		StripDeadGlobals: *stripDeadGlobals,
		KeepGlobals:      keepList,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *out != "" {
		if err := os.WriteFile(*out, []byte(src), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("sgoiter: %s → %s\n", *in, *out)
	}

	if *splitDir != "" {
		if err := os.MkdirAll(*splitDir, 0o755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		pkgName := m.Name
		if pkgName == "" {
			pkgName = "generated"
		}
		domainFiles, err := emit.SplitGoSource(src, pkgName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "sgoiter split error: %v\n", err)
			os.Exit(1)
		}
		for fname, fcontent := range domainFiles {
			destPath := filepath.Join(*splitDir, fname)
			if err := os.WriteFile(destPath, []byte(fcontent), 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "sgoiter write %s error: %v\n", destPath, err)
				os.Exit(1)
			}
			fmt.Printf("sgoiter split: %s (%d L)\n", destPath, len(strings.Split(fcontent, "\n")))
		}
	}
}
