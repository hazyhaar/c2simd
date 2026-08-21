package c2vtparser

import (
	"fmt"
	"strings"
	"testing"
)

// BenchmarkThroughput_ASCII_Direct mesure le débit d'ingestion sur texte brut ASCII pur (64 Ko).
func BenchmarkThroughput_ASCII_Direct(b *testing.B) {
	g := &CursorGrid{}
	g.Reset(160, 50)
	p := &Parser{}
	p.Reset(g)

	var sb strings.Builder
	line := "The quick brown fox jumps over the lazy dog 0123456789 ABCDEFGHIJKLMNOPQRSTUVWXYZ\n"
	for sb.Len() < 64*1024 {
		sb.WriteString(line)
	}
	trace := []byte(sb.String())
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

// BenchmarkThroughput_CSI_Storm mesure le débit sous une tempête de séquences CSI SGR (TrueColor, 256c, Styles).
func BenchmarkThroughput_CSI_Storm(b *testing.B) {
	g := &CursorGrid{}
	g.Reset(160, 50)
	p := &Parser{}
	p.Reset(g)

	var sb strings.Builder
	for i := 0; i < 500; i++ {
		sb.WriteString(fmt.Sprintf("\x1b[38;2;%d;%d;%dm\x1b[48;5;%dm\x1b[1;4mRGB_CELL_%04d\x1b[0m ",
			i%256, (i*2)%256, (i*3)%256, i%256, i))
	}
	trace := []byte(sb.String())
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

// BenchmarkThroughput_DEC_SpecialGraphics mesure le débit sur flux dense de tracé de boîtes VT100.
func BenchmarkThroughput_DEC_SpecialGraphics(b *testing.B) {
	g := &CursorGrid{}
	g.Reset(160, 50)
	p := &Parser{}
	p.Reset(g)

	var sb strings.Builder
	sb.WriteString("\x1b(0") // G0 = DEC Graphics
	for i := 0; i < 1000; i++ {
		sb.WriteString("lqqqqqqqqqqqqqqk\r\nx              x\r\nmqqqqqqqqqqqqqqj\r\n")
	}
	sb.WriteString("\x1b(B") // G0 = ASCII
	trace := []byte(sb.String())
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

// BenchmarkThroughput_WideGrid_400x120 mesure l'ingestion sur une résolution haute (400x120).
func BenchmarkThroughput_WideGrid_400x120(b *testing.B) {
	g := &CursorGrid{}
	g.Reset(400, 120)
	p := &Parser{}
	p.Reset(g)

	var sb strings.Builder
	for i := 0; i < 200; i++ {
		sb.WriteString("\x1b[32m[LOG]\x1b[0m ")
		sb.WriteString("System telemetry packet data payload stream segment ")
		sb.WriteString(strings.Repeat("X", 300))
		sb.WriteString("\r\n")
	}
	trace := []byte(sb.String())
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
