// p2go — CLI du transpileur PHP→Go 1.27 (subset v0.1, SPEC.md).
// Usage : p2go -in prog.php [-o out.go]  (défaut : stdout)
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"code.hazyhaar.fr/devhoros/c2simd/p2go"
)

func main() {
	in := flag.String("in", "", "fichier source PHP (subset v0.1)")
	out := flag.String("o", "", "fichier Go de sortie mono-fichier (défaut : stdout)")
	outdir := flag.String("outdir", "", "répertoire de sortie (requis si helpers SIMD multi-fichiers)")
	flag.Parse()
	if *in == "" {
		fmt.Fprintln(os.Stderr, "usage: p2go -in prog.php [-o out.go | -outdir dir]")
		os.Exit(2)
	}
	src, err := os.ReadFile(*in)
	if err != nil {
		fmt.Fprintln(os.Stderr, "p2go:", err)
		os.Exit(1)
	}
	files, err := p2go.TranspileFiles(string(src), *in)
	if err != nil {
		fmt.Fprintln(os.Stderr, "p2go:", err)
		os.Exit(1)
	}
	if *outdir != "" {
		if err := os.MkdirAll(*outdir, 0o755); err != nil {
			fmt.Fprintln(os.Stderr, "p2go:", err)
			os.Exit(1)
		}
		for name, content := range files {
			if err := os.WriteFile(filepath.Join(*outdir, name), []byte(content), 0o644); err != nil {
				fmt.Fprintln(os.Stderr, "p2go:", err)
				os.Exit(1)
			}
		}
		return
	}
	if len(files) > 1 {
		fmt.Fprintln(os.Stderr, "p2go: sortie multi-fichiers (helpers SIMD) — utiliser -outdir")
		os.Exit(1)
	}
	if *out == "" {
		fmt.Print(files["main.go"])
		return
	}
	if err := os.WriteFile(*out, []byte(files["main.go"]), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "p2go:", err)
		os.Exit(1)
	}
}
