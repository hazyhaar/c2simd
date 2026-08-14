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

func TestBuiltinMemsetMemcpy(t *testing.T) {
	src := `
#include <stdint.h>
#include <string.h>
void zero16(uint8_t *p) { memset(p, 0, 16); }
void copy8(uint8_t *d, const uint8_t *s) { memcpy(d, s, 8); }
void fill4(uint8_t *p, uint8_t v) { memset(p, v, 4); }
`
	runEmitOracle(t, "membi", src, `
package membi
import "testing"
func TestMem(t *testing.T) {
	p := make([]byte, 16)
	for i := range p { p[i] = 1 }
	Zero16(p)
	for i := 0; i < 16; i++ {
		if p[i] != 0 { t.Fatalf("zero %d", i) }
	}
	s := []byte{9, 8, 7, 6, 5, 4, 3, 2}
	d := make([]byte, 8)
	Copy8(d, s)
	for i := 0; i < 8; i++ {
		if d[i] != s[i] { t.Fatalf("copy %d", i) }
	}
	Fill4(p, 0xab)
	for i := 0; i < 4; i++ {
		if p[i] != 0xab { t.Fatalf("fill %d", i) }
	}
}
`)
}

func TestBlake2b2DSigma(t *testing.T) {
	cpath, err := filepath.Abs(filepath.Join("..", "spec", "c_sources", "testdata", "c_sources", "blake2b_compress.c"))
	if err != nil {
		t.Fatal(err)
	}
	m, err := front.ParseFile(cpath)
	if err != nil {
		t.Fatal(err)
	}
	// must harvest 2D sigma
	found := false
	for _, g := range m.Globals {
		if g.Name == "blake2b_sigma" {
			found = true
			if g.Cols != 16 || g.Rows != 12 {
				t.Fatalf("sigma dims rows=%d cols=%d", g.Rows, g.Cols)
			}
			if g.InitCSV == "" {
				t.Fatal("empty csv")
			}
		}
	}
	if !found {
		t.Fatal("blake2b_sigma not harvested")
	}
	m, err = rules.ApplyAll(m)
	if err != nil {
		t.Fatal(err)
	}
	m.Name = "blake2d"
	src, err := emit.Emit(m, emit.ProfileGo127)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "b.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module blake2d\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	testGo := `package blake2d
import "testing"
func TestCompress(t *testing.T) {
	h0 := make([]uint64, 8)
	h1 := make([]uint64, 8)
	block0 := make([]byte, 128)
	block1 := make([]byte, 128)
	for i := range block1 { block1[i] = byte(i) }
	f := uint64(0xffffffffffffffff)
	Blake2b_compress_block(h0, block0, 0, 0, f, f)
	Blake2b_compress_block(h1, block1, 0, 0, f, f)
	nz := false
	for _, v := range h0 {
		if v != 0 { nz = true }
	}
	if !nz { t.Fatal("all zero h0") }
	diff := false
	for i := range h0 {
		if h0[i] != h1[i] { diff = true }
	}
	if !diff { t.Fatal("compress ignores block content") }
}
`
	if err := os.WriteFile(filepath.Join(dir, "b_test.go"), []byte(testGo), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "test", "-count=1", "-v", ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("blake2d: %v\n%s\n---\n%s", err, out, src[:min(800, len(src))])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
