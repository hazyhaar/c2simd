package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hazyhaar/c2simd/internal/astmatch"
)

func main() {
	inPath := flag.String("in", "", "Fichier Go transpilé source en entrée")
	outPath := flag.String("out", "", "Fichier Go transformé en sortie")
	flag.Parse()

	if *inPath == "" || *outPath == "" {
		fmt.Fprintln(os.Stderr, "Usage: c2simd-gen -in <file.go> -out <file_simd.go>")
		os.Exit(1)
	}

	src, err := os.ReadFile(*inPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur lecture %s: %v\n", *inPath, err)
		os.Exit(1)
	}

	res, err := astmatch.TransformRotations(src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur transformation AST: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*outPath, res, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Erreur écriture %s: %v\n", *outPath, err)
		os.Exit(1)
	}

	fmt.Printf("c2simd-gen: transformé avec succès %s -> %s\n", *inPath, *outPath)
}
