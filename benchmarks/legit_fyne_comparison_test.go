package benchmarks

import (
	"fmt"
	"image/color"
	"runtime"
	"testing"

	// 1. Fyne Légitime (Upstream v2.8.0)
	fyne "fyne.io/fyne/v2"
	fynecanvas "fyne.io/fyne/v2/canvas"
	fynecontainer "fyne.io/fyne/v2/container"
	fynetest "fyne.io/fyne/v2/test"
	fynewidget "fyne.io/fyne/v2/widget"

	// 2. Fyne55 Souverain (CGO-Free / sgoiter)
	c2fynedriver "code.hazyhaar.fr/devhoros/pkg/c2fynedriver"
	c2painter "code.hazyhaar.fr/devhoros/pkg/c2painter"
)

// =============================================================================
// 1. SCÈNES DE CHARGE ÉQUIVALENTES (500 WIDGETS MUTANTS)
// =============================================================================

func helperBuildFyneLegitScene(widgetCount int) (fynetest.WindowlessCanvas, []fyne.CanvasObject) {
	c := fynetest.NewCanvas()
	c.Resize(fyne.NewSize(1280, 800))

	mutants := make([]fyne.CanvasObject, 0, widgetCount)
	content := fynecontainer.NewWithoutLayout()

	// En-tête
	header := fynecanvas.NewRectangle(color.NRGBA{R: 15, G: 23, B: 42, A: 255})
	header.Move(fyne.NewPos(0, 0))
	header.Resize(fyne.NewSize(1280, 60))
	content.Add(header)

	title := fynecanvas.NewText("BENCHMARK FYNE LEGIT - UPSTREAM", color.NRGBA{R: 248, G: 250, B: 252, A: 255})
	title.Move(fyne.NewPos(20, 18))
	title.TextSize = 16
	content.Add(title)

	startX := float32(540)
	startY := float32(80)
	cols := 20

	for i := 0; i < widgetCount; i++ {
		row := i / cols
		col := i % cols
		x := startX + float32(col*36)
		y := startY + float32(row*28)

		var obj fyne.CanvasObject
		mod := i % 6
		switch mod {
		case 0:
			rect := fynecanvas.NewRectangle(color.NRGBA{R: uint8((i * 37) % 255), G: uint8((i * 59) % 255), B: 180, A: 255})
			rect.Move(fyne.NewPos(x, y))
			rect.Resize(fyne.NewSize(32, 24))
			obj = rect
		case 1:
			circ := fynecanvas.NewCircle(color.NRGBA{R: 220, G: uint8((i * 29) % 255), B: uint8((i * 83) % 255), A: 255})
			circ.Move(fyne.NewPos(x, y))
			circ.Resize(fyne.NewSize(24, 24))
			obj = circ
		case 2:
			lbl := fynewidget.NewLabel(fmt.Sprintf("W%03d", i))
			lbl.Move(fyne.NewPos(x, y))
			lbl.Resize(fyne.NewSize(32, 20))
			obj = lbl
		case 3:
			btn := fynewidget.NewButton("K", nil)
			btn.Move(fyne.NewPos(x, y))
			btn.Resize(fyne.NewSize(30, 22))
			obj = btn
		case 4:
			line := fynecanvas.NewLine(color.NRGBA{R: 245, G: 158, B: 11, A: 255})
			line.Position1 = fyne.NewPos(x, y)
			line.Position2 = fyne.NewPos(x+30, y+20)
			line.StrokeWidth = 2
			obj = line
		case 5:
			card := fynewidget.NewCard(fmt.Sprintf("C%d", i%10), "", nil)
			card.Move(fyne.NewPos(x, y))
			card.Resize(fyne.NewSize(32, 24))
			obj = card
		}

		content.Add(obj)
		mutants = append(mutants, obj)
	}

	c.SetContent(content)
	return c, mutants
}

// BenchmarkCompare_Sustained_FyneLegit mesure le rendu complet de trame Fyne Légitime (v2.8.0).
func BenchmarkCompare_Sustained_FyneLegit(b *testing.B) {
	c, mutants := helperBuildFyneLegitScene(500)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		shift := uint8(i % 255)
		for j, obj := range mutants {
			switch elem := obj.(type) {
			case *fynecanvas.Rectangle:
				elem.FillColor = color.NRGBA{R: uint8((shift + uint8(j*13)) % 255), G: uint8((j * 41) % 255), B: 180, A: 255}
				elem.Refresh()
			case *fynecanvas.Circle:
				elem.FillColor = color.NRGBA{R: 240, G: uint8((shift + uint8(j*7)) % 255), B: uint8((j * 31) % 255), A: 255}
				elem.Refresh()
			case *fynewidget.Label:
				if i%10 == 0 {
					elem.SetText(fmt.Sprintf("M%03d", (j+i)%1000))
				}
			}
		}

		_ = c.Capture()
	}
}

// BenchmarkCompare_Sustained_Fyne55 mesure le rendu complet de trame Fyne55 (sgoiter / CGO-Free).
func BenchmarkCompare_Sustained_Fyne55(b *testing.B) {
	drv, err := c2fynedriver.NewHeadlessDriver(1280, 800)
	if err != nil {
		b.Fatalf("Erreur NewHeadlessDriver: %v", err)
	}

	win, err := drv.CreateWindow("Fyne55 Benchmark", 1280, 800)
	if err != nil {
		b.Fatalf("Erreur CreateWindow: %v", err)
	}
	defer win.Close()

	root, _, mutants := helperBuildDenseUI(drv, 500)
	win.SetContent(root)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		helperMutateWidgets(mutants, i)
		win.RenderFrame()
	}
}

// =============================================================================
// 2. BANC COMPARATIF : MONTÉE À L'ÉCHELLE (100 -> 10 000 WIDGETS)
// =============================================================================

func BenchmarkCompare_Scale_FyneLegit(b *testing.B) {
	counts := []int{100, 500, 1000, 5000}
	for _, count := range counts {
		b.Run(fmt.Sprintf("Legit_%d_Widgets", count), func(subB *testing.B) {
			c, _ := helperBuildFyneLegitScene(count)

			subB.ResetTimer()
			subB.ReportAllocs()

			for i := 0; i < subB.N; i++ {
				_ = c.Capture()
			}
		})
	}
}

func BenchmarkCompare_Scale_Fyne55(b *testing.B) {
	counts := []int{100, 500, 1000, 5000}
	for _, count := range counts {
		b.Run(fmt.Sprintf("Fyne55_%d_Widgets", count), func(subB *testing.B) {
			drv, err := c2fynedriver.NewHeadlessDriver(1280, 800)
			if err != nil {
				subB.Fatalf("Erreur NewHeadlessDriver: %v", err)
			}

			win, err := drv.CreateWindow("Scale Benchmark", 1280, 800)
			if err != nil {
				subB.Fatalf("Erreur CreateWindow: %v", err)
			}
			defer win.Close()

			root, _, _ := helperBuildDenseUI(drv, count)
			win.SetContent(root)

			subB.ResetTimer()
			subB.ReportAllocs()

			for i := 0; i < subB.N; i++ {
				win.RenderFrame()
			}
		})
	}
}

// =============================================================================
// 3. BILAN COMPARATIF MÉMOIRE ET RAPPORT EXHAUSTIF
// =============================================================================

func TestCompare_Comprehensive_Summary(t *testing.T) {
	// 1. Empreinte Mémoire Fyne Légitime
	runtime.GC()
	var mBeforeLegit runtime.MemStats
	runtime.ReadMemStats(&mBeforeLegit)

	cLegit, _ := helperBuildFyneLegitScene(1000)
	_ = cLegit.Capture()

	var mAfterLegit runtime.MemStats
	runtime.ReadMemStats(&mAfterLegit)
	legitHeapKB := int64(mAfterLegit.HeapAlloc-mBeforeLegit.HeapAlloc) / 1024

	// 2. Empreinte Mémoire Fyne55
	runtime.GC()
	var mBefore55 runtime.MemStats
	runtime.ReadMemStats(&mBefore55)

	drv, _ := c2fynedriver.NewHeadlessDriver(1280, 800)
	win55, _ := drv.CreateWindow("Test", 1280, 800)
	root, _, _ := helperBuildDenseUI(drv, 1000)
	win55.SetContent(root)
	win55.RenderFrame()

	var mAfter55 runtime.MemStats
	runtime.ReadMemStats(&mAfter55)
	fyne55HeapKB := int64(mAfter55.HeapAlloc-mBefore55.HeapAlloc) / 1024

	t.Logf("=================================================================")
	t.Logf("BILAN COMPARATIF FORMEL : FYNE LÉGITIME (v2.8.0) vs FYNE55 (sgoiter)")
	t.Logf("=================================================================")
	t.Logf("Scène 1 000 widgets sur surface 1280x800 px :")
	t.Logf("  * Fyne Légitime (Upstream) : Heap alloué = %d Ko", legitHeapKB)
	t.Logf("  * Fyne55 Souverain (sgoiter) : Heap alloué = %d Ko", fyne55HeapKB)
	t.Logf("=================================================================")
}

// =============================================================================
// 4. BANC COMPARATIF : RASTERISATION D'UN RECTANGLE 100x100
// =============================================================================

func BenchmarkCompare_PrimitiveRect_FyneLegit(b *testing.B) {
	c := fynetest.NewCanvas()
	c.Resize(fyne.NewSize(100, 100))
	rect := fynecanvas.NewRectangle(color.NRGBA{R: 255, G: 0, B: 0, A: 255})
	rect.Resize(fyne.NewSize(100, 100))
	c.SetContent(rect)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = c.Capture()
	}
}

func BenchmarkCompare_PrimitiveRect_Fyne55(b *testing.B) {
	surf := c2painter.NewSurface(100, 100)
	p := c2painter.NewPainter(surf)
	col := c2painter.PackRGBA(255, 0, 0, 255)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		p.DrawRect(0, 0, 100, 100, col)
	}
}
