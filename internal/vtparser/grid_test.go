package c2vtparser

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestC2GridScrollVsGCC(t *testing.T) {
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("gcc not available")
	}
	hdr, err := filepath.Abs(filepath.Join("..", "..", "sources", "c2_grid.h"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	mainC := `#include <stdio.h>
#include "` + hdr + `"
int main(void) {
  c2_cell_t c[6];
  int i;
  for (i = 0; i < 6; i++) {
    c[i].rune = 65 + i;
    c[i].fg = 0; c[i].bg = 0; c[i].flags = 0; c[i].width = 1;
  }
  c2_grid_scroll_up(c, 2, 3, 2, 1);
  printf("%u %u %u %u %u %u\n", c[0].rune, c[1].rune, c[2].rune, c[3].rune, c[4].rune, c[5].rune);
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
	cells := make([]C2_cell_t, 6)
	for i := 0; i < 6; i++ {
		cells[i] = C2_cell_t{Rune_: uint32(65 + i), Width: 1}
	}
	C2_grid_scroll_up(cells, 2, 3, 2, 1)
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) != 6 {
		t.Fatalf("gcc out %q", out)
	}
	for i, f := range fields {
		want, _ := strconv.Atoi(f)
		if int(cells[i].Rune_) != want {
			t.Fatalf("cell %d gen=%d gcc=%d out=%q", i, cells[i].Rune_, want, out)
		}
	}
	if cells[0].Rune_ != 67 || cells[4].Rune_ != 32 {
		t.Fatalf("scroll: %+v", cells)
	}
}

func TestC2GridClear(t *testing.T) {
	cells := make([]C2_cell_t, 4)
	for i := range cells {
		cells[i].Rune_ = 88
		cells[i].Width = 2
	}
	C2_grid_clear(cells, 4, 7, 0)
	for i, c := range cells {
		if c.Rune_ != 32 || c.Fg != 7 || c.Width != 1 {
			t.Fatalf("cell %d %+v", i, c)
		}
	}
}
