package c2tui

import "testing"

func FuzzFeedThenDiff(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte("hello"),
		[]byte("\x1b[2J\x1b[H"),
		[]byte("\x1b[31;1mX\x1b[0m"),
		[]byte("\x1b[10;10H\x1b[Kzzz"),
		[]byte("\x1b]52;c;QQ==\x07"),
		[]byte("\r\n\t\x00\x1b"),
		[]byte("日本語"),
		[]byte("\x1b[1;80H\x1b[2K\x1b[A"),
		[]byte("\x1b[;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;H"),
		[]byte("\x1b[38;2;1"),
		[]byte{0xc3},
		[]byte{0xf0, 0x9f, 0xa7},
		[]byte("\x1bPxxxx\x1b\\"),
		[]byte("\x1b[9999;9999H"),
		[]byte("\x1b[2;2r\n\n\n\n"),
		append([]byte("\x1b["), make([]byte, 200)...),
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 8192 {
			data = data[:8192]
		}
		runFuzzBody(t, data, 0)
	})
}

func FuzzFeedChunked(f *testing.F) {
	f.Add([]byte("\x1b[31mABC\x1b[0m"), 1)
	f.Add([]byte("\x1b[2Jhello\n"), 2)
	f.Add([]byte{0x1b, '[', '3', '8', ';', '2'}, 1)
	f.Fuzz(func(t *testing.T, data []byte, chunk int) {
		if len(data) > 4096 {
			data = data[:4096]
		}
		if chunk < 1 {
			chunk = 1
		}
		if chunk > 64 {
			chunk = 64
		}
		runFuzzBody(t, data, chunk)
	})
}

func runFuzzBody(t *testing.T, data []byte, chunk int) {
	t.Helper()
	var g CursorGrid
	var p Parser
	g.Reset(40, 12)
	p.Reset(&g)
	front := make([]Cell, 40*12)
	copy(front, g.DiffCells())
	if chunk <= 0 {
		p.Feed(data)
	} else {
		for off := 0; off < len(data); {
			n := chunk
			if off+n > len(data) {
				n = len(data) - off
			}
			p.Feed(data[off : off+n])
			off += n
		}
	}
	checkGrid(t, &g, "fuzz")
	spans := make([]Span, 0, 40*12)
	n := DiffGrid(front, g.DiffCells(), 40, 12, 40, &spans)
	if n < 0 {
		t.Fatalf("DiffGrid=%d", n)
	}
	sum := 0
	for _, s := range spans {
		if s.Length < 1 || s.X < 0 || s.Y < 0 || s.X+s.Length > 40 || s.Y >= 12 {
			t.Fatalf("span %+v", s)
		}
		sum += s.Length
	}
	if n != 0 && sum != n {
		t.Fatalf("dirty=%d spanSum=%d", n, sum)
	}
}
