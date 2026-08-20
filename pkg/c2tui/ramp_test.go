package c2tui

import (
	"math/rand"
	"runtime"
	"testing"
	"time"
)

func TestRampUntilLimit(t *testing.T) {
	type step struct {
		name string
		w, h int
		kind int
		n    int
	}
	steps := []step{
		{"80x24 scroll 2k", 80, 24, 7, 2000},
		{"80x24 scroll 8k", 80, 24, 7, 8000},
		{"200x60 scroll 2k", 200, 60, 7, 2000},
		{"400x120 scroll 1k", 400, 120, 7, 1000},
		{"800x240 scroll 400", 800, 240, 7, 400},
		{"1600x480 scroll 100", 1600, 480, 7, 100},
		{"80x24 csi 4k", 80, 24, 1, 4000},
		{"80x24 noise 8k", 80, 24, 9, 8000},
		{"80x24 full 8k", 80, 24, 11, 8000},
	}
	for _, sc := range steps {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("%s CRASH %v", sc.name, r)
				}
			}()
			var g CursorGrid
			var p Parser
			g.Reset(sc.w, sc.h)
			p.Reset(&g)
			front := make([]Cell, sc.w*sc.h)
			spans := make([]Span, 0, sc.w*sc.h)
			rng := rand.New(rand.NewSource(20260821))
			frames := make([][]byte, sc.n)
			for i := range frames {
				frames[i] = append([]byte(nil), exoticFrame(rng, nil, sc.kind)...)
			}
			runtime.GC()
			_, _, rss0 := rusageMS()
			t0 := time.Now()
			ph := runPhases(&g, &p, front, &spans, frames, 0)
			wall := time.Since(t0)
			var ms runtime.MemStats
			runtime.ReadMemStats(&ms)
			_, _, rss1 := rusageMS()
			n := int64(ph.n)
			t.Logf("%s: %dns/f feed=%dns diff=%dns wall=%s heap=%dMiB ΔRSS=%dKiB allocs=%d",
				sc.name, ph.total/n, ph.feed/n, ph.diff/n, wall.Truncate(time.Millisecond),
				ms.HeapInuse>>20, rss1-rss0, ms.Mallocs)
			if wall > 8*time.Second {
				t.Fatalf("%s trop lent: %s (goulet perceptible)", sc.name, wall)
			}
			checkGrid(t, &g, sc.name)
		}()
	}
}

func TestRampGiantFeed(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("CRASH %v", r)
		}
	}()
	var g CursorGrid
	var p Parser
	g.Reset(80, 24)
	p.Reset(&g)
	buf := make([]byte, 32<<20)
	for i := range buf {
		buf[i] = byte(0x20 + i%95)
		if i%80 == 79 {
			buf[i] = '\n'
		}
	}
	t0 := time.Now()
	n := p.Feed(buf)
	d := time.Since(t0)
	t.Logf("Feed 32MiB newlines: writes=%d wall=%s %.1f MB/s", n, d.Truncate(time.Millisecond), float64(len(buf))/d.Seconds()/1e6)
	checkGrid(t, &g, "giant")
}
