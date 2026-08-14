package sgoiter_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/emit"
	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/front"
	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/rules"
)

// Runtime KATs for simple dogfood kernels (build-green ≠ bit-correct).
func TestKernelRuntimeKATs(t *testing.T) {
	srcRoot := filepath.Join("..", "spec", "c_sources", "testdata", "c_sources")
	kernels := []struct {
		file string
		name string
	}{
		{"fnv1a_64.c", "fnv1a"},
		{"crc32_ieee.c", "crc32"},
		{"fast_xor.c", "fast_xor"},
		{"siphash24.c", "siphash"},
	}
	dir := t.TempDir()
	var parts []string
	for _, k := range kernels {
		cpath, err := filepath.Abs(filepath.Join(srcRoot, k.file))
		if err != nil {
			t.Fatal(err)
		}
		m, err := front.ParseFile(cpath)
		if err != nil {
			t.Fatalf("parse %s: %v", k.file, err)
		}
		m, err = rules.ApplyAll(m)
		if err != nil {
			t.Fatalf("rules %s: %v", k.file, err)
		}
		m.Name = "kerkat"
		src, err := emit.Emit(m, emit.ProfileGo127)
		if err != nil {
			t.Fatalf("emit %s: %v", k.file, err)
		}
		// drop package line; keep body
		lines := splitPkg(src)
		parts = append(parts, lines)
		_ = os.WriteFile(filepath.Join(dir, k.name+"_emit.go.txt"), []byte(src), 0o644)
	}
	combined := "// Code generated; DO NOT EDIT.\npackage kerkat\n\nimport (\n\t\"encoding/binary\"\n\t\"math/bits\"\n)\n\n"
	// dedupe binary import — emit already has it per file; strip imports from parts
	for _, p := range parts {
		combined += stripImports(p) + "\n"
	}
	if err := os.WriteFile(filepath.Join(dir, "kernels.go"), []byte(combined), 0o644); err != nil {
		t.Fatal(err)
	}
	testGo := `package kerkat

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestFnv1a64(t *testing.T) {
	type v struct{ s string; want uint64 }
	// python/reference FNV-1a 64
	vecs := []v{
		{"", 0xcbf29ce484222325},
		{"abc", 0xe71fa2190541574b},
		{"0123456789abcdef", 0x2e373913e5ad677d},
		{"0123456789abcdefX", 0x353d21cf45a643df},
	}
	for _, tc := range vecs {
		b := []byte(tc.s)
		got := Fnv1a_64(b, uint64(len(b)))
		if got != tc.want {
			t.Errorf("fnv %q: got %#x want %#x", tc.s, got, tc.want)
		}
	}
}

func TestCrc32Ieee(t *testing.T) {
	type v struct{ s string; want uint32 }
	// binascii.crc32
	vecs := []v{
		{"", 0},
		{"abc", 0x352441c2},
		{"0123456789abcdef", 0x68c4f033},
		{"0123456789abcdefX", 0x080a93ad},
	}
	for _, tc := range vecs {
		b := []byte(tc.s)
		got := Crc32_ieee(b, uint64(len(b)))
		if got != tc.want {
			t.Errorf("crc %q: got %#x want %#x", tc.s, got, tc.want)
		}
	}
}

func TestFastXor(t *testing.T) {
	a := []byte("0123456789abcdefXY") // 18 bytes — hits word + tail loops
	b := make([]byte, len(a))
	for i := range a {
		b[i] = a[i] ^ 0x5a
	}
	dst := make([]byte, len(a))
	Fast_xor_bytes(dst, a, b, uint64(len(a)))
	want := bytes.Repeat([]byte{0x5a}, len(a))
	if !bytes.Equal(dst, want) {
		t.Fatalf("xor got %x want %x", dst, want)
	}
	// pure tail (<8)
	a2 := []byte{1, 2, 3}
	b2 := []byte{1, 0, 3}
	d2 := make([]byte, 3)
	Fast_xor_bytes(d2, a2, b2, 3)
	if d2[0] != 0 || d2[1] != 2 || d2[2] != 0 {
		t.Fatalf("xor tail %v", d2)
	}
}

func TestSiphash24(t *testing.T) {
	k := make([]byte, 16)
	for i := range k {
		k[i] = byte(i)
	}
	type v struct{ s string; want uint64 }
	// python siphash-2-4 reference
	vecs := []v{
		{"", 0xd96f54279d770740},
		{"abc", 0x8ce1ceee54d53cd9},
		{"0123456789abcdef", 0xfb9abf18e1be1837},
		{"0123456789abcdefX", 0x2400bfee621cba5b},
	}
	for _, tc := range vecs {
		in := []byte(tc.s)
		got := Siphash24(in, uint64(len(in)), k)
		if got != tc.want {
			t.Errorf("sip %q: got %#x want %#x", tc.s, got, tc.want)
		}
	}
	_ = binary.LittleEndian // keep import if kernels strip it
}
`
	if err := os.WriteFile(filepath.Join(dir, "kat_test.go"), []byte(testGo), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module kerkat\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "test", "-count=1", "-v", ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("kernel KAT: %v\n%s", err, out)
	}
	t.Logf("%s", out)
}

func splitPkg(src string) string { return src }

func stripImports(src string) string {
	var b strings.Builder
	inImport := false
	for _, l := range strings.Split(src, "\n") {
		trim := strings.TrimSpace(l)
		if strings.HasPrefix(l, "package ") {
			continue
		}
		if strings.Contains(l, "Code generated") || strings.Contains(l, "DO NOT EDIT") {
			continue
		}
		if strings.HasPrefix(trim, "import ") {
			if strings.HasPrefix(trim, "import (") {
				inImport = true
				continue
			}
			// single-line import "..."
			continue
		}
		if inImport {
			if trim == ")" {
				inImport = false
			}
			continue
		}
		b.WriteString(l)
		b.WriteByte('\n')
	}
	return b.String()
}
