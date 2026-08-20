package c2tui

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func try(t *testing.T, name string, fn func()) (outcome string) {
	t.Helper()
	done := make(chan struct{})
	var panicked any
	go func() {
		defer func() {
			panicked = recover()
			close(done)
		}()
		fn()
	}()
	select {
	case <-done:
		if panicked != nil {
			outcome = fmt.Sprintf("panic: %v", panicked)
		} else {
			outcome = "ok"
		}
	case <-time.After(2 * time.Second):
		outcome = "HANG"
	}
	t.Logf("%s: %s", name, outcome)
	return outcome
}

func TestAbuseCatalogue(t *testing.T) {
	hang := 0
	panicN := 0
	note := func(s string) {
		if s == "HANG" {
			hang++
		}
		if len(s) > 6 && s[:6] == "panic:" {
			panicN++
		}
	}

	note(try(t, "Feed sans Reset", func() {
		var p Parser
		p.Feed([]byte("hello"))
	}))
	note(try(t, "Reset -1x10", func() {
		var g CursorGrid
		g.Reset(-1, 10)
	}))
	note(try(t, "DiffGrid spans nil", func() {
		var spans *[]Span
		_ = DiffGrid(make([]Cell, 80), make([]Cell, 80), 8, 2, 8, spans)
	}))
	note(try(t, "DiffGrid cap spans 0 (realloc)", func() {
		front := make([]Cell, 80*24)
		back := make([]Cell, 80*24)
		back[0].Rune = 'X'
		spans := []Span{}
		n := DiffGrid(front, back, 80, 24, 80, &spans)
		if n < 1 || cap(spans) < 1 {
			panic(fmt.Sprintf("n=%d cap=%d", n, cap(spans)))
		}
	}))
	note(try(t, "CSI 200k digits", func() {
		var g CursorGrid
		var p Parser
		g.Reset(80, 24)
		p.Reset(&g)
		buf := make([]byte, 0, 200003)
		buf = append(buf, 0x1b, '[')
		for i := 0; i < 200000; i++ {
			buf = append(buf, '9')
		}
		buf = append(buf, 'H')
		p.Feed(buf)
	}))
	note(try(t, "Feed 8 MiB noise", func() {
		var g CursorGrid
		var p Parser
		g.Reset(80, 24)
		p.Reset(&g)
		buf := make([]byte, 8<<20)
		for i := range buf {
			buf[i] = byte(i)
		}
		p.Feed(buf)
	}))
	note(try(t, "Reset 4000x2000", func() {
		var g CursorGrid
		var p Parser
		g.Reset(4000, 2000)
		p.Reset(&g)
		p.Feed([]byte("x\n\n\n"))
		front := make([]Cell, len(g.Cells))
		copy(front, g.DiffCells())
		p.Feed([]byte("y"))
		spans := make([]Span, 0, 64)
		_ = DiffGrid(front, g.DiffCells(), 4000, 2000, 4000, &spans)
	}))
	note(try(t, "Feed concurrent 8G", func() {
		var g CursorGrid
		var p Parser
		g.Reset(80, 24)
		p.Reset(&g)
		var wg sync.WaitGroup
		var mu sync.Mutex
		var child any
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						mu.Lock()
						child = r
						mu.Unlock()
					}
				}()
				for j := 0; j < 200; j++ {
					p.Feed([]byte("\x1b[31mhello\n"))
				}
			}()
		}
		wg.Wait()
		if child != nil {
			panic(child)
		}
	}))
	note(try(t, "Reset 0x0 puis Feed ASCII", func() {
		var g CursorGrid
		var p Parser
		g.Reset(0, 0)
		p.Reset(&g)
		p.Feed([]byte("AAAA"))
	}))

	t.Logf("hangs=%d panics=%d (catalogue, pas un contrat)", hang, panicN)
}
