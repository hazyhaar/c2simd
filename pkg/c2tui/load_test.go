package c2tui

import (
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"syscall"
	"testing"
	"time"
)

func rusageMS() (user, sys, maxRSS int64) {
	var ru syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil {
		return 0, 0, 0
	}
	return ru.Utime.Nano() / 1e6, ru.Stime.Nano() / 1e6, ru.Maxrss
}

func checkGrid(t *testing.T, g *CursorGrid, where string) {
	t.Helper()
	need := g.Width * g.Height
	if len(g.Cells) != need {
		t.Fatalf("%s: cells=%d want %d×%d", where, len(g.Cells), g.Width, g.Height)
	}
	if g.CursorX < 0 || g.CursorX > g.Width {
		t.Fatalf("%s: CursorX=%d", where, g.CursorX)
	}
	if g.CursorY < 0 || g.CursorY >= g.Height {
		t.Fatalf("%s: CursorY=%d", where, g.CursorY)
	}
	for i, c := range g.Cells {
		if c.Flags > 0x0f {
			t.Fatalf("%s: cell %d flags=%d", where, i, c.Flags)
		}
	}
}

func mixFrame(rng *rand.Rand, dst []byte) []byte {
	return exoticFrame(rng, dst, rng.Intn(12))
}

func exoticFrame(rng *rand.Rand, dst []byte, kind int) []byte {
	dst = dst[:0]
	switch kind {
	case 0:
		dst = append(dst, "\x1b[2J\x1b[H"...)
	case 1:
		for i := 0; i < 200; i++ {
			dst = append(dst, "\x1b[38;5;"...)
			dst = append(dst, byte('0'+i%10))
			dst = append(dst, 'm', byte('A'+i%26), '\n')
		}
	case 2:
		dst = append(dst, "\x1b["...)
		for i := 0; i < 64; i++ {
			dst = append(dst, ';')
		}
		dst = append(dst, 'H')
	case 3:
		dst = append(dst, "\x1b[38;2;"...)
		for i := 0; i < 80; i++ {
			dst = append(dst, byte('0'+rng.Intn(10)))
		}
		dst = append(dst, 'm')
	case 4:
		dst = append(dst, 0xc3, 0xa9, 0xe6, 0x97, 0xa5, 0xf0, 0x9f, 0xa7, 0xea, 0xc0, 0x80, 0xed, 0xa0, 0x80)
	case 5:
		dst = append(dst, "\x1b]52;c;"...)
		for i := 0; i < 256; i++ {
			dst = append(dst, 'A')
		}
		dst = append(dst, 0x07, 0x1b, ']', '0', ';')
		for i := 0; i < 128; i++ {
			dst = append(dst, byte(rng.Intn(256)))
		}
		dst = append(dst, 0x07)
	case 6:
		dst = append(dst, "\x1bP"...)
		for i := 0; i < 300; i++ {
			dst = append(dst, byte(rng.Intn(256)))
		}
		dst = append(dst, 0x1b, '\\')
	case 7:
		for i := 0; i < 120; i++ {
			for x := 0; x < 90; x++ {
				dst = append(dst, byte('a'+(x%26)))
			}
			dst = append(dst, '\n')
		}
	case 8:
		dst = append(dst, "\x1b[9999;9999H\x1b[-1;-1H\x1b[0;0H\x1b[2;2r\x1b[s\x1b[u"...)
	case 9:
		for i := 0; i < 400; i++ {
			dst = append(dst, byte(rng.Intn(256)))
		}
	case 10:
		dst = append(dst, "\x1b[?25h\x1b[?25l\x1b[?1049h\x1b[?1049l\x1b[?1h"...)
	default:
		dst = append(dst, "\x1b[1;1H"...)
		for y := 0; y < 24; y++ {
			for x := 0; x < 80; x++ {
				dst = append(dst, byte('0'+((x+y)%10)))
			}
			dst = append(dst, '\r', '\n')
		}
	}
	return dst
}

type phaseNS struct {
	copy, feed, diff, total int64
	n, bytes, dirty         int
}

func runPhases(g *CursorGrid, p *Parser, front []Cell, spans *[]Span, frames [][]byte, chunk int) phaseNS {
	var ph phaseNS
	w, h := g.Width, g.Height
	for _, fr := range frames {
		t0 := time.Now()
		copy(front, g.DiffCells())
		t1 := time.Now()
		if chunk <= 1 {
			p.Feed(fr)
		} else {
			for off := 0; off < len(fr); {
				n := chunk
				if off+n > len(fr) {
					n = len(fr) - off
				}
				p.Feed(fr[off : off+n])
				off += n
			}
		}
		t2 := time.Now()
		*spans = (*spans)[:0]
		d := DiffGrid(front, g.DiffCells(), w, h, w, spans)
		t3 := time.Now()
		ph.copy += t1.Sub(t0).Nanoseconds()
		ph.feed += t2.Sub(t1).Nanoseconds()
		ph.diff += t3.Sub(t2).Nanoseconds()
		ph.total += t3.Sub(t0).Nanoseconds()
		ph.n++
		ph.bytes += len(fr)
		ph.dirty += d
	}
	return ph
}

func logPhase(t *testing.T, name string, ph phaseNS) {
	t.Helper()
	n := int64(ph.n)
	if n == 0 {
		return
	}
	t.Logf("%s: frames=%d in=%dB dirty=%d copy=%dns/f feed=%dns/f diff=%dns/f total=%dns/f feed/total=%.0f%%",
		name, ph.n, ph.bytes, ph.dirty, ph.copy/n, ph.feed/n, ph.diff/n, ph.total/n,
		100*float64(ph.feed)/float64(ph.total))
}

func TestLoadBottlenecks(t *testing.T) {
	framesN := 2000
	if testing.Short() {
		framesN = 200
	}
	if v := os.Getenv("C2TUI_LOAD_FRAMES"); v != "" {
		fmt.Sscanf(v, "%d", &framesN)
	}
	rng := rand.New(rand.NewSource(20260820))
	type scen struct {
		name  string
		w, h  int
		kind  int
		chunk int
	}
	scens := []scen{
		{"mix80x24", 80, 24, -1, 0},
		{"scroll80x24", 80, 24, 7, 0},
		{"fullpaint80x24", 80, 24, 11, 0},
		{"csistorm80x24", 80, 24, 1, 0},
		{"chunk1_mix", 80, 24, -1, 1},
		{"wide200x60_scroll", 200, 60, 7, 0},
		{"wide200x60_full", 200, 60, 11, 0},
		{"noise80x24", 80, 24, 9, 0},
	}
	runtime.GC()
	var ms0 runtime.MemStats
	runtime.ReadMemStats(&ms0)
	u0, s0, _ := rusageMS()
	g0 := runtime.NumGoroutine()
	wall0 := time.Now()

	for _, sc := range scens {
		var g CursorGrid
		var p Parser
		g.Reset(sc.w, sc.h)
		p.Reset(&g)
		front := make([]Cell, sc.w*sc.h)
		spans := make([]Span, 0, sc.w*sc.h)
		frames := make([][]byte, framesN)
		for i := range frames {
			k := sc.kind
			if k < 0 {
				k = rng.Intn(12)
			}
			frames[i] = append([]byte(nil), exoticFrame(rng, nil, k)...)
		}
		p.Feed(frames[0])
		ph := runPhases(&g, &p, front, &spans, frames, sc.chunk)
		checkGrid(t, &g, sc.name)
		logPhase(t, sc.name, ph)
	}

	runtime.GC()
	var ms1 runtime.MemStats
	runtime.ReadMemStats(&ms1)
	u1, s1, rss := rusageMS()
	g1 := runtime.NumGoroutine()
	if g1 > g0+2 {
		t.Fatalf("goroutine leak %d→%d", g0, g1)
	}
	heapDelta := int64(ms1.HeapInuse) - int64(ms0.HeapInuse)
	if heapDelta > 32<<20 {
		t.Fatalf("heap +%d", heapDelta)
	}
	t.Logf("probes: Δheap=%dB allocs=%d maxRSS=%dKiB user=%dms sys=%dms wall=%dms G=%d P=%d CPU=%d",
		heapDelta, ms1.Mallocs-ms0.Mallocs, rss, u1-u0, s1-s0, time.Since(wall0).Milliseconds(),
		g1, runtime.GOMAXPROCS(0), runtime.NumCPU())
}

func TestLoadZeroAllocHotPath(t *testing.T) {
	const w, h = 80, 24
	var g CursorGrid
	var p Parser
	g.Reset(w, h)
	p.Reset(&g)
	trace := []byte("\x1b[H\x1b[2J\x1b[31mhello\x1b[0m\r\nABCDEFGHIJ")
	p.Feed(trace)
	front := make([]Cell, w*h)
	spans := make([]Span, 0, w*h)
	p.Feed(trace)
	allocs := testing.AllocsPerRun(100, func() {
		copy(front, g.DiffCells())
		p.Feed(trace)
		spans = spans[:0]
		DiffGrid(front, g.DiffCells(), w, h, w, &spans)
	})
	if allocs != 0 {
		t.Fatalf("hot path allocs=%v want 0", allocs)
	}
}

func TestLoadWideGrid(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	const w, h = 200, 60
	var g CursorGrid
	var p Parser
	g.Reset(w, h)
	p.Reset(&g)
	front := make([]Cell, w*h)
	spans := make([]Span, 0, w*h)
	rng := rand.New(rand.NewSource(7))
	frames := make([][]byte, 400)
	for i := range frames {
		frames[i] = append([]byte(nil), exoticFrame(rng, nil, 7)...)
	}
	ph := runPhases(&g, &p, front, &spans, frames, 0)
	checkGrid(t, &g, "wide")
	logPhase(t, "wide200x60x400", ph)
}
