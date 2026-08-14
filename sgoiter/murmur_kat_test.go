package sgoiter_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/emit"
	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/front"
	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/rules"
)

// Oracle Murmur3_x86_32 vs gcc -O0 on the same C source (testdata/c/murmur3_lab.c).
func TestMurmur3OracleGCC(t *testing.T) {
	root := filepath.Join("testdata", "c", "murmur3_lab.c")
	cpath, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	// transpile
	m, err := front.ParseFile(cpath)
	if err != nil {
		t.Fatal(err)
	}
	m, err = rules.ApplyAll(m)
	if err != nil {
		t.Fatal(err)
	}
	// force package name
	m.Name = "murmurkat"
	src, err := emit.Emit(m, emit.ProfileGo127)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	gopath := filepath.Join(dir, "murmurkat.go")
	if err := os.WriteFile(gopath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	testGo := `package murmurkat

import "testing"

func TestOracle(t *testing.T) {
	type vec struct {
		s    string
		seed uint32
		want uint32
	}
	// filled by TestMurmur3OracleGCC parent via //go:generate — hardcoded from gcc -O0
	vecs := []vec{
		{"", 0, 0},
		{"hello", 0, 613153351},
		{"hello", 42, 3806057185},
		{"The quick brown fox jumps over the lazy dog", 0, 776992547},
	}
	for _, v := range vecs {
		var b []byte
		if v.s != "" {
			b = []byte(v.s)
		}
		got := Murmur3_x86_32(b, uint64(len(b)), v.seed)
		if got != v.want {
			t.Errorf("%q seed=%d: got %d want %d", v.s, v.seed, got, v.want)
		}
	}
	bin := make([]byte, 16)
	for i := range bin {
		bin[i] = byte(i)
	}
	if g := Murmur3_x86_32(bin, 16, 42); g != 3660047595 {
		t.Errorf("bin16: got %d want 3660047595", g)
	}
}
`
	if err := os.WriteFile(filepath.Join(dir, "oracle_test.go"), []byte(testGo), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module murmurkat\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "test", "-count=1", ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test oracle: %v\n%s\n--- emitted ---\n%s", err, out, src)
	}
}

// Cross-check vectors still match live gcc when available.
func TestMurmur3GCCLive(t *testing.T) {
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("gcc not available")
	}
	dir := t.TempDir()
	src := filepath.Join("testdata", "c", "murmur3_lab.c")
	abs, _ := filepath.Abs(src)
	mainC := `
#include <stdint.h>
#include <stdio.h>
#include <string.h>
#include "` + abs + `"
int main(void) {
  printf("%u\n", murmur3_x86_32((const uint8_t*)"hello", 5, 0));
  return 0;
}
`
	mainPath := filepath.Join(dir, "main.c")
	if err := os.WriteFile(mainPath, []byte(mainC), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "ref")
	cmd := exec.Command("gcc", "-O0", "-o", bin, mainPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("gcc: %v\n%s", err, out)
	}
	out, err := exec.Command(bin).CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "613153351\n" {
		t.Fatalf("gcc live got %q", out)
	}
}
