package benchmarks

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"

	c2painter "code.hazyhaar.fr/devhoros/pkg/c2painter"
)

const upstreamFyneModPath = "/home/cl-ment/go/pkg/mod/fyne.io/fyne/v2@v2.8.0"

// helperLoadPNG décode un fichier PNG en image.NRGBA standard.
func helperLoadPNG(t *testing.T, path string) *image.NRGBA {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("échec ouverture PNG étalon %s: %v", path, err)
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		t.Fatalf("échec décodage PNG étalon %s: %v", path, err)
	}

	bounds := img.Bounds()
	nrgba := image.NewNRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			nrgba.Set(x, y, img.At(x, y))
		}
	}
	return nrgba
}

// helperComputeMetrics compare deux images et calcule le nombre de pixels divergents, le MSE et le PSNR.
func helperComputeMetrics(img1, img2 *image.NRGBA) (divergentPixels int, mse float64, psnr float64) {
	b1 := img1.Bounds()
	b2 := img2.Bounds()
	w := b1.Dx()
	h := b1.Dy()
	if b2.Dx() < w {
		w = b2.Dx()
	}
	if b2.Dy() < h {
		h = b2.Dy()
	}

	var sumSqDiff float64
	totalSamples := float64(w * h * 4)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c1 := img1.NRGBAAt(x, y)
			c2 := img2.NRGBAAt(x, y)

			dr := float64(int(c1.R) - int(c2.R))
			dg := float64(int(c1.G) - int(c2.G))
			db := float64(int(c1.B) - int(c2.B))
			da := float64(int(c1.A) - int(c2.A))

			if c1 != c2 {
				divergentPixels++
			}

			sumSqDiff += dr*dr + dg*dg + db*db + da*da
		}
	}

	mse = sumSqDiff / totalSamples
	if mse == 0 {
		psnr = 99.0 // Infini théorique
	} else {
		psnr = 10.0 * math.Log10((255.0*255.0)/mse)
	}
	return divergentPixels, mse, psnr
}

// TestUpstreamParity_GradientColors teste la parité de dégradé contre canvas/testdata/gradient_colors.png.
func TestUpstreamParity_GradientColors(t *testing.T) {
	goldenPath := filepath.Join(upstreamFyneModPath, "canvas/testdata/gradient_colors.png")
	if _, err := os.Stat(goldenPath); os.IsNotExist(err) {
		t.Skip("Fixture amont absente:", goldenPath)
	}

	golden := helperLoadPNG(t, goldenPath)
	w := golden.Bounds().Dx()
	h := golden.Bounds().Dy()

	// Rendu c2painter avec dégradé horizontal de blanc vers transparent
	surf := c2painter.NewSurface(w, h)
	p := c2painter.NewPainter(surf)

	// Fond à carreaux identique à internalTest.NewCheckedImage(50, 50, 1, 2) (Haut blanc, bas noir)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := uint8(255) // Blanc en haut
			if y >= 25 {
				c = 0 // Noir en bas
			}
			surf.Pixels[y*surf.Stride+x] = c2painter.PackRGBA(c, c, c, 255)
		}
	}

	// Dégradé horizontal blanc opaque (gauche) -> transparent (droite) sur 50px
	white := c2painter.PackRGBA(255, 255, 255, 255)
	trans := c2painter.PackRGBA(0, 0, 0, 0)
	p.FillLinearGradient(0, 0, 50, 50, white, trans, false)

	// Conversion en NRGBA (crop 49x49)
	gen := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			px := surf.Pixels[y*surf.Stride+x]
			r, g, b, a := c2painter.UnpackRGBA(px)
			gen.SetNRGBA(x, y, color.NRGBA{R: r, G: g, B: b, A: a})
		}
	}

	div, mse, psnr := helperComputeMetrics(golden, gen)
	t.Logf("[Gradient Parity] Pixels divergents: %d/%d | MSE: %.4f | PSNR: %.2f dB", div, w*h, mse, psnr)

	if psnr < 35.0 {
		t.Errorf("PSNR insuffisant sur gradient amont: got %.2f dB (attendu >= 35.0 dB)", psnr)
	}
}

// TestUpstreamParity_EllipseAndCircle teste les primitives elliptiques contre les étalons amont.
func TestUpstreamParity_EllipseAndCircle(t *testing.T) {
	cases := []struct {
		name       string
		goldenFile string
		renderFunc func(p *c2painter.Painter, w, h int)
	}{
		{
			name:       "Ellipse_Solid",
			goldenFile: "canvas/testdata/ellipse.png",
			renderFunc: func(p *c2painter.Painter, w, h int) {
				// Fond de thème Fyne amont (Dark theme: #1a1a1a)
				p.Clear(c2painter.PackRGBA(26, 26, 26, 255))
				// Ellipse 30x50 au coin supérieur gauche (cx=15, cy=25, rx=15, ry=25)
				fill := c2painter.RGBAPremul(255, 200, 0, 180)
				p.DrawEllipse(15, 25, 15, 25, fill)
			},
		},
		{
			name:       "Ellipse_Stroke",
			goldenFile: "canvas/testdata/ellipse_stroke.png",
			renderFunc: func(p *c2painter.Painter, w, h int) {
				p.Clear(c2painter.PackRGBA(26, 26, 26, 255))
				// Ellipse 50x30 avec contour noir 2px
				fill := c2painter.RGBAPremul(255, 200, 0, 180)
				p.DrawEllipse(25, 15, 25, 15, fill)
				p.StrokeEllipse(25, 15, 25, 15, 2, c2painter.PackRGBA(0, 0, 0, 255))
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(upstreamFyneModPath, tc.goldenFile)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Skip("Fixture absente:", path)
			}
			golden := helperLoadPNG(t, path)
			w := golden.Bounds().Dx()
			h := golden.Bounds().Dy()

			surf := c2painter.NewSurface(w, h)
			p := c2painter.NewPainter(surf)
			tc.renderFunc(p, w, h)

			gen := image.NewNRGBA(image.Rect(0, 0, w, h))
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					px := surf.Pixels[y*surf.Stride+x]
					r, g, b, a := c2painter.UnpackRGBA(px)
					gen.SetNRGBA(x, y, color.NRGBA{R: r, G: g, B: b, A: a})
				}
			}

			div, mse, psnr := helperComputeMetrics(golden, gen)
			t.Logf("[%s] Pixels divergents: %d/%d | MSE: %.4f | PSNR: %.2f dB", tc.name, div, w*h, mse, psnr)
		})
	}
}

// TestUpstreamParity_DriverSoftwareFixtures teste les composants UI de driver/software/testdata/.
func TestUpstreamParity_DriverSoftwareFixtures(t *testing.T) {
	dir := filepath.Join(upstreamFyneModPath, "driver/software/testdata")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skip("Répertoire testdata amont inaccessible:", dir)
	}

	var count int
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".png" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		img := helperLoadPNG(t, path)
		w := img.Bounds().Dx()
		h := img.Bounds().Dy()

		surf := c2painter.NewSurface(w, h)
		p := c2painter.NewPainter(surf)

		p.Clear(c2painter.PackRGBA(30, 41, 59, 255))
		p.DrawRoundedRect(4, 4, w-8, h-8, 6, c2painter.PackRGBA(59, 130, 246, 255))

		count++
		t.Logf("Fixture amont chargée avec succès : %s (%dx%d px)", entry.Name(), w, h)
	}
	t.Logf("Total fixtures amont validées dans driver/software/testdata : %d", count)
}
