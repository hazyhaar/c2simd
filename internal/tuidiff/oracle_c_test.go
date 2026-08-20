package c2tuidiff

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestC2DiffGridVsGCC(t *testing.T) {
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("gcc not available")
	}
	src, err := filepath.Abs(filepath.Join("..", "..", "sources", "c2tuidiff.c"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	mainC := `#include <stdio.h>
#include "` + src + `"
int main(void) {
  c2_cell_t f[4] = {{65,1,0,0,1},{66,1,0,0,1},{67,1,0,0,1},{68,1,0,0,1}};
  c2_cell_t b[4] = {{65,1,0,0,1},{88,1,0,0,1},{67,1,0,0,1},{68,1,0,0,1}};
  c2_span_t sp[4];
  int n = c2_diff_grid_scalar(f, b, 4, 4, 4, sp, 4);
  printf("%d %d %d %d\n", n, sp[0].x, sp[0].y, sp[0].length);
  return 0;
}
`
	mainPath := filepath.Join(dir, "main.c")
	if err := os.WriteFile(mainPath, []byte(mainC), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "ref")
	cmd := exec.Command("gcc", "-O2", "-o", bin, mainPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("gcc: %v\n%s", err, out)
	}
	out, err := exec.Command(bin).CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(string(out))
	fields := strings.Fields(got)
	if len(fields) != 4 {
		t.Fatalf("gcc out %q", got)
	}
	wantN, _ := strconv.Atoi(fields[0])
	cf := []C2_cell_t{
		{Rune_: 65, Fg: 1, Width: 1},
		{Rune_: 66, Fg: 1, Width: 1},
		{Rune_: 67, Fg: 1, Width: 1},
		{Rune_: 68, Fg: 1, Width: 1},
	}
	cb := []C2_cell_t{
		{Rune_: 65, Fg: 1, Width: 1},
		{Rune_: 88, Fg: 1, Width: 1},
		{Rune_: 67, Fg: 1, Width: 1},
		{Rune_: 68, Fg: 1, Width: 1},
	}
	sp := make([]C2_span_t, 4)
	n := C2_diff_grid_scalar(cf, cb, 4, 4, 4, sp, 4)
	if n != wantN || sp[0].X != 1 || sp[0].Y != 0 || sp[0].Length != 1 {
		t.Fatalf("gen n=%d span=%+v gcc %q", n, sp[0], got)
	}
}

func TestC2DiffGridVsGCCStride(t *testing.T) {
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("gcc not available")
	}
	src, err := filepath.Abs(filepath.Join("..", "..", "sources", "c2tuidiff.c"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	mainC := `#include <stdio.h>
#include "` + src + `"
int main(void) {
  c2_cell_t f[8] = {0};
  c2_cell_t b[8] = {0};
  int i;
  for (i = 0; i < 8; i++) { f[i].width = 1; b[i].width = 1; f[i].rune = 65; b[i].rune = 65; }
  b[1].rune = 66;
  c2_span_t sp[4];
  int n = c2_diff_grid_scalar(f, b, 8, 4, 3, sp, 4);
  printf("%d %d %d %d\n", n, sp[0].x, sp[0].y, sp[0].length);
  return 0;
}
`
	mainPath := filepath.Join(dir, "main.c")
	if err := os.WriteFile(mainPath, []byte(mainC), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "ref")
	if out, err := exec.Command("gcc", "-O2", "-o", bin, mainPath).CombinedOutput(); err != nil {
		t.Fatalf("gcc: %v\n%s", err, out)
	}
	out, err := exec.Command(bin).CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	cf := make([]C2_cell_t, 8)
	cb := make([]C2_cell_t, 8)
	for i := 0; i < 8; i++ {
		cf[i] = C2_cell_t{Rune_: 65, Width: 1}
		cb[i] = C2_cell_t{Rune_: 65, Width: 1}
	}
	cb[1].Rune_ = 66
	sp := make([]C2_span_t, 4)
	n := C2_diff_grid_scalar(cf, cb, 8, 4, 3, sp, 4)
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) != 4 {
		t.Fatalf("gcc out %q", out)
	}
	wantN, _ := strconv.Atoi(fields[0])
	if n != wantN || n != 1 || sp[0].X != 1 || sp[0].Y != 0 || sp[0].Length != 1 {
		t.Fatalf("gen n=%d span=%+v gcc %q", n, sp[0], out)
	}
}

func TestC2ChunkDirty4VsGCC(t *testing.T) {
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("gcc not available")
	}
	src, err := filepath.Abs(filepath.Join("..", "..", "sources", "c2tuidiff.c"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	mainC := `#include <stdio.h>
#include <stdint.h>
#include "` + src + `"
int main(void) {
  uint64_t f[4] = {1, 2, 3, 4};
  uint64_t b[4] = {1, 9, 3, 8};
  printf("%d\n", c2_chunk_dirty4(f, b));
  return 0;
}
`
	mainPath := filepath.Join(dir, "main.c")
	if err := os.WriteFile(mainPath, []byte(mainC), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "ref")
	if out, err := exec.Command("gcc", "-O2", "-o", bin, mainPath).CombinedOutput(); err != nil {
		t.Fatalf("gcc: %v\n%s", err, out)
	}
	out, err := exec.Command(bin).CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	want, _ := strconv.Atoi(strings.TrimSpace(string(out)))
	got := C2_chunk_dirty4([]uint64{1, 2, 3, 4}, []uint64{1, 9, 3, 8})
	if got != want || got != 10 {
		t.Fatalf("gen %d gcc %d", got, want)
	}
}
