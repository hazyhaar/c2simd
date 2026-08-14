package emit_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestKernelOverridesCompileAndShape: each override kernel emits and go-builds.
func TestKernelOverridesCompileAndShape(t *testing.T) {
	root := "/devhoros/c2simd"
	sgo := filepath.Join(root, "bin/sgoiter")
	if _, err := os.Stat(sgo); err != nil {
		cmd := exec.Command("go", "build", "-o", sgo, "./sgoiter/cmd/sgoiter")
		cmd.Dir = root
		cmd.Env = append(os.Environ(), "GOWORK=off")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("build sgoiter: %v\n%s", err, out)
		}
	}
	cs := filepath.Join(root, "spec/c_sources/testdata/c_sources")
	cases := []struct {
		cfile string
		want  []string
	}{
		{"fnv1a_64.c", []string{"0x100000001b3", "i+8 <= n", "b := "}},
		{"blake2b_compress.c", []string{"blake2b_sigma", "RotateLeft64"}},
		{"fast_xor.c", []string{"SliceData", "i+16 <= n"}},
		{"murmur3_x86_32.c", []string{"for j <", "RotateLeft32"}},
		{"base64_simd.c", []string{"b64_table", "j : j+4"}},
		{"siphash24.c", []string{"0x646f72616d617461", "btail"}},
		{"strlenspn_lab.c", []string{"ok := c == 'h'", "return uint64(i)"}},
	}
	dir := t.TempDir()
	for _, tc := range cases {
		t.Run(tc.cfile, func(t *testing.T) {
			out := filepath.Join(dir, tc.cfile+".go")
			cmd := exec.Command(sgo, "-in", filepath.Join(cs, tc.cfile), "-out", out, "-mode", "kernel")
			if o, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("sgoiter: %v\n%s", err, o)
			}
			raw, _ := os.ReadFile(out)
			s := string(raw)
			for _, w := range tc.want {
				if !strings.Contains(s, w) {
					t.Errorf("missing %q in emit", w)
				}
			}
			// compile
			mod := filepath.Join(dir, "m_"+tc.cfile)
			_ = os.MkdirAll(mod, 0o755)
			_ = os.WriteFile(filepath.Join(mod, "go.mod"), []byte("module m\n\ngo 1.22\n"), 0o644)
			body := strings.Replace(s, "package "+packageLine(s), "package m", 1)
			_ = os.WriteFile(filepath.Join(mod, "k.go"), []byte(body), 0o644)
			bcmd := exec.Command("go", "build", "-o", os.DevNull, ".")
			bcmd.Dir = mod
			bcmd.Env = append(os.Environ(), "GOWORK=off")
			if o, err := bcmd.CombinedOutput(); err != nil {
				t.Fatalf("go build: %v\n%s\n%s", err, o, body[:min(500, len(body))])
			}
		})
	}
}

func packageLine(s string) string {
	for _, ln := range strings.Split(s, "\n") {
		if strings.HasPrefix(ln, "package ") {
			return strings.TrimSpace(strings.TrimPrefix(ln, "package "))
		}
	}
	return "main"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
