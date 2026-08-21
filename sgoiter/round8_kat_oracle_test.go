package sgoiter_test

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/emit"
	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/front"
	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/rules"
)

// TestRound8SimdAndTerminalOracles valide la conformité bit-exacte de la transpilation
// sgoiter pour le Round 8 face à un oracle compilé avec gcc -O2.
func TestRound8SimdAndTerminalOracles(t *testing.T) {
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("gcc non disponible sur l'hôte")
	}

	t.Run("SimSIMD_Dot_F32_Vs_GCC", testSimsimdDotVsGCC)
	t.Run("SimSIMD_L2sq_F32_Vs_GCC", testSimsimdL2sqVsGCC)
	t.Run("FastHadamard_FHT_Vs_GCC", testFastHadamardVsGCC)
}

func testSimsimdDotVsGCC(t *testing.T) {
	cSource, err := filepath.Abs(filepath.Join("..", "spec", "c_sources", "testdata", "c_sources", "simsimd_dot_f32.c"))
	if err != nil {
		t.Fatal(err)
	}

	m, err := front.ParseFile(cSource)
	if err != nil {
		t.Fatalf("front.ParseFile: %v", err)
	}
	m, err = rules.ApplyAll(m)
	if err != nil {
		t.Fatalf("rules.ApplyAll: %v", err)
	}
	m.Name = "main"
	goCode, err := emit.Emit(m, emit.ProfileGo127)
	if err != nil {
		t.Fatalf("emit.Emit: %v", err)
	}

	dir := t.TempDir()
	mainC := fmt.Sprintf(`#include <stdio.h>
#include <stdint.h>
#include <stddef.h>
#include "%s"

int main(void) {
    float a[8] = {1.5f, 2.0f, -3.5f, 4.25f, 0.5f, -1.0f, 2.5f, 3.0f};
    float b[8] = {2.0f, -1.5f, 1.0f, 2.0f, 4.0f, 3.0f, -2.0f, 1.5f};
    double res = 0.0;
    simsimd_dot_f32(a, b, 8, &res);
    printf("%%.10f\n", res);
    return 0;
}
`, cSource)

	cMainPath := filepath.Join(dir, "main.c")
	if err := os.WriteFile(cMainPath, []byte(mainC), 0o644); err != nil {
		t.Fatal(err)
	}
	cBin := filepath.Join(dir, "ref_dot")
	if out, err := exec.Command("gcc", "-O2", "-o", cBin, cMainPath).CombinedOutput(); err != nil {
		t.Fatalf("gcc compilation error: %v\n%s", err, out)
	}
	out, err := exec.Command(cBin).CombinedOutput()
	if err != nil {
		t.Fatalf("gcc execution error: %v", err)
	}
	gccVal, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil {
		t.Fatalf("strconv.ParseFloat: %v", err)
	}

	// Écriture de gen.go et main.go
	if err := os.WriteFile(filepath.Join(dir, "gen.go"), []byte(goCode), 0o644); err != nil {
		t.Fatal(err)
	}
	goTest := `package main

import (
	"fmt"
	"os"
)

func main() {
	a := []float32{1.5, 2.0, -3.5, 4.25, 0.5, -1.0, 2.5, 3.0}
	b := []float32{2.0, -1.5, 1.0, 2.0, 4.0, 3.0, -2.0, 1.5}
	var res float64
	Simsimd_dot_f32(a, b, 8, &res)
	fmt.Printf("%.10f\n", res)
	os.Exit(0)
}
`

	goMainPath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(goMainPath, []byte(goTest), 0o644); err != nil {
		t.Fatal(err)
	}
	goMod := "module simsimd_test\n\ngo 1.27\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}
	cmdGo := exec.Command("go", "run", "main.go", "gen.go")
	cmdGo.Dir = dir
	goOut, err := cmdGo.CombinedOutput()
	if err != nil {
		t.Fatalf("go run error: %v\n%s", err, goOut)
	}
	goVal, err := strconv.ParseFloat(strings.TrimSpace(string(goOut)), 64)
	if err != nil {
		t.Fatalf("strconv.ParseFloat go: %v", err)
	}

	if math.Abs(gccVal-goVal) > 1e-9 {
		t.Fatalf("Oracle mismatch SimSIMD Dot: GCC=%f Go=%f", gccVal, goVal)
	}
}

func testSimsimdL2sqVsGCC(t *testing.T) {
	cSource, err := filepath.Abs(filepath.Join("..", "spec", "c_sources", "testdata", "c_sources", "simsimd_l2sq_f32.c"))
	if err != nil {
		t.Fatal(err)
	}

	m, err := front.ParseFile(cSource)
	if err != nil {
		t.Fatalf("front.ParseFile: %v", err)
	}
	m, err = rules.ApplyAll(m)
	if err != nil {
		t.Fatalf("rules.ApplyAll: %v", err)
	}
	m.Name = "main"
	goCode, err := emit.Emit(m, emit.ProfileGo127)
	if err != nil {
		t.Fatalf("emit.Emit: %v", err)
	}

	dir := t.TempDir()
	mainC := fmt.Sprintf(`#include <stdio.h>
#include <stdint.h>
#include <stddef.h>
#include "%s"

int main(void) {
    float a[8] = {1.0f, 2.0f, 3.0f, 4.0f, 5.0f, 6.0f, 7.0f, 8.0f};
    float b[8] = {1.5f, 1.5f, 3.5f, 3.5f, 5.5f, 5.5f, 7.5f, 7.5f};
    double res = 0.0;
    simsimd_l2sq_f32(a, b, 8, &res);
    printf("%%.10f\n", res);
    return 0;
}
`, cSource)

	cMainPath := filepath.Join(dir, "main.c")
	if err := os.WriteFile(cMainPath, []byte(mainC), 0o644); err != nil {
		t.Fatal(err)
	}
	cBin := filepath.Join(dir, "ref_l2")
	if out, err := exec.Command("gcc", "-O2", "-o", cBin, cMainPath).CombinedOutput(); err != nil {
		t.Fatalf("gcc compilation error: %v\n%s", err, out)
	}
	out, err := exec.Command(cBin).CombinedOutput()
	if err != nil {
		t.Fatalf("gcc execution error: %v", err)
	}
	gccVal, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil {
		t.Fatalf("strconv.ParseFloat: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "gen.go"), []byte(goCode), 0o644); err != nil {
		t.Fatal(err)
	}
	goTest := `package main

import (
	"fmt"
	"os"
)

func main() {
	a := []float32{1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0}
	b := []float32{1.5, 1.5, 3.5, 3.5, 5.5, 5.5, 7.5, 7.5}
	var res float64
	Simsimd_l2sq_f32(a, b, 8, &res)
	fmt.Printf("%.10f\n", res)
	os.Exit(0)
}
`

	goMainPath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(goMainPath, []byte(goTest), 0o644); err != nil {
		t.Fatal(err)
	}
	goMod := "module simsimd_l2_test\n\ngo 1.27\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}
	cmdGo := exec.Command("go", "run", "main.go", "gen.go")
	cmdGo.Dir = dir
	goOut, err := cmdGo.CombinedOutput()
	if err != nil {
		t.Fatalf("go run error: %v\n%s", err, goOut)
	}
	goVal, err := strconv.ParseFloat(strings.TrimSpace(string(goOut)), 64)
	if err != nil {
		t.Fatalf("strconv.ParseFloat go: %v", err)
	}

	if math.Abs(gccVal-goVal) > 1e-9 {
		t.Fatalf("Oracle mismatch SimSIMD L2sq: GCC=%f Go=%f", gccVal, goVal)
	}
}

func testFastHadamardVsGCC(t *testing.T) {
	cSource, err := filepath.Abs(filepath.Join("..", "spec", "c_sources", "ann_lab", "fast_hadamard.c"))
	if err != nil {
		t.Fatal(err)
	}

	m, err := front.ParseFile(cSource)
	if err != nil {
		t.Fatalf("front.ParseFile: %v", err)
	}
	m, err = rules.ApplyAll(m)
	if err != nil {
		t.Fatalf("rules.ApplyAll: %v", err)
	}
	m.Name = "main"
	goCode, err := emit.Emit(m, emit.ProfileGo127)
	if err != nil {
		t.Fatalf("emit.Emit: %v", err)
	}

	dir := t.TempDir()
	rnd := rand.New(rand.NewSource(42))
	const count = 16
	var cInit strings.Builder
	for i := 0; i < count; i++ {
		val := rnd.Float32()*10.0 - 5.0
		cInit.WriteString(fmt.Sprintf("%ff, ", val))
	}
	initStr := strings.TrimSuffix(cInit.String(), ", ")

	mainC := fmt.Sprintf(`#include <stdio.h>
#include <stdint.h>
#include <stddef.h>
#include "%s"

int main(void) {
    float v[%d] = {%s};
    c2simd_fht_in_place_f32(v, %d);
    for (int i = 0; i < %d; i++) {
        printf("%%.6f ", v[i]);
    }
    printf("\n");
    return 0;
}
`, cSource, count, initStr, count, count)

	cMainPath := filepath.Join(dir, "main.c")
	if err := os.WriteFile(cMainPath, []byte(mainC), 0o644); err != nil {
		t.Fatal(err)
	}
	cBin := filepath.Join(dir, "ref_fht")
	if out, err := exec.Command("gcc", "-O2", "-o", cBin, cMainPath).CombinedOutput(); err != nil {
		t.Fatalf("gcc compilation error: %v\n%s", err, out)
	}
	out, err := exec.Command(cBin).CombinedOutput()
	if err != nil {
		t.Fatalf("gcc execution error: %v", err)
	}
	gccOutputs := strings.Fields(strings.TrimSpace(string(out)))

	// Conversion C init string to Go slice
	goInit := strings.ReplaceAll(initStr, "f", "")

	if err := os.WriteFile(filepath.Join(dir, "gen.go"), []byte(goCode), 0o644); err != nil {
		t.Fatal(err)
	}
	goTest := fmt.Sprintf(`package main

import (
	"fmt"
	"os"
)

func main() {
	v := []float32{%s}
	C2simd_fht_in_place_f32(v, %d)
	for _, val := range v {
		fmt.Printf("%%.6f ", val)
	}
	fmt.Println()
	os.Exit(0)
}
`, goInit, count)

	goMainPath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(goMainPath, []byte(goTest), 0o644); err != nil {
		t.Fatal(err)
	}
	goMod := "module hadamard_test\n\ngo 1.27\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}
	cmdGo := exec.Command("go", "run", "main.go", "gen.go")
	cmdGo.Dir = dir
	goOut, err := cmdGo.CombinedOutput()
	if err != nil {
		t.Fatalf("go run error: %v\n%s", err, goOut)
	}
	goOutputs := strings.Fields(strings.TrimSpace(string(goOut)))

	if len(gccOutputs) != len(goOutputs) || len(gccOutputs) != count {
		t.Fatalf("Hadamard output count mismatch: GCC=%d Go=%d", len(gccOutputs), len(goOutputs))
	}

	for i := 0; i < count; i++ {
		gv, _ := strconv.ParseFloat(gccOutputs[i], 64)
		gov, _ := strconv.ParseFloat(goOutputs[i], 64)
		if math.Abs(gv-gov) > 1e-4 {
			t.Fatalf("Hadamard mismatch at index %d: GCC=%f Go=%f", i, gv, gov)
		}
	}
}
