// p2go-dogfood — balayage du corpus de dogfooding (Jalon 3) : verdict par
// source (ok / refused err_* / build_fail), rapport JSON.
// Usage : p2go-dogfood -corpus testdata/dogfood [-report rapport.json]
package main

import (
	"flag"
	"fmt"
	"os"

	"code.hazyhaar.fr/devhoros/c2simd/p2go/dogfood"
)

func main() {
	corpus := flag.String("corpus", "testdata/dogfood", "répertoire des sources PHP du corpus")
	report := flag.String("report", "", "chemin du rapport JSON (défaut : stdout)")
	flag.Parse()
	work, err := os.MkdirTemp("", "p2go-dogfood-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, "p2go-dogfood:", err)
		os.Exit(1)
	}
	defer os.RemoveAll(work)
	results, err := dogfood.Sweep(*corpus, work)
	if err != nil {
		fmt.Fprintln(os.Stderr, "p2go-dogfood:", err)
		os.Exit(1)
	}
	if *report != "" {
		if err := dogfood.WriteReport(results, *report); err != nil {
			fmt.Fprintln(os.Stderr, "p2go-dogfood:", err)
			os.Exit(1)
		}
	}
	nOK, nRef, nFail := 0, 0, 0
	for _, r := range results {
		switch r.Status {
		case "ok":
			nOK++
		case "refused":
			nRef++
		default:
			nFail++
		}
		fmt.Printf("%-24s %-10s %s %s\n", r.File, r.Status, r.Code, r.Msg)
	}
	fmt.Printf("total %d : %d ok, %d refused, %d build_fail\n", len(results), nOK, nRef, nFail)
	if nFail > 0 { // un Go émis qui ne compile pas = bug d'emit, échec dur
		os.Exit(1)
	}
}
