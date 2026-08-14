// Command probebench — sondes stratifiées CPU/RAM + cartographie disques I/O.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/probebench"
)

func main() {
	root := flag.String("root", "/devhoros/c2simd", "racine c2simd")
	work := flag.String("work", "", "workdir probes (défaut: /tmp NVMe, jamais /data)")
	out := flag.String("out", "", "répertoire rapport")
	sgo := flag.String("sgoiter", "", "binaire sgoiter")
	ccgo := flag.String("ccgo", "", "binaire ccgo (défaut: PATH)")
	only := flag.String("only", "", "libs CSV")
	skipIO := flag.Bool("skip-io", false, "pas de probe disque 64MiB")
	skipCcgo := flag.Bool("skip-ccgo", false, "ne pas sonder ccgo")
	requireCcgo := flag.Bool("require-ccgo", false, "avec -validate: exiger aussi des lignes ccgo saines")
	repeat := flag.Int("repeat", 1, "répéter chaque strate N fois et garder la médiane ns/op")
	validate := flag.String("validate", "", "vérifie un probe_report.json existant (gate commit) et exit 0/1")
	flag.Parse()

	if *validate != "" {
		raw, err := os.ReadFile(*validate)
		if err != nil {
			fmt.Fprintf(os.Stderr, "probebench -validate: %v\n", err)
			os.Exit(1)
		}
		var rep probebench.Report
		if err := json.Unmarshal(raw, &rep); err != nil {
			fmt.Fprintf(os.Stderr, "probebench -validate: json: %v\n", err)
			os.Exit(1)
		}
		if err := probebench.ValidateSgoiterProbes(rep.Probes); err != nil {
			fmt.Fprintf(os.Stderr, "probebench -validate FAIL: %v\n", err)
			os.Exit(1)
		}
		if *requireCcgo {
			if err := probebench.ValidateCcgoProbes(rep.Probes); err != nil {
				fmt.Fprintf(os.Stderr, "probebench -validate -require-ccgo FAIL: %v\n", err)
				os.Exit(1)
			}
		}
		fmt.Printf("probebench -validate OK (%d probes)\n", len(rep.Probes))
		return
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
	sg := *sgo
	if sg == "" {
		sg = filepath.Join(*root, "bin/sgoiter")
	}
	rep, err := probebench.Run(probebench.Options{
		C2simdRoot: *root,
		WorkDir:    *work,
		OutDir:     *out,
		SgoiterBin: sg,
		CcgoBin:    *ccgo,
		Only:       onlyList,
		SkipIO:     *skipIO,
		SkipCcgo:   *skipCcgo,
		Repeat:     *repeat,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "probebench: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(probebench.FormatReportMD(rep))
	fmt.Printf("\nJSON: %s/probe_report.json\n", rep.OutDir)
	fmt.Fprintf(os.Stderr, "workdir=%s\n  %s\n", rep.WorkDir, probebench.FormatDiskEnv(rep.WorkDisk))
}
