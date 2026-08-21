package emit_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/emit"
	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/front"
	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/rules"
)

func TestForContinueExecution(t *testing.T) {
	src := `
int test_for_continue(void) {
	int c = 0;
	int i;
	for (i = 0; i < 3; i++) {
		if (i == 1) continue;
		c++;
	}
	return c;
}
`
	m, err := front.Parse(src, "test")
	if err != nil {
		t.Fatal(err)
	}
	m, err = rules.ApplyAll(m)
	if err != nil {
		t.Fatal(err)
	}
	out, err := emit.Emit(m, emit.ProfileGo127)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Emitted code:\n%s", out)

	dir := t.TempDir()
	mainGo := `package main

import (
	"fmt"
	"os"
)

` + strings.Replace(out, "package test", "", 1) + `

func main() {
	c := Test_for_continue()
	if c != 2 {
		fmt.Fprintf(os.Stderr, "want 2, got %d\n", c)
		os.Exit(1)
	}
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module testpkg\n\ngo 1.27\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "main.go")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	res, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("execution failed: %v, output: %s", err, string(res))
	}
}

func TestForPostIncrementFill(t *testing.T) {
	src := `
typedef struct { uint32_t rune; uint8_t fg; uint8_t bg; uint8_t flags; uint8_t width; } c2_cell_t;
int fill_count(c2_cell_t *cells, int n) {
	int i;
	int c;
	c = 0;
	for (i = 0; i < n; i++) {
		cells[i].rune = 32;
		c = c + 1;
	}
	return c;
}
void fill_range(c2_cell_t *cells, int i0, int i1) {
	int i;
	for (i = i0; i < i1; i++) {
		cells[i].rune = 32;
		cells[i].width = 1;
	}
}
`
	m, err := front.Parse(src, "test_forfill")
	if err != nil {
		t.Fatal(err)
	}
	m, err = rules.ApplyAll(m)
	if err != nil {
		t.Fatal(err)
	}
	out, err := emit.Emit(m, emit.ProfileGo127)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "_ = 0 + 1") || strings.Contains(out, "_ = 0+1") {
		t.Fatalf("increment DCE'd:\n%s", out)
	}
	t.Logf("emit:\n%s", out)
	rangeFn := out
	if i := strings.Index(out, "func Fill_range"); i >= 0 {
		rangeFn = out[i:]
	}
	if !strings.Contains(rangeFn, "++") && !strings.Contains(rangeFn, " += 1") && !strings.Contains(rangeFn, " = ") {
		t.Fatalf("Fill_range missing increment:\n%s", rangeFn)
	}
	if strings.Count(rangeFn, "for ") > 0 && !strings.Contains(rangeFn, "++") && !strings.Contains(rangeFn, "+=") {
		t.Fatalf("Fill_range loop has no ++/+= :\n%s", rangeFn)
	}
	dir := t.TempDir()
	mainGo := `package main
import ("fmt"; "os")
` + strings.Replace(out, "package test_forfill", "", 1) + `
func main() {
	cells := make([]C2_cell_t, 8)
	c := Fill_count(cells, 4)
	if c != 4 {
		fmt.Fprintf(os.Stderr, "fill_count want 4 got %d\n", c)
		os.Exit(1)
	}
	Fill_range(cells, 2, 6)
	if cells[2].Rune_ != 32 || cells[5].Rune_ != 32 {
		fmt.Fprintf(os.Stderr, "fill_range missed cells\n")
		os.Exit(1)
	}
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module testpkg\n\ngo 1.27\n"), 0644); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "run", "main.go")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	res, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("execution failed: %v\n%s\n--- emit ---\n%s", err, res, out)
	}
}

func TestBug3StructArrayVsScalar(t *testing.T) {
	src := `
typedef struct {
    uint32_t rune;
    uint8_t fg;
} c2_cell_t;

typedef struct {
    int cursor_x;
    int cursor_y;
} c2_vt_parser_t;

void c2_vt_put_cell(c2_vt_parser_t *p, c2_cell_t *cells, int idx, uint32_t r) {
    if (cells == 0) return;
    cells[idx].rune = r;
    p->cursor_x = p->cursor_x + 1;
}

void c2_vt_csi_test(c2_vt_parser_t *p, c2_cell_t *cells, int idx, uint8_t b) {
    if (cells != 0) {
        c2_vt_put_cell(p, cells, idx, (uint32_t)b);
    }
}
`
	m, err := front.Parse(src, "test_bug3")
	if err != nil {
		t.Fatal(err)
	}
	m, err = rules.ApplyAll(m)
	if err != nil {
		t.Fatal(err)
	}
	out, err := emit.Emit(m, emit.ProfileGo127)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Emitted code:\n%s", out)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "k.go"), []byte(out), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module testpkg\n\ngo 1.27\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "build", ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	res, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed: %v\nOutput:\n%s\nEmitted code:\n%s", err, string(res), out)
	}
}

func TestIntMinusNotPtrAlias(t *testing.T) {
	src := `
typedef struct { int rune; } c2_cell_t;
void c2_grid_scroll_up(c2_cell_t *cells, int width, int height, int stride, int nlines) {
    int y;
    int lim;
    int srcy;
    if (nlines < 1) {
        return;
    }
    lim = height - nlines;
    for (y = 0; y < lim; y++) {
        srcy = y + nlines;
        cells[y * stride].rune = cells[srcy * stride].rune;
    }
}
`
	m, err := front.Parse(src, "test_sub")
	if err != nil {
		t.Fatal(err)
	}
	m, err = rules.ApplyAll(m)
	if err != nil {
		t.Fatal(err)
	}
	out, err := emit.Emit(m, emit.ProfileGo127)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "&nlines") || strings.Contains(out, "nlines[") {
		t.Fatalf("int minus lowered as pointer:\n%s", out)
	}
	if !strings.Contains(out, "height") || !strings.Contains(out, "-") {
		t.Fatalf("expected integer height-nlines:\n%s", out)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "k.go"), []byte(out), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module testpkg\n\ngo 1.27\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "build", ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOTOOLCHAIN=go1.27.0")
	res, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed: %v\n%s\n%s", err, res, out)
	}
}
