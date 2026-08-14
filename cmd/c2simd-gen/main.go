package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"code.hazyhaar.fr/devhoros/c2simd/internal/astmatch"
)

func main() {
	inPath := flag.String("in", "", "Fichier Go transpilé source en entrée")
	outPath := flag.String("out", "", "Fichier Go transformé en sortie")
	stats := flag.Bool("stats", false, "Écrire un JSON de métriques raw→opt sur stdout (après le message OK)")
	flag.Parse()

	if *inPath == "" || *outPath == "" {
		fmt.Fprintln(os.Stderr, "Usage: c2simd-gen -in <file.go> -out <file_opt.go> [-stats]")
		os.Exit(1)
	}

	src, err := os.ReadFile(*inPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur lecture %s: %v\n", *inPath, err)
		os.Exit(1)
	}

	before := astmatch.CountMarkers(src)

	res, err := astmatch.TransformAST(src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur transformation AST: %v\n", err)
		os.Exit(1)
	}

	after := astmatch.CountMarkers(res)

	if err := os.WriteFile(*outPath, res, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Erreur écriture %s: %v\n", *outPath, err)
		os.Exit(1)
	}

	fmt.Printf("c2simd-gen: transformé avec succès %s -> %s\n", *inPath, *outPath)

	if *stats {
		payload := map[string]any{
			"in":  *inPath,
			"out": *outPath,
			"raw": before,
			"opt": after,
			"delta": map[string]int{
				"ccgo_up":     after.CcgoUp - before.CcgoUp,
				"bits_rotate": after.BitsRotate - before.BitsRotate,
				"rotl_calls":  after.RotlCalls - before.RotlCalls,
				"tls_param":   after.TLSParamFn - before.TLSParamFn,
				"lines":       after.Lines - before.Lines,
			},
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(payload)
	}
}
