package sgoiter_test

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/emit"
	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/front"
	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/rules"
)

// emitKernel runs the full front → rules → emit pipeline on a kernel source.
func emitKernel(t *testing.T, file string) string {
	t.Helper()
	cpath, err := filepath.Abs(filepath.Join("..", "spec", "c_sources", "testdata", "c_sources", file))
	if err != nil {
		t.Fatal(err)
	}
	m, err := front.ParseFile(cpath)
	if err != nil {
		t.Fatalf("parse %s: %v", file, err)
	}
	m, err = rules.ApplyAll(m)
	if err != nil {
		t.Fatalf("rules %s: %v", file, err)
	}
	m.Name = "det"
	src, err := emit.Emit(m, emit.ProfileGo127)
	if err != nil {
		t.Fatalf("emit %s: %v", file, err)
	}
	return src
}

// Same input must yield byte-identical output. Map iteration order used to leak
// into the emitted file (hoisted var block, single-use fold pick order).
func TestEmitIsDeterministic(t *testing.T) {
	for _, file := range []string{
		"blake2b_compress.c",
		"siphash24.c",
		"chacha20_qr.c",
		"md5_transform.c",
	} {
		first := emitKernel(t, file)
		for i := 2; i <= 5; i++ {
			if got := emitKernel(t, file); got != first {
				t.Fatalf("%s: emit run %d differs from run 1", file, i)
			}
		}
	}
}

// Q10 — index conversions. Pre-unroll target was <20; after const-trip unroll
// of 12 rounds the sigma-byte→index casts scale with G count (~384). Guard
// the pre-index-only blowup (thousands) without blocking the unroll path.
func TestBlakeIndexConversionsUnderTarget(t *testing.T) {
	src := emitKernel(t, "blake2b_compress.c")
	if n := strings.Count(src, "int("); n >= 500 {
		t.Errorf("blake2b emits %d conversions, want fewer than 500", n)
	}
}

var manualShiftPat = regexp.MustCompile(`<< uint8\(|>> uint8\(`)

// blake2b rotations must reach bits.RotateLeft64, not stay as shift/xor pairs.
// The single-use fold budget used to starve on this kernel, splitting each
// rotation across a hoisted temp that foldRotateExprs could no longer match.
func TestBlakeRotationsAreFolded(t *testing.T) {
	src := emitKernel(t, "blake2b_compress.c")
	rot := strings.Count(src, "bits.RotateLeft64")
	if rot < 24 {
		t.Errorf("bits.RotateLeft64 = %d, want >= 24 (8 G-functions x 4 rotations)", rot)
	}
	if n := len(manualShiftPat.FindAllString(src, -1)); n != 0 {
		t.Errorf("manual shift sites = %d, want 0 — a rotation stayed unfolded", n)
	}
}
