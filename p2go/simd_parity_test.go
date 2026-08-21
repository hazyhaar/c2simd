package p2go_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd/p2go"
	"code.hazyhaar.fr/devhoros/c2simd/p2go/phpt"
)

// Toolchain épinglée du pôle (décret 2026-08-14, /devhoros/c2simd/CLAUDE.md) :
// la strate SIMD ne se valide que sous go1.27rc3 + GOEXPERIMENT=simd.
const simdToolchain = "go1.27rc3"

// TestSimdEmitted vérifie que la fixture de réduction produit bien les helpers
// duals (l'abaissement SumLoop a eu lieu — pas un faux vert scalaire).
func TestSimdEmitted(t *testing.T) {
	c, err := phpt.ParseFile(filepath.Join("testdata", "phpt", "array_sum.phpt"))
	if err != nil {
		t.Fatal(err)
	}
	files, err := p2go.TranspileFiles(c.File, c.Name)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := files["p2go_simd_on.go"]; !ok {
		t.Fatalf("helper SIMD absent : la règle SumLoop n'a pas reconnu la boucle ; fichiers émis : %v", keys(files))
	}
	if !strings.Contains(files["main.go"], "p2goSumI64(") {
		t.Fatal("main.go n'appelle pas p2goSumI64 — SumLoop non émis")
	}
}

// TestSimdParity exécute les fixtures des QUATRE règles SIMD sous la toolchain
// épinglée avec GOEXPERIMENT=simd (chemin archsimd réel, garde AVX2) et
// compare au EXPECT — parité bit-exacte scalaire/vectorisé exigée (CLAUDE.md
// règle 4). Échec explicite si la toolchain du pôle manque : fail-loud.
func TestSimdParity(t *testing.T) {
	if _, err := exec.LookPath(simdToolchain); err != nil {
		home, _ := os.UserHomeDir()
		if _, serr := os.Stat(filepath.Join(home, "go", "bin", simdToolchain)); serr != nil {
			t.Fatalf("toolchain %s introuvable (PATH et ~/go/bin) — installer la toolchain épinglée du pôle", simdToolchain)
		}
	}
	fixtures := []string{"array_sum.phpt", "simd_dot.phpt", "simd_minmax.phpt", "ascii_case_long.phpt"}
	env := []string{"GOTOOLCHAIN=" + simdToolchain, "GOEXPERIMENT=simd"}
	for _, fx := range fixtures {
		fx := fx
		t.Run(fx, func(t *testing.T) {
			t.Parallel()
			c, err := phpt.ParseFile(filepath.Join("testdata", "phpt", fx))
			if err != nil {
				t.Fatal(err)
			}
			if err := phpt.RunEnv(c, t.TempDir(), env); err != nil {
				t.Fatalf("parité SIMD : %v", err)
			}
		})
	}
}

// TestSimdRulesEmitted vérifie au sol que chaque règle a bien abaissé sa
// boucle (pas un faux vert scalaire) : l'helper attendu est appelé dans main.go.
func TestSimdRulesEmitted(t *testing.T) {
	cases := map[string]string{
		"simd_dot.phpt":        "p2goDotI64(",
		"simd_minmax.phpt":     "p2goMaxI64(",
		"ascii_case_long.phpt": "p2goToUpper(",
	}
	for fx, marker := range cases {
		c, err := phpt.ParseFile(filepath.Join("testdata", "phpt", fx))
		if err != nil {
			t.Fatal(err)
		}
		files, err := p2go.TranspileFiles(c.File, c.Name)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(files["main.go"], marker) {
			t.Fatalf("%s : %s absent de main.go — règle non appliquée\n%s", fx, marker, files["main.go"])
		}
		if _, ok := files["p2go_simd_on.go"]; !ok {
			t.Fatalf("%s : fichier archsimd absent", fx)
		}
	}
}

func keys(m map[string]string) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}
