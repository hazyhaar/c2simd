package sgoiter_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/emit"
	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/front"
	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/rules"
)

// The tribench harness exercises only the entry points it knows about. A helper
// that is harvested but never called can be silently wrong — ld32 accumulated in
// uint8 and truncated every shift while the bench stayed green. These tests call
// the helpers directly and compare against the C sources they came from.
func TestTweetnaclHelpersAgainstC(t *testing.T) {
	if _, err := exec.LookPath("cc"); err != nil {
		t.Skip("no C compiler")
	}
	cpath, err := filepath.Abs(filepath.Join("..", "spec", "c_sources", "testdata", "c_sources", "tweetnacl_dogfood.c"))
	if err != nil {
		t.Fatal(err)
	}
	m, err := front.ParseFile(cpath)
	if err != nil {
		t.Fatal(err)
	}
	if m, err = rules.ApplyAll(m); err != nil {
		t.Fatal(err)
	}
	m.Name = "tn"
	emit.FillStubs(m)
	src, err := emit.Emit(m, emit.ProfileGo127)
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "tn.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module tn\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// four bytes, little-endian, with the high byte set so a narrow accumulator
	// cannot pass by accident
	inputs := [][4]byte{
		{0x00, 0x00, 0x00, 0x00},
		{0x01, 0x02, 0x03, 0x04},
		{0xff, 0xff, 0xff, 0xff},
		{0x78, 0x56, 0x34, 0x12},
		{0x00, 0x00, 0x00, 0x80},
	}

	var goTest strings.Builder
	goTest.WriteString("package tn\n\nimport \"testing\"\n\nfunc TestLd32(t *testing.T) {\n")
	for i, in := range inputs {
		fmt.Fprintf(&goTest, "\tif got := Ld32([]byte{%d,%d,%d,%d}); got != uint64(want%d) {\n\t\tt.Errorf(\"ld32 %%d: got %%d want %%d\", %d, got, want%d)\n\t}\n",
			in[0], in[1], in[2], in[3], i, i, i)
	}
	goTest.WriteString("}\n")

	// the C oracle supplies the expected values
	var cprog strings.Builder
	cprog.WriteString("#include <stdio.h>\n#include <stdint.h>\ntypedef uint8_t u8;\ntypedef uint32_t u32;\n")
	cprog.WriteString("static u32 ld32(const u8 *x){u32 u=x[3];u=(u<<8)|x[2];u=(u<<8)|x[1];return (u<<8)|x[0];}\n")
	cprog.WriteString("int main(void){\n")
	for _, in := range inputs {
		fmt.Fprintf(&cprog, "    { u8 b[4]={%d,%d,%d,%d}; printf(\"%%u\\n\", ld32(b)); }\n", in[0], in[1], in[2], in[3])
	}
	cprog.WriteString("    return 0;\n}\n")

	// the C oracle lives outside the Go module dir: a .c file next to the
	// package makes `go test` refuse to build without cgo
	coracle := filepath.Join(dir, "coracle")
	if err := os.MkdirAll(coracle, 0o755); err != nil {
		t.Fatal(err)
	}
	csrc := filepath.Join(coracle, "oracle.c")
	if err := os.WriteFile(csrc, []byte(cprog.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	cbin := filepath.Join(coracle, "oracle")
	if out, err := exec.Command("cc", "-O2", "-o", cbin, csrc).CombinedOutput(); err != nil {
		t.Fatalf("cc: %v\n%s", err, out)
	}
	out, err := exec.Command(cbin).Output()
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	wants := strings.Fields(strings.TrimSpace(string(out)))
	if len(wants) != len(inputs) {
		t.Fatalf("oracle printed %d values for %d inputs", len(wants), len(inputs))
	}

	var consts strings.Builder
	consts.WriteString("package tn\n\n")
	for i, w := range wants {
		fmt.Fprintf(&consts, "const want%d = %s\n", i, w)
	}
	if err := os.WriteFile(filepath.Join(dir, "want.go"), []byte(consts.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tn_test.go"), []byte(goTest.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("go", "test", "-count=1", ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("emitted Ld32 disagrees with the C ld32:\n%s", out)
	}
	t.Logf("Ld32 matches the C oracle on %d inputs", len(inputs))
}
