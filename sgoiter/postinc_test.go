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

func TestPostIncIndexAndIfReeval(t *testing.T) {
	src := `
#include <stdint.h>
#include <stddef.h>
void fill_postinc(uint8_t *dst) {
    size_t j = 0;
    dst[j++] = 10;
    dst[j++] = 20;
    dst[j++] = 30;
}
size_t encode_tail_like(const uint8_t *src, size_t len, uint8_t *dst) {
    size_t i = 0, j = 0;
    if (i < len) {
        uint32_t t = (uint32_t)src[i] << 16;
        if (i + 1 < len) t |= (uint32_t)src[i+1] << 8;
        dst[j++] = (uint8_t)((t >> 18) & 63);
        dst[j++] = (uint8_t)((t >> 12) & 63);
        if (i + 1 < len) {
            dst[j++] = (uint8_t)((t >> 6) & 63);
        } else {
            dst[j++] = 61;
        }
        dst[j++] = 61;
    }
    return j;
}
`
	m, err := front.Parse(src, "postinc")
	if err != nil {
		t.Fatal(err)
	}
	m, err = rules.ApplyAll(m)
	if err != nil {
		t.Fatal(err)
	}
	m.Name = "postinc"
	out, err := emit.Emit(m, emit.ProfileGo127)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "p.go"), []byte(out), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "p_test.go"), []byte(`package postinc
import "testing"
func TestFill(t *testing.T) {
	d := make([]byte, 8)
	Fill_postinc(d)
	if d[0]!=10 || d[1]!=20 || d[2]!=30 {
		t.Fatalf("got %v", d[:3])
	}
}
func TestTail(t *testing.T) {
	// 1 byte remainder path → two data nibbles + two pads
	src := []byte{0x00}
	dst := make([]byte, 8)
	n := Encode_tail_like(src, 1, dst)
	if n != 4 || dst[2] != 61 || dst[3] != 61 {
		t.Fatalf("n=%d dst=%v", n, dst[:n])
	}
	// 2 byte remainder → three data + one pad
	src = []byte{0x00, 0x00}
	n = Encode_tail_like(src, 2, dst)
	if n != 4 || dst[3] != 61 || dst[2] == 61 {
		t.Fatalf("2byte n=%d dst=%v", n, dst[:n])
	}
}
`), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module postinc\n\ngo 1.22\n"), 0o644)
	cmd := exec.Command("go", "test", "-count=1", ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	o, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v\n%s\n---\n%s", err, o, out)
	}
}
