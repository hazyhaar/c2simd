package c2tui

import (
	"math/rand"
	"testing"
)

func benchPrep(w, h, kind, nframe int) (*CursorGrid, *Parser, [][]byte, []Cell, []Span) {
	var g CursorGrid
	var p Parser
	g.Reset(w, h)
	p.Reset(&g)
	rng := rand.New(rand.NewSource(1))
	frames := make([][]byte, nframe)
	for i := range frames {
		k := kind
		if k < 0 {
			k = rng.Intn(12)
		}
		frames[i] = append([]byte(nil), exoticFrame(rng, nil, k)...)
	}
	p.Feed(frames[0])
	front := make([]Cell, w*h)
	spans := make([]Span, 0, w*h)
	return &g, &p, frames, front, spans
}

func BenchmarkCopyFront80x24(b *testing.B) {
	g, _, _, front, _ := benchPrep(80, 24, 11, 1)
	b.SetBytes(int64(len(front) * 8))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(front, g.DiffCells())
	}
}

func BenchmarkFeedScroll80x24(b *testing.B) {
	g, p, frames, _, _ := benchPrep(80, 24, 7, 4)
	var nbyte int64
	for _, fr := range frames {
		nbyte += int64(len(fr))
	}
	b.SetBytes(nbyte / int64(len(frames)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Feed(frames[i%len(frames)])
		_ = g
	}
}

func BenchmarkFeedFullpaint80x24(b *testing.B) {
	g, p, frames, _, _ := benchPrep(80, 24, 11, 4)
	var nbyte int64
	for _, fr := range frames {
		nbyte += int64(len(fr))
	}
	b.SetBytes(nbyte / int64(len(frames)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Feed(frames[i%len(frames)])
		_ = g
	}
}

func BenchmarkDiff80x24(b *testing.B) {
	g, p, frames, front, spans := benchPrep(80, 24, 11, 2)
	copy(front, g.DiffCells())
	p.Feed(frames[1])
	b.SetBytes(int64(80 * 24 * 8))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		spans = spans[:0]
		DiffGrid(front, g.DiffCells(), 80, 24, 80, &spans)
	}
}

func BenchmarkPipeline80x24(b *testing.B) {
	g, p, frames, front, spans := benchPrep(80, 24, -1, 32)
	var nbyte int64
	for _, fr := range frames {
		nbyte += int64(len(fr))
	}
	b.SetBytes(nbyte / int64(len(frames)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fr := frames[i%len(frames)]
		copy(front, g.DiffCells())
		p.Feed(fr)
		spans = spans[:0]
		DiffGrid(front, g.DiffCells(), 80, 24, 80, &spans)
	}
}

func BenchmarkPipelineScroll(b *testing.B) {
	g, p, frames, front, spans := benchPrep(80, 24, 7, 8)
	var nbyte int64
	for _, fr := range frames {
		nbyte += int64(len(fr))
	}
	b.SetBytes(nbyte / int64(len(frames)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fr := frames[i%len(frames)]
		copy(front, g.DiffCells())
		p.Feed(fr)
		spans = spans[:0]
		DiffGrid(front, g.DiffCells(), 80, 24, 80, &spans)
	}
}

func BenchmarkPipelineChunk1(b *testing.B) {
	g, p, frames, front, spans := benchPrep(80, 24, 11, 4)
	fr := frames[0]
	b.SetBytes(int64(len(fr)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(front, g.DiffCells())
		for off := 0; off < len(fr); off++ {
			p.Feed(fr[off : off+1])
		}
		spans = spans[:0]
		DiffGrid(front, g.DiffCells(), 80, 24, 80, &spans)
	}
}

func BenchmarkPipeline200x60(b *testing.B) {
	if testing.Short() {
		b.Skip("short")
	}
	g, p, frames, front, spans := benchPrep(200, 60, 7, 8)
	var nbyte int64
	for _, fr := range frames {
		nbyte += int64(len(fr))
	}
	b.SetBytes(nbyte / int64(len(frames)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fr := frames[i%len(frames)]
		copy(front, g.DiffCells())
		p.Feed(fr)
		spans = spans[:0]
		DiffGrid(front, g.DiffCells(), 200, 60, 200, &spans)
	}
}
