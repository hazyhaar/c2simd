package c2grid

import "testing"

func TestGridPutCell(t *testing.T) {
	g := NewGrid(10, 10)
	g.PutCell('A', 1)
	if (*g.active)[0].Rune != 'A' {
		t.Errorf("Expected 'A', got %v", (*g.active)[0].Rune)
	}
}

func BenchmarkPutCell(b *testing.B) {
	g := NewGrid(80, 24)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		g.PutCell('A', 1)
	}
}
