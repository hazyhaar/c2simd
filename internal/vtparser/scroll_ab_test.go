package c2vtparser

import (
	"strings"
	"testing"
)

var sinkRune rune

func runtimeKeep(r rune) { sinkRune = r }

func scrollTrace(lines, width int) []byte {
	var b strings.Builder
	row := strings.Repeat("a", width) + "\n"
	for i := 0; i < lines; i++ {
		b.WriteString(row)
	}
	return []byte(b.String())
}

func TestScrollCountFeed(t *testing.T) {
	const w, h, lines = 80, 24, 400
	trace := scrollTrace(lines, w)
	var ncall, nlines, cells int
	scrollHook = func(n, c int) {
		ncall++
		nlines += n
		cells += c
	}
	t.Cleanup(func() { scrollHook = nil })
	g := &CursorGrid{}
	g.Reset(w, h)
	p := &Parser{}
	p.Reset(g)
	p.Feed(trace)
	t.Logf("Feed %d lines %dx%d: scrollCalls=%d scrollLines=%d cellsTouched=%d bytes=%d",
		lines, w, h, ncall, nlines, cells, len(trace))
	if ncall < lines-h {
		t.Fatalf("expected ~%d scrolls, got %d", lines-h, ncall)
	}
	if ncall > 0 && nlines/ncall != 1 {
		t.Fatalf("mean n per call=%d want 1", nlines/ncall)
	}
}

func fillCells(n int) []Cell {
	s := make([]Cell, n)
	for i := range s {
		s[i] = Cell{Rune: rune('A' + i%26), Width: 1}
	}
	return s
}

func goScroll1(cells []Cell, w, h int) {
	copy(cells[:w*(h-1)], cells[w:])
	clear(cells[w*(h-1):])
}

func goScrollN(cells []Cell, w, h, n int) {
	if n >= h {
		clear(cells)
		return
	}
	copy(cells[:w*(h-n)], cells[w*n:])
	clear(cells[w*(h-n):])
}

func BenchmarkScrollC_n1_80x24(b *testing.B) {
	cells := make([]C2_cell_t, 80*24)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		C2_grid_scroll_up(cells, 80, 24, 80, 1)
	}
}

func BenchmarkScrollGo_n1_80x24(b *testing.B) {
	cells := fillCells(80 * 24)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		goScroll1(cells, 80, 24)
	}
}

func BenchmarkScrollC_n23_80x24(b *testing.B) {
	cells := make([]C2_cell_t, 80*24)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		C2_grid_scroll_up(cells, 80, 24, 80, 23)
	}
}

func BenchmarkScrollGo_n23_80x24(b *testing.B) {
	cells := fillCells(80 * 24)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		goScrollN(cells, 80, 24, 23)
	}
	runtimeKeep(cells[0].Rune)
}

func BenchmarkScrollC_n1x376(b *testing.B) {
	cells := make([]C2_cell_t, 80*24)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for k := 0; k < 376; k++ {
			C2_grid_scroll_up(cells, 80, 24, 80, 1)
		}
	}
}

func BenchmarkScrollGo_n1x376(b *testing.B) {
	cells := fillCells(80 * 24)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for k := 0; k < 376; k++ {
			goScroll1(cells, 80, 24)
		}
	}
}

func BenchmarkFeedScrollC(b *testing.B) {
	old := scrollUseC
	scrollUseC = true
	b.Cleanup(func() { scrollUseC = old })
	trace := scrollTrace(400, 80)
	g := &CursorGrid{}
	g.Reset(80, 24)
	p := &Parser{}
	p.Reset(g)
	p.Feed(trace)
	b.SetBytes(int64(len(trace)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.CursorX, g.CursorY = 0, 0
		g.WrapPending = false
		p.Feed(trace)
	}
}

func BenchmarkFeedScrollGoCopy(b *testing.B) {
	old := scrollUseC
	scrollUseC = false
	b.Cleanup(func() { scrollUseC = old })
	trace := scrollTrace(400, 80)
	g := &CursorGrid{}
	g.Reset(80, 24)
	p := &Parser{}
	p.Reset(g)
	p.Feed(trace)
	b.SetBytes(int64(len(trace)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.CursorX, g.CursorY = 0, 0
		g.WrapPending = false
		p.Feed(trace)
	}
}

func TestScrollParityCVsGo(t *testing.T) {
	const w, h = 80, 24
	src := fillCells(w * h)
	c := make([]C2_cell_t, w*h)
	g := make([]Cell, w*h)
	for i := range src {
		c[i] = C2_cell_t{Rune_: uint32(src[i].Rune), Fg: src[i].Fg, Bg: src[i].Bg, Flags: src[i].Flags, Width: src[i].Width}
		g[i] = src[i]
	}
	C2_grid_scroll_up(c, w, h, w, 1)
	goScroll1(g, w, h)
	for i := 0; i < w*(h-1); i++ {
		if rune(c[i].Rune_) != g[i].Rune {
			t.Fatalf("cell %d C=%d Go=%d", i, c[i].Rune_, g[i].Rune)
		}
	}
}
