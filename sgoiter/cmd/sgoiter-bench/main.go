// Command sgoiter-bench — Banc d'évaluation et de métrologie C2Go (sgoiter vs C vs ccgo)
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/sgoiterbench"
	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/tribench"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "report" {
		runReportCmd(os.Args[2:])
		return
	}

	root := flag.String("root", "", "racine c2simd (défaut: déduit de cwd ou /devhoros/c2simd)")
	out := flag.String("out", "", "répertoire rapport")
	sgo := flag.String("sgoiter", "", "binaire sgoiter")
	only := flag.String("only", "", "libs CSV (ex: fnv1a_64,fast_xor)")
	heavy := flag.Bool("heavy", false, "inclure les jeux de test sous forte charge (1M, 10M, 100M)")
	skipCcgo := flag.Bool("skip-ccgo", false, "ne pas lancer ccgo")
	skipBench := flag.Bool("skip-bench", false, "pas de boucle bench")
	v := flag.Bool("v", true, "mode verbose")
	flag.Parse()

	r := *root
	if r == "" {
		cwd, _ := os.Getwd()
		for d := cwd; d != "/" && d != "."; d = filepath.Dir(d) {
			if filepath.Base(d) == "c2simd" {
				r = d; break
			}
			if _, err := os.Stat(filepath.Join(d, "sgoiter")); err == nil && filepath.Base(d) == "c2simd" {
				r = d; break
			}
		}
		if r == "" {
			if _, err := os.Stat(filepath.Join(cwd, "sgoiter")); err == nil {
				r = cwd
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
		Heavy:      *heavy,
		Verbose:    *v,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur sgoiter-bench: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(tribench.FormatSummaryMD(rep))
	fmt.Printf("\nRapport disponible dans : %s/report.json\n", rep.OutDir)

	if rep.Summary.SgoiterMatch < rep.Summary.LibsTotal {
		os.Exit(2)
	}
}

func runReportCmd(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: sgoiter-bench report <chemin/vers/report.json>")
		os.Exit(1)
	}
	path := args[0]
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur lecture fichier %s: %v\n", path, err)
		os.Exit(1)
	}

	var rep sgoiterbench.Report
	if err := json.Unmarshal(data, &rep); err != nil {
		fmt.Fprintf(os.Stderr, "Erreur parsing JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(sgoiterbench.FormatSummaryMD(&rep))
}
