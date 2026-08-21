package benchmarks

import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"testing"

	c2fynedriver "code.hazyhaar.fr/devhoros/pkg/c2fynedriver"
	c2painter "code.hazyhaar.fr/devhoros/pkg/c2painter"
)

// helperBuildDenseUI crée une scène UI complexe avec 500 widgets variés et un terminal interactif.
func helperBuildDenseUI(drv *c2fynedriver.Driver, widgetCount int) (*c2fynedriver.Container, *c2fynedriver.TerminalComponent, []c2fynedriver.CanvasObject) {
	root := c2fynedriver.NewContainer()

	// En-tête Slate 900
	headerBg := c2fynedriver.NewRectangle(c2painter.PackRGBA(15, 23, 42, 255))
	headerBg.Move(c2fynedriver.Position{X: 0, Y: 0})
	headerBg.Resize(c2fynedriver.Size{Width: 1280, Height: 60})
	root.Add(headerBg)

	title := c2fynedriver.NewLabel("BENCHMARK FYNE55 - RENDU SOUTENU")
	title.Move(c2fynedriver.Position{X: 20, Y: 18})
	title.TextSize = 16
	title.Color = c2painter.PackRGBA(248, 250, 252, 255)
	root.Add(title)

	badge := c2fynedriver.NewBadge("500 MUTANTS", c2painter.PackRGBA(16, 185, 129, 255), c2painter.PackRGBA(255, 255, 255, 255))
	badge.Move(c2fynedriver.Position{X: 380, Y: 16})
	badge.Resize(badge.MinSize())
	root.Add(badge)

	fpsWidget := c2fynedriver.NewFPSCounter(drv)
	fpsWidget.Move(c2fynedriver.Position{X: 1100, Y: 16})
	root.Add(fpsWidget)

	// Terminal actif
	termComp := c2fynedriver.NewTerminalComponent(60, 15)
	termComp.Move(c2fynedriver.Position{X: 20, Y: 80})
	termComp.Resize(termComp.MinSize())
	_, _ = termComp.Term.Write([]byte("\x1b[1;32m[INIT]\x1b[0m Banc de charge Fyne55 prêt.\r\n"))
	root.Add(termComp)

	// Grille de widgets mutants
	mutants := make([]c2fynedriver.CanvasObject, 0, widgetCount)
	startX := 540
	startY := 80
	cols := 20

	for i := 0; i < widgetCount; i++ {
		row := i / cols
		col := i % cols
		x := startX + col*36
		y := startY + row*28

		var obj c2fynedriver.CanvasObject
		mod := i % 7
		switch mod {
		case 0:
			r := c2fynedriver.NewRectangle(c2painter.PackRGBA(uint8((i*37)%255), uint8((i*59)%255), 180, 255))
			r.Move(c2fynedriver.Position{X: x, Y: y})
			r.Resize(c2fynedriver.Size{Width: 32, Height: 24})
			obj = r
		case 1:
			rr := c2fynedriver.NewRoundedRectangle(c2painter.PackRGBA(uint8((i*43)%255), 140, uint8((i*71)%255), 255), 4)
			rr.Move(c2fynedriver.Position{X: x, Y: y})
			rr.Resize(c2fynedriver.Size{Width: 32, Height: 24})
			rr.Elevation = 1
			rr.ShadowColor = c2painter.PackRGBA(0, 0, 0, 80)
			obj = rr
		case 2:
			c := c2fynedriver.NewCircle(c2painter.PackRGBA(220, uint8((i*29)%255), uint8((i*83)%255), 255))
			c.Move(c2fynedriver.Position{X: x, Y: y})
			c.Resize(c2fynedriver.Size{Width: 24, Height: 24})
			obj = c
		case 3:
			l := c2fynedriver.NewLabel(fmt.Sprintf("W%03d", i))
			l.Move(c2fynedriver.Position{X: x, Y: y})
			l.TextSize = 10
			l.Color = c2painter.PackRGBA(200, 220, 240, 255)
			obj = l
		case 4:
			b := c2fynedriver.NewBadge(fmt.Sprintf("B%d", i%10), c2painter.PackRGBA(40, 120, 200, 255), c2painter.PackRGBA(255, 255, 255, 255))
			b.Move(c2fynedriver.Position{X: x, Y: y})
			b.Resize(c2fynedriver.Size{Width: 32, Height: 20})
			obj = b
		case 5:
			btn := c2fynedriver.NewButton("K", nil)
			btn.Move(c2fynedriver.Position{X: x, Y: y})
			btn.Resize(c2fynedriver.Size{Width: 30, Height: 22})
			obj = btn
		case 6:
			line := c2fynedriver.NewLine(
				c2fynedriver.Position{X: x, Y: y},
				c2fynedriver.Position{X: x + 30, Y: y + 20},
				c2painter.PackRGBA(245, 158, 11, 255),
				2,
			)
			obj = line
		}

		root.Add(obj)
		mutants = append(mutants, obj)
	}

	return root, termComp, mutants
}

// helperMutateWidgets applique des transformations géométriques et visuelles aux widgets.
func helperMutateWidgets(mutants []c2fynedriver.CanvasObject, frame int) {
	shift := frame % 255
	for i, obj := range mutants {
		switch elem := obj.(type) {
		case *c2fynedriver.Rectangle:
			elem.SetFillColor(c2painter.PackRGBA(uint8((shift+i*13)%255), uint8((i*41)%255), 180, 255))
		case *c2fynedriver.RoundedRectangle:
			elem.SetFillColor(c2painter.PackRGBA(uint8((i*23)%255), uint8((shift+i*17)%255), 200, 255))
		case *c2fynedriver.Circle:
			elem.SetFillColor(c2painter.PackRGBA(240, uint8((shift+i*7)%255), uint8((i*31)%255), 255))
		case *c2fynedriver.Label:
			if frame%10 == 0 {
				elem.SetText(labelCacheTable[(i+frame)%1000])
			}
		case *c2fynedriver.Badge:
			if frame%5 == 0 {
				elem.SetBgColor(c2painter.PackRGBA(uint8((shift+i*19)%255), 80, 160, 255))
			}
		case *c2fynedriver.Button:
			elem.SetBgColor(c2painter.PackRGBA(30, uint8((shift+i*11)%255), 220, 255))
		case *c2fynedriver.Line:
			elem.SetStrokeColor(c2painter.PackRGBA(uint8((shift*3)%255), 200, 100, 255))
		}
	}
}

var labelCacheTable [1000]string

func init() {
	for i := 0; i < 1000; i++ {
		labelCacheTable[i] = fmt.Sprintf("M%03d", i)
	}
}

// BenchmarkSustainedLoad_100kFrames effectue le rendu continu sous charge soutenue avec 500 mutants et terminal.
func BenchmarkSustainedLoad_100kFrames(b *testing.B) {
	drv, err := c2fynedriver.NewHeadlessDriver(1280, 800)
	if err != nil {
		b.Fatalf("Erreur NewHeadlessDriver: %v", err)
	}
	defer drv.Quit()

	win, err := drv.CreateWindow("Sustained Load Bench", 1280, 800)
	if err != nil {
		b.Fatalf("Erreur CreateWindow: %v", err)
	}
	defer win.Close()

	root, termComp, mutants := helperBuildDenseUI(drv, 500)
	win.SetContent(root)

	// Écriture du profil goroutines à la première invocation
	if gFile, err := os.Create("/tmp/fyne55_goroutine.pprof"); err == nil {
		if gProfile := pprof.Lookup("goroutine"); gProfile != nil {
			_ = gProfile.WriteTo(gFile, 0)
		}
		_ = gFile.Close()
	}

	b.ReportAllocs()
	b.ResetTimer()

	termMsg := []byte("\x1b[32m[FRAME]\x1b[0m Tick\r\n")
	for i := 0; i < b.N; i++ {
		if i%10 == 0 {
			_, _ = termComp.Term.Write(termMsg)
		}
		helperMutateWidgets(mutants, i)
		win.RenderFrame()
	}
}

// BenchmarkWidgetScale_100_to_50k mesure les performances de rendu selon le nombre de widgets (100 à 50 000).
func BenchmarkWidgetScale_100_to_50k(b *testing.B) {
	scales := []int{100, 500, 1000, 5000, 10000, 50000}

	for _, count := range scales {
		b.Run(fmt.Sprintf("%d_Widgets", count), func(subB *testing.B) {
			surf := c2painter.NewSurface(1920, 1080)
			paint := c2painter.NewPainter(surf)

			root := c2fynedriver.NewContainer()
			cols := 100
			if count < 100 {
				cols = 10
			}

			for i := 0; i < count; i++ {
				row := i / cols
				col := i % cols
				x := col * 18
				y := row * 18

				mod := i % 6
				switch mod {
				case 0:
					r := c2fynedriver.NewRectangle(c2painter.PackRGBA(59, 130, 246, 255))
					r.Move(c2fynedriver.Position{X: x, Y: y})
					r.Resize(c2fynedriver.Size{Width: 16, Height: 16})
					root.Add(r)
				case 1:
					rr := c2fynedriver.NewRoundedRectangle(c2painter.PackRGBA(16, 185, 129, 255), 3)
					rr.Move(c2fynedriver.Position{X: x, Y: y})
					rr.Resize(c2fynedriver.Size{Width: 16, Height: 16})
					root.Add(rr)
				case 2:
					c := c2fynedriver.NewCircle(c2painter.PackRGBA(239, 68, 68, 255))
					c.Move(c2fynedriver.Position{X: x, Y: y})
					c.Resize(c2fynedriver.Size{Width: 16, Height: 16})
					root.Add(c)
				case 3:
					line := c2fynedriver.NewLine(
						c2fynedriver.Position{X: x, Y: y},
						c2fynedriver.Position{X: x + 16, Y: y + 16},
						c2painter.PackRGBA(234, 179, 8, 255),
						1,
					)
					root.Add(line)
				case 4:
					lbl := c2fynedriver.NewLabel("A")
					lbl.Move(c2fynedriver.Position{X: x, Y: y})
					lbl.TextSize = 10
					lbl.Color = c2painter.PackRGBA(255, 255, 255, 255)
					root.Add(lbl)
				case 5:
					badge := c2fynedriver.NewBadge(".", c2painter.PackRGBA(168, 85, 247, 255), c2painter.PackRGBA(255, 255, 255, 255))
					badge.Move(c2fynedriver.Position{X: x, Y: y})
					badge.Resize(c2fynedriver.Size{Width: 16, Height: 16})
					root.Add(badge)
				}
			}

			canvas := c2fynedriver.NewCanvas(nil, 1920, 1080)
			canvas.SetContent(root)

			subB.ReportAllocs()
			subB.ResetTimer()

			for i := 0; i < subB.N; i++ {
				canvas.Paint(paint)
			}
		})
	}
}

// BenchmarkMemoryStability_LongRun surveille l'empreinte mémoire avant/après 1 000 000 mutations d'éléments.
func BenchmarkMemoryStability_LongRun(b *testing.B) {
	drv, err := c2fynedriver.NewHeadlessDriver(1024, 768)
	if err != nil {
		b.Fatalf("Erreur NewHeadlessDriver: %v", err)
	}
	defer drv.Quit()

	win, err := drv.CreateWindow("Memory Stability Bench", 1024, 768)
	if err != nil {
		b.Fatalf("Erreur CreateWindow: %v", err)
	}
	defer win.Close()

	root, termComp, mutants := helperBuildDenseUI(drv, 500)
	win.SetContent(root)

	// Stabilisation GC initiale
	runtime.GC()
	var msBefore runtime.MemStats
	runtime.ReadMemStats(&msBefore)

	const totalMutations = 1000000
	const batchSize = 500 // 2000 passes de 500 mutations = 1 000 000 mutations

	b.ReportAllocs()
	b.ResetTimer()

	termMsg := []byte("\x1b[33m[MEMTEST]\x1b[0m Mutation en cours...\r\n")

	for i := 0; i < totalMutations/batchSize; i++ {
		if i%100 == 0 {
			_, _ = termComp.Term.Write(termMsg)
		}
		helperMutateWidgets(mutants, i)
		win.RenderFrame()
	}

	b.StopTimer()

	runtime.GC()
	var msAfter runtime.MemStats
	runtime.ReadMemStats(&msAfter)

	heapAllocDelta := int64(msAfter.HeapAlloc) - int64(msBefore.HeapAlloc)
	heapObjectsDelta := int64(msAfter.HeapObjects) - int64(msBefore.HeapObjects)
	totalAllocDelta := int64(msAfter.TotalAlloc) - int64(msBefore.TotalAlloc)

	b.ReportMetric(float64(heapAllocDelta)/1024.0, "heap_delta_KB")
	b.ReportMetric(float64(heapObjectsDelta), "heap_obj_delta")
	b.ReportMetric(float64(totalAllocDelta)/(1024.0*1024.0), "total_alloc_MB")

	b.Logf("--- BILAN STABILITÉ MÉMOIRE (1 000 000 mutations) ---")
	b.Logf("HeapAlloc Avant: %d Ko | Après: %d Ko | Delta: %+d Ko", msBefore.HeapAlloc/1024, msAfter.HeapAlloc/1024, heapAllocDelta/1024)
	b.Logf("HeapObjects Avant: %d | Après: %d | Delta: %+d", msBefore.HeapObjects, msAfter.HeapObjects, heapObjectsDelta)
	b.Logf("HeapInuse Avant: %d Ko | Après: %d Ko", msBefore.HeapInuse/1024, msAfter.HeapInuse/1024)
	b.Logf("TotalAlloc cumulé: %.2f Mo", float64(totalAllocDelta)/(1024.0*1024.0))
	b.Logf("Nombre de GC déclenchés: %d", msAfter.NumGC-msBefore.NumGC)
}
