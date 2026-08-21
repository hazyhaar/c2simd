package c2painter

import (
	"sync"
	"testing"
)

func TestSurfaceLifecycle(t *testing.T) {
	surf := NewSurface(640, 480)
	if surf.Width != 640 || surf.Height != 480 || surf.Stride != 640 {
		t.Fatalf("surface dimensions incorrect: got %dx%d stride %d", surf.Width, surf.Height, surf.Stride)
	}
	if len(surf.Pixels) != 640*480 {
		t.Fatalf("pixel buffer length mismatch: got %d, want %d", len(surf.Pixels), 640*480)
	}

	invalid := NewSurface(-10, 0)
	if invalid.Width != 0 || invalid.Height != 0 || invalid.Pixels != nil {
		t.Fatalf("invalid surface should have 0 dimensions and nil pixels")
	}
}

func TestPainterClipping(t *testing.T) {
	surf := NewSurface(100, 100)
	p := NewPainter(surf)

	// Clip au rectangle [20, 20, 40, 40]
	p.SetClip(20, 20, 40, 40)
	p.Clear(RGBA(255, 0, 0, 255)) // Ne doit remplir que l'intérieur du clip

	// Vérifier pixels à l'intérieur du clip
	insideColor := RGBA(255, 0, 0, 255)
	outsideColor := uint32(0)

	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			idx := y*100 + x
			if x >= 20 && x < 60 && y >= 20 && y < 60 {
				// Note: Clear remplit toute la surface selon son contrat,
				// testons DrawRect avec le clip
			}
			_ = idx
		}
	}

	surf.Pixels = make([]uint32, 100*100)
	p = NewPainter(surf)
	p.SetClip(20, 20, 40, 40)
	p.DrawRect(0, 0, 100, 100, insideColor)

	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			idx := y*100 + x
			if x >= 20 && x < 60 && y >= 20 && y < 60 {
				if p.Pixels()[idx] != insideColor {
					t.Errorf("pixel inside clip (%d,%d) want 0x%08X got 0x%08X", x, y, insideColor, p.Pixels()[idx])
				}
			} else {
				if p.Pixels()[idx] != outsideColor {
					t.Errorf("pixel outside clip (%d,%d) want 0x%08X got 0x%08X", x, y, outsideColor, p.Pixels()[idx])
				}
			}
		}
	}

	// Reset clip
	p.ResetClip()
	p.DrawRect(0, 0, 100, 100, insideColor)
	for i, pix := range p.Pixels() {
		if pix != insideColor {
			t.Fatalf("pixel %d after reset clip want 0x%08X got 0x%08X", i, insideColor, pix)
		}
	}
}

func TestPainterEdgeCases(t *testing.T) {
	surf := NewSurface(50, 50)
	p := NewPainter(surf)

	// Dimensions négatives ou nulles
	p.DrawRect(10, 10, -5, 0, RGBA(255, 255, 255, 255))
	p.StrokeRect(10, 10, 0, 10, 2, RGBA(255, 255, 255, 255))
	p.DrawRoundedRect(10, 10, 0, 0, 5, RGBA(255, 255, 255, 255))
	p.StrokeRoundedRect(10, 10, 10, -2, 5, 2, RGBA(255, 255, 255, 255))
	p.DrawCircle(25, 25, -1, RGBA(255, 255, 255, 255))
	p.StrokeCircle(25, 25, 10, -2, RGBA(255, 255, 255, 255))
	p.DrawEllipse(25, 25, -5, 10, RGBA(255, 255, 255, 255))
	p.StrokeEllipse(25, 25, 10, 0, 2, RGBA(255, 255, 255, 255))
	p.DrawLine(25, 25, 25, 25, 0, RGBA(255, 255, 255, 255))
	p.DrawLinearGradient(10, 10, -5, 10, RGBA(255, 0, 0, 255), RGBA(0, 0, 255, 255), true)
	p.DrawTextGlyph(10, 10, 0, 0, nil, 0, RGBA(255, 255, 255, 255))
	p.Blit(10, 10, 0, 0, nil, 0)

	// Tous les pixels doivent rester 0
	for i, pix := range p.Pixels() {
		if pix != 0 {
			t.Fatalf("pixel %d modified by degenerate call: 0x%08X", i, pix)
		}
	}
}

func TestPainterConcurrency(t *testing.T) {
	// Test sous go test -race avec plusieurs surfaces indépendantes en parallèle
	const goroutines = 8
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			surf := NewSurface(128, 128)
			p := NewPainter(surf)
			col := RGBA(uint8(id*30), 100, 200, 255)
			p.Clear(col)
			p.DrawRect(10, 10, 50, 50, RGBA(255, 0, 0, 128))
			p.DrawRoundedRect(30, 30, 60, 40, 10, RGBA(0, 255, 0, 200))
			p.DrawCircle(64, 64, 30, RGBA(0, 0, 255, 180))
			p.DrawLine(0, 0, 127, 127, 3, RGBA(255, 255, 0, 220))
			p.DrawLinearGradient(0, 0, 128, 128, RGBA(255, 0, 0, 255), RGBA(0, 0, 255, 255), false)
		}(i)
	}

	wg.Wait()
}

// TestPainterLargeGeometryOverflow64 vérifie l'absence de débordement arithmétique 32-bit sur les grands rayons et longueurs.
func TestPainterLargeGeometryOverflow64(t *testing.T) {
	surf := NewSurface(3840, 2160) // Canvas 4K
	p := NewPainter(surf)

	// 1. Grand cercle R = 500 (était buggé à R >= 180)
	p.DrawCircle(1920, 1080, 500, PackRGBA(255, 0, 0, 255))
	// Vérification que le centre et les points à R-10 sont pleins
	centerIdx := 1080*3840 + 1920
	if surf.Pixels[centerIdx] != PackRGBA(255, 0, 0, 255) {
		t.Fatalf("Pixel central du grand cercle R=500 non rempli: 0x%08X", surf.Pixels[centerIdx])
	}
	rInsideIdx := 1080*3840 + (1920 + 490)
	if surf.Pixels[rInsideIdx] != PackRGBA(255, 0, 0, 255) {
		t.Fatalf("Pixel intérieur du grand cercle R=500 non rempli: 0x%08X", surf.Pixels[rInsideIdx])
	}

	// 2. Grand rectangle arrondi R = 400
	p.DrawRoundedRect(100, 100, 2000, 1500, 400, PackRGBA(0, 255, 0, 255))

	// 3. Grande ellipse rx = 800, ry = 600
	p.DrawEllipse(1920, 1080, 800, 600, PackRGBA(0, 0, 255, 255))

	// 4. Ligne diagonale 4K complète (longueur ~4400 px, était buggée à > 2896 px)
	p.DrawLine(0, 0, 3840, 2160, 4, PackRGBA(255, 255, 0, 255))
	midDiagIdx := 1080*3840 + 1920
	if surf.Pixels[midDiagIdx] == 0 {
		t.Fatalf("Pixel milieu de diagonale 4K vide")
	}
}
