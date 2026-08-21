// p2go-bench — banc de mesure de la campagne v0.4 : pour chaque charge
// testdata/bench/*.php, mesure le temps d'exécution (meilleur de N runs) de
// trois voies — php CLI interprété, Go transpilé scalaire (toolchain défaut),
// Go transpilé SIMD (go1.27rc3 + GOEXPERIMENT=simd) — et VÉRIFIE d'abord que
// les trois stdout sont bit-exacts (le banc porte son propre gate de parité).
// Sortie : table markdown (débit relatif) sur stdout.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"

	"code.hazyhaar.fr/devhoros/c2simd/p2go"
)

const simdToolchain = "go1.27rc3"

func main() {
	bench := flag.String("bench", "testdata/bench", "répertoire des charges PHP")
	runs := flag.Int("runs", 3, "runs par voie, meilleur temps retenu")
	flag.Parse()

	phpBin, err := exec.LookPath("php")
	if err != nil {
		fail("oracle php absent du PATH")
	}
	paths, err := filepath.Glob(filepath.Join(*bench, "*.php"))
	if err != nil || len(paths) == 0 {
		fail(fmt.Sprintf("aucune charge dans %s", *bench))
	}
	sort.Strings(paths)

	fmt.Println("| charge | php (ms) | go scalaire (ms) | go simd (ms) | scalaire/php | simd/scalaire | parité |")
	fmt.Println("|---|---|---|---|---|---|---|")
	for _, path := range paths {
		name := filepath.Base(path)
		src, err := os.ReadFile(path)
		if err != nil {
			fail(err.Error())
		}
		files, terr := p2go.TranspileFiles(string(src), name)
		if terr != nil {
			fail(name + " : " + terr.Error())
		}
		dir, err := os.MkdirTemp("", "p2go-bench-*")
		if err != nil {
			fail(err.Error())
		}
		defer os.RemoveAll(dir)
		for fname, content := range files {
			if err := os.WriteFile(filepath.Join(dir, fname), []byte(content), 0o644); err != nil {
				fail(err.Error())
			}
		}
		if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module p2go_bench\n\ngo 1.24\n"), 0o644); err != nil {
			fail(err.Error())
		}

		// Builds : scalaire (toolchain défaut) et simd (toolchain épinglée).
		binScalar := filepath.Join(dir, "bin_scalar")
		build(dir, binScalar, nil)
		binSimd := filepath.Join(dir, "bin_simd")
		build(dir, binSimd, []string{"GOTOOLCHAIN=" + simdToolchain, "GOEXPERIMENT=simd"})

		// Gate de parité tri-voies AVANT toute mesure.
		outPhp := capture(phpBin, path)
		outScalar := capture(binScalar)
		outSimd := capture(binSimd)
		parity := "ok"
		if outScalar != outPhp || outSimd != outPhp {
			parity = "DIVERGENCE"
		}

		tPhp := best(*runs, func() { capture(phpBin, path) })
		tScalar := best(*runs, func() { capture(binScalar) })
		tSimd := best(*runs, func() { capture(binSimd) })

		fmt.Printf("| %s | %.1f | %.1f | %.1f | ×%.1f | ×%.2f | %s |\n",
			name, ms(tPhp), ms(tScalar), ms(tSimd),
			tPhp.Seconds()/tScalar.Seconds(),
			tScalar.Seconds()/tSimd.Seconds(),
			parity)
		if parity != "ok" {
			fail(name + " : divergence de parité — mesures invalides")
		}
	}
}

func build(dir, out string, extraEnv []string) {
	cmd := exec.Command("go", "build", "-o", out, ".")
	cmd.Dir = dir
	if len(extraEnv) > 0 {
		cmd.Env = append(os.Environ(), extraEnv...)
	}
	if b, err := cmd.CombinedOutput(); err != nil {
		fail("go build : " + err.Error() + "\n" + string(b))
	}
}

func capture(bin string, args ...string) string {
	cmd := exec.Command(bin, args...)
	out, err := cmd.Output()
	if err != nil {
		fail(bin + " : " + err.Error())
	}
	return string(out)
}

func best(runs int, f func()) time.Duration {
	bestD := time.Duration(1<<62 - 1)
	for r := 0; r < runs; r++ {
		t0 := time.Now()
		f()
		if d := time.Since(t0); d < bestD {
			bestD = d
		}
	}
	return bestD
}

func ms(d time.Duration) float64 { return float64(d.Microseconds()) / 1000 }

func fail(msg string) {
	fmt.Fprintln(os.Stderr, "p2go-bench:", msg)
	os.Exit(1)
}
