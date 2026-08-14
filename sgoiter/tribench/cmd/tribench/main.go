// Command tribench — banc C vs ccgo vs sgoiter (fixtures + observabilité).
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/tribench"
)

func main() {
	root := flag.String("root", "", "racine c2simd (défaut: déduit de l'exécutable ou cwd)")
	out := flag.String("out", "", "répertoire rapport")
	sgo := flag.String("sgoiter", "", "binaire sgoiter")
	only := flag.String("only", "", "libs CSV (ex: fnv1a_64,fast_xor)")
	skipCcgo := flag.Bool("skip-ccgo", false, "ne pas lancer ccgo")
	skipBench := flag.Bool("skip-bench", false, "pas de boucle bench")
	pprof := flag.Bool("pprof", false, "marque chemins pprof (hook)")
	v := flag.Bool("v", true, "verbose")
	flag.Parse()

	r := *root
	if r == "" {
		cwd, _ := os.Getwd()
		// walk up for c2simd
		for d := cwd; d != "/" && d != "."; d = filepath.Dir(d) {
			if filepath.Base(d) == "c2simd" {
				r = d
				break
			}
			if _, err := os.Stat(filepath.Join(d, "sgoiter")); err == nil && filepath.Base(d) == "c2simd" {
				r = d
				break
			}
		}
		if r == "" {
			// try cwd/c2simd or parent
			if _, err := os.Stat(filepath.Join(cwd, "sgoiter", "tribench")); err == nil {
				r = cwd
			} else if _, err := os.Stat(filepath.Join(cwd, "c2simd", "sgoiter")); err == nil {
				r = filepath.Join(cwd, "c2simd")
			} else {
				r = "/devhoros/c2simd"
			}
		}
	}

	var onlyList []string
	if *only != "" {
		for _, p := range strings.Split(*only, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				onlyList = append(onlyList, p)
			}
		}
	}

	rep, err := tribench.Run(tribench.Options{
		C2simdRoot: r,
		OutDir:     *out,
		SgoiterBin: *sgo,
		Only:       onlyList,
		SkipCcgo:   *skipCcgo,
		SkipBench:  *skipBench,
		Pprof:      *pprof,
		Verbose:    *v,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(tribench.FormatSummaryMD(rep))
	fmt.Printf("\nreport: %s/report.json\n", rep.OutDir)
	// Gate on the kernels that had an oracle. A kernel with no C reference is
	// not a pass and not a failure — counting it as a pass is what made the
	// score read 12/12 while only 11 were ever compared.
	if rep.Summary.SgoiterMatch < rep.Summary.LibsCompared {
		os.Exit(2)
	}
}
