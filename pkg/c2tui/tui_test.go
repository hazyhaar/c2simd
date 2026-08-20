package c2tui

import "testing"

func TestFeedDiff(t *testing.T) {
	var g CursorGrid
	var p Parser
	g.Reset(8, 2)
	p.Reset(&g)
	p.Feed([]byte("HELLO"))
	front := append([]Cell(nil), g.DiffCells()...)
	p.Feed([]byte("\rXXXXX"))
	spans := make([]Span, 0, 8)
	n := DiffGrid(front, g.DiffCells(), g.Width, g.Height, g.Width, &spans)
	if n < 4 {
		t.Fatalf("dirty=%d spans=%v", n, spans)
	}
}
