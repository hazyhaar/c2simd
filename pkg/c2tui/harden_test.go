package c2tui

import (
	"sync"
	"testing"
	"time"
)

func TestResetZeroFeed(t *testing.T) {
	var g CursorGrid
	var p Parser
	g.Reset(0, 0)
	p.Reset(&g)
	done := make(chan struct{})
	go func() {
		p.Feed([]byte("AAAA"))
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("hang")
	}
}

func TestResetNegative(t *testing.T) {
	var g CursorGrid
	g.Reset(-1, 10)
	if g.Width != 0 || len(g.Cells) != 0 {
		t.Fatalf("%+v", g)
	}
}

func TestDiffGridNilSpans(t *testing.T) {
	n := DiffGrid(make([]Cell, 16), make([]Cell, 16), 8, 2, 8, nil)
	if n != 0 {
		t.Fatalf("n=%d", n)
	}
}

func TestFeedReentrantPanics(t *testing.T) {
	var g CursorGrid
	var p Parser
	g.Reset(80, 24)
	p.Reset(&g)
	var saw string
	var wg sync.WaitGroup
	start := make(chan struct{})
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					saw = "panic"
				}
			}()
			<-start
			for j := 0; j < 500; j++ {
				p.Feed([]byte("hello\nworld\n"))
			}
		}()
	}
	close(start)
	wg.Wait()
	if saw != "panic" {
		t.Fatal("expected concurrent Feed panic")
	}
}
