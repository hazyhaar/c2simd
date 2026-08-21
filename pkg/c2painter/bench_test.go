package c2painter

import (
	"fmt"
	"testing"
)

func BenchmarkClear1080p(b *testing.B) {
	surf := NewSurface(1920, 1080)
	p := NewPainter(surf)
	col := RGBA(30, 30, 35, 255)

	b.SetBytes(1920 * 1080 * 4)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		p.Clear(col)
	}
}

func BenchmarkClear4K(b *testing.B) {
	surf := NewSurface(3840, 2160)
	p := NewPainter(surf)
	col := RGBA(30, 30, 35, 255)

	b.SetBytes(3840 * 2160 * 4)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		p.Clear(col)
	}
}

func BenchmarkFillRectOpaque(b *testing.B) {
	surf := NewSurface(1920, 1080)
	p := NewPainter(surf)
	col := RGBA(255, 100, 50, 255)

	b.SetBytes(500 * 300 * 4)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		p.DrawRect(100, 100, 500, 300, col)
	}
}

func BenchmarkFillRectAlphaBlend(b *testing.B) {
	surf := NewSurface(1920, 1080)
	p := NewPainter(surf)
	col := RGBA(255, 100, 50, 128)

	b.SetBytes(500 * 300 * 4)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		p.DrawRect(100, 100, 500, 300, col)
	}
}

func BenchmarkFillRoundedRectAA(b *testing.B) {
	surf := NewSurface(1920, 1080)
	p := NewPainter(surf)
	col := RGBA(50, 150, 255, 220)

	b.SetBytes(400 * 250 * 4)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		p.DrawRoundedRect(150, 150, 400, 250, 20, col)
	}
}

func BenchmarkFillCircleAA(b *testing.B) {
	surf := NewSurface(1920, 1080)
	p := NewPainter(surf)
	col := RGBA(255, 50, 150, 220)

	b.SetBytes(200 * 200 * 4)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		p.DrawCircle(500, 500, 100, col)
	}
}

func BenchmarkDrawLineAA(b *testing.B) {
	surf := NewSurface(1920, 1080)
	p := NewPainter(surf)
	col := RGBA(255, 255, 0, 200)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		p.DrawLine(100, 100, 900, 700, 4, col)
	}
}

func BenchmarkLinearGradient1080p(b *testing.B) {
	surf := NewSurface(1920, 1080)
	p := NewPainter(surf)
	col0 := RGBA(15, 23, 42, 255)
	col1 := RGBA(88, 28, 135, 255)

	b.SetBytes(1920 * 1080 * 4)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		p.DrawLinearGradient(0, 0, 1920, 1080, col0, col1, true)
	}
}

func BenchmarkBlitMaskGlyph(b *testing.B) {
	surf := NewSurface(1920, 1080)
	p := NewPainter(surf)
	mask := make([]byte, 32*32)
	for i := range mask {
		mask[i] = uint8((i * 17) & 0xFF)
	}
	col := RGBA(255, 255, 255, 255)

	b.SetBytes(32 * 32 * 4)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		p.DrawTextGlyph(200, 200, 32, 32, mask, 32, col)
	}
}

func BenchmarkUIFrame1080p(b *testing.B) {
	surf := NewSurface(1920, 1080)
	p := NewPainter(surf)

	// Pré-génération de masque de glyphe (simulant police de caractères)
	glyph := make([]byte, 16*24)
	for i := range glyph {
		glyph[i] = 200
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// 1. Fond dégradé
		p.DrawLinearGradient(0, 0, 1920, 1080, RGBA(20, 24, 33, 255), RGBA(10, 12, 18, 255), true)

		// 2. Barre latérale
		p.DrawRect(0, 0, 260, 1080, RGBA(15, 18, 25, 240))
		p.DrawLine(260, 0, 260, 1080, 1, RGBA(40, 48, 65, 255))

		// 3. Barre d'en-tête
		p.DrawRect(260, 0, 1660, 60, RGBA(18, 22, 30, 220))
		p.DrawLine(260, 60, 1920, 60, 1, RGBA(40, 48, 65, 255))

		// 4. Cartes / Panneaux (6 cartes arrondies)
		for row := 0; row < 2; row++ {
			for col := 0; col < 3; col++ {
				cx := 300 + col*520
				cy := 100 + row*450
				p.DrawRoundedRect(cx, cy, 480, 400, 12, RGBA(25, 30, 42, 230))
				p.StrokeRoundedRect(cx, cy, 480, 400, 12, 1, RGBA(50, 60, 85, 255))

				// Bouton d'action dans la carte
				p.DrawRoundedRect(cx+30, cy+320, 140, 44, 8, RGBA(59, 130, 246, 255))
				p.DrawCircle(cx+420, cy+50, 18, RGBA(16, 185, 129, 255))

				// 10 "glyphes" de texte simulés
				for g := 0; g < 10; g++ {
					p.DrawTextGlyph(cx+30+g*18, cy+80, 16, 24, glyph, 16, RGBA(240, 245, 255, 255))
				}
			}
		}
	}
}

func BenchmarkUIFrame4K(b *testing.B) {
	surf := NewSurface(3840, 2160)
	p := NewPainter(surf)

	glyph := make([]byte, 32*48)
	for i := range glyph {
		glyph[i] = 200
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		p.DrawLinearGradient(0, 0, 3840, 2160, RGBA(20, 24, 33, 255), RGBA(10, 12, 18, 255), true)
		p.DrawRect(0, 0, 520, 2160, RGBA(15, 18, 25, 240))
		p.DrawRoundedRect(600, 200, 960, 800, 24, RGBA(25, 30, 42, 230))
		p.StrokeRoundedRect(600, 200, 960, 800, 24, 2, RGBA(50, 60, 85, 255))
		p.DrawCircle(3000, 1000, 80, RGBA(59, 130, 246, 255))
		p.DrawLine(520, 120, 3840, 120, 2, RGBA(40, 48, 65, 255))
	}
}

func TestFPSReporting(t *testing.T) {
	surf := NewSurface(1920, 1080)
	p := NewPainter(surf)

	// Exécution d'une frame et vérification non vide
	p.Clear(RGBA(10, 10, 10, 255))
	p.DrawRoundedRect(100, 100, 800, 600, 16, RGBA(0, 128, 255, 200))
	p.DrawCircle(500, 400, 150, RGBA(255, 200, 0, 220))
	p.DrawLine(50, 50, 1870, 1030, 4, RGBA(255, 50, 50, 255))

	nonZero := 0
	for _, pix := range p.Pixels() {
		if pix != 0 {
			nonZero++
		}
	}
	if nonZero == 0 {
		t.Fatalf("frame vide")
	}
	fmt.Printf("Validation rendu 1080p: %d/%d pixels renseignés\n", nonZero, len(p.Pixels()))
}
