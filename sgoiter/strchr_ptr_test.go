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

func TestStrchrSemantic(t *testing.T) {
	src := `
#include <stdint.h>
#include <stddef.h>
uint64_t has_char(const uint8_t *s, uint64_t n, uint8_t c) {
    uint64_t i;
    for (i = 0; i < n; i++) {
        if (strchr((const char*)s, s[i]) != 0) {
            /* always true if s non-empty — exercise strchr on haystack */
        }
    }
    if (strchr((const char*)s, c) != 0) return 1;
    return 0;
}
uint64_t find_offset(const uint8_t *s, uint64_t n, uint8_t c) {
    const uint8_t *p = (const uint8_t*)strchr((const char*)s, c);
    if (p == 0) return 0xffffffffffffffffULL;
    return (uint64_t)(p - s);
}
`
	// simpler pure strchr membership + offset via index loop (no ptr sub result)
	src = `
#include <stdint.h>
#include <stddef.h>
/* strchr builtin under test */
uint64_t contains(const uint8_t *hay, uint8_t c) {
    if (strchr((const char*)hay, c) != 0) return 1;
    return 0;
}
uint64_t strspn_like(const uint8_t *s, uint64_t n, const uint8_t *accept) {
    uint64_t i;
    for (i = 0; i < n; i++) {
        if (strchr((const char*)accept, s[i]) == 0) return i;
    }
    return n;
}
`
	runEmitOracle(t, "strchrkat", src, `
package strchrkat
import "testing"
func TestStrchr(t *testing.T) {
	hay := []byte("hello\x00")
	if Contains(hay, 'e') != 1 { t.Fatalf("e in hello") }
	if Contains(hay, 'z') != 0 { t.Fatalf("z not in hello") }
	if Contains(hay, 0) != 1 { t.Fatalf("NUL in hay") }
	accept := []byte("ab")
	s := []byte("aaXbb")
	if g := Strspn_like(s, 5, accept); g != 2 {
		t.Fatalf("strspn_like got %d want 2 (stop at X)", g)
	}
}
`)
}

func TestPtrMinusAssign(t *testing.T) {
	src := `
#include <stdint.h>
#include <stddef.h>
/* m += n; m -= k; then read first byte at cursor */
uint8_t peek_after(const uint8_t *m, uint64_t n, uint64_t back) {
    m += n;
    m -= back;
    return m[0];
}
void copy_window(uint8_t *dst, const uint8_t *m, uint64_t n, uint64_t win) {
    uint64_t i;
    m += n;
    m -= win;
    for (i = 0; i < win; i++) dst[i] = m[i];
}
`
	runEmitOracle(t, "ptrmkat", src, `
package ptrmkat
import (
	"bytes"
	"testing"
)
func TestPtrMinus(t *testing.T) {
	buf := []byte("ABCDEFGHIJ")
	// m += 7 → at 'H'; m -= 3 → at 'E'
	if g := Peek_after(buf, 7, 3); g != 'E' {
		t.Fatalf("peek got %c want E", g)
	}
	dst := make([]byte, 4)
	// m += 10; m -= 4 → last 4 bytes "GHIJ"
	Copy_window(dst, buf, 10, 4)
	if !bytes.Equal(dst, []byte("GHIJ")) {
		t.Fatalf("window %q", dst)
	}
}
`)
}

func runEmitOracle(t *testing.T, pkg, csrc, testGo string) {
	t.Helper()
	dir := t.TempDir()
	cpath := filepath.Join(dir, "input.c.txt")
	if err := os.WriteFile(cpath, []byte(csrc), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := front.Parse(csrc, pkg)
	_ = cpath
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	m, err = rules.ApplyAll(m)
	if err != nil {
		t.Fatalf("rules: %v", err)
	}
	m.Name = pkg
	out, err := emit.Emit(m, emit.ProfileGo127)
	if err != nil {
		t.Fatalf("emit: %v\n", err)
	}
	if err := os.WriteFile(filepath.Join(dir, pkg+".go"), []byte(out), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "oracle_test.go"), []byte(testGo), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module "+pkg+"\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "test", "-count=1", "-v", ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	o, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test: %v\n%s\n--- emitted ---\n%s", err, o, out)
	}
	t.Logf("%s", o)
}
