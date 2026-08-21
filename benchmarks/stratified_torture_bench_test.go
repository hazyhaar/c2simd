package benchmarks

import (
	"fmt"
	"runtime"
	"runtime/metrics"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	c2display "code.hazyhaar.fr/devhoros/pkg/c2display"
	c2fynedriver "code.hazyhaar.fr/devhoros/pkg/c2fynedriver"
	c2fyneterm "code.hazyhaar.fr/devhoros/pkg/c2fyneterm"
	c2grid "code.hazyhaar.fr/devhoros/c2simd/pkg/c2grid"
	c2painter "code.hazyhaar.fr/devhoros/pkg/c2painter"
)

// =============================================================================
// STRUCTURES DE TÉLÉMÉTRIE STRATIFIÉE MULTIDIMENSIONNELLE
// =============================================================================

// StratifiedMetrics regroupe les métriques physiques captées pour une couche ou épreuve.
type StratifiedMetrics struct {
	Name            string
	Duration        time.Duration
	Ops             int64
	NsPerOp         float64
	MpixelsPerSec   float64
	MBPerSec        float64
	BytesPerOp      int64
	AllocsPerOp     int64
	GCCycles        uint32
	GCPauseTotalNs  uint64
	MutexWaitSec    float64
	PeakHeapAllocMB float64
}

func (m StratifiedMetrics) String() string {
	return fmt.Sprintf(
		"[%-28s] %10.1f ns/op | %8.1f MPix/s | %8.1f MB/s | %6d B/op | %3d allocs | GC:%2d (%5.1fµs) | MutexWait:%6.3fms | Heap:%6.1fMB",
		m.Name, m.NsPerOp, m.MpixelsPerSec, m.MBPerSec, m.BytesPerOp, m.AllocsPerOp,
		m.GCCycles, float64(m.GCPauseTotalNs)/1000.0, m.MutexWaitSec*1000.0, m.PeakHeapAllocMB,
	)
}

// measureMetrics exécute une fonction sur N itérations et capture l'ensemble des dimensions physiques.
func measureMetrics(name string, iterations int64, pixelsPerOp int64, bytesPerOp int64, fn func(i int64)) StratifiedMetrics {
	runtime.GC()
	var mBefore runtime.MemStats
	runtime.ReadMemStats(&mBefore)

	// Capture métriques runtime / mutex
	sampleMutex := []metrics.Sample{{Name: "/sync/mutex/wait/total:seconds"}}
	metrics.Read(sampleMutex)
	var mutexWaitBefore float64
	if sampleMutex[0].Value.Kind() == metrics.KindFloat64 {
		mutexWaitBefore = sampleMutex[0].Value.Float64()
	}

	start := time.Now()
	for i := int64(0); i < iterations; i++ {
		fn(i)
	}
	elapsed := time.Since(start)

	var mAfter runtime.MemStats
	runtime.ReadMemStats(&mAfter)

	metrics.Read(sampleMutex)
	var mutexWaitAfter float64
	if sampleMutex[0].Value.Kind() == metrics.KindFloat64 {
		mutexWaitAfter = sampleMutex[0].Value.Float64()
	}

	nsPerOp := float64(elapsed.Nanoseconds()) / float64(iterations)
	var mpixPerSec float64
	if pixelsPerOp > 0 && elapsed.Seconds() > 0 {
		mpixPerSec = (float64(pixelsPerOp*iterations) / 1e6) / elapsed.Seconds()
	}
	var mbPerSec float64
	if bytesPerOp > 0 && elapsed.Seconds() > 0 {
		mbPerSec = (float64(bytesPerOp*iterations) / (1024 * 1024)) / elapsed.Seconds()
	}

	bPerOp := int64(0)
	if mAfter.TotalAlloc > mBefore.TotalAlloc {
		bPerOp = int64(mAfter.TotalAlloc-mBefore.TotalAlloc) / iterations
	}
	allocsPerOp := int64(0)
	if mAfter.Mallocs > mBefore.Mallocs {
		allocsPerOp = int64(mAfter.Mallocs-mBefore.Mallocs) / iterations
	}

	return StratifiedMetrics{
		Name:            name,
		Duration:        elapsed,
		Ops:             iterations,
		NsPerOp:         nsPerOp,
		MpixelsPerSec:   mpixPerSec,
		MBPerSec:        mbPerSec,
		BytesPerOp:      bPerOp,
		AllocsPerOp:     allocsPerOp,
		GCCycles:        mAfter.NumGC - mBefore.NumGC,
		GCPauseTotalNs:  mAfter.PauseTotalNs - mBefore.PauseTotalNs,
		MutexWaitSec:    mutexWaitAfter - mutexWaitBefore,
		PeakHeapAllocMB: float64(mAfter.HeapAlloc) / (1024 * 1024),
	}
}

// =============================================================================
// MODULE 1 : PROBES STRATIFIÉES PAR COUCHE LOGICIELLE
// =============================================================================

func TestStratified_LayerBreakdown(t *testing.T) {
	t.Log("====================================================================================================================")
	t.Log("PROBES STRATIFIÉES : PROFILAGE FIN DES 5 COUCHES DU MOTEUR GRAPHIQUE")
	t.Log("====================================================================================================================")

	var results []StratifiedMetrics

	// -------------------------------------------------------------------------
	// Couche 1 : Algèbre & Rastérisation Bas-Niveau (c2painter)
	// -------------------------------------------------------------------------
	surf1080p := c2painter.NewSurface(1920, 1080)
	p := c2painter.NewPainter(surf1080p)
	solidCol := c2painter.PackRGBA(15, 23, 42, 255)
	alphaCol := c2painter.PackRGBA(239, 68, 68, 128)

	// 1.1 Clear 1080p (Memmove)
	m1 := measureMetrics("Couche 1.1: Clear (1080p)", 100, 1920*1080, 1920*1080*4, func(i int64) {
		p.Clear(solidCol)
	})
	results = append(results, m1)

	// 1.2 FillRect Solid 500x500
	m2 := measureMetrics("Couche 1.2: FillRect Solid 500x500", 500, 500*500, 500*500*4, func(i int64) {
		p.DrawRect(100, 100, 500, 500, solidCol)
	})
	results = append(results, m2)

	// 1.3 FillRect Alpha Blended 500x500 (Porter-Duff)
	m3 := measureMetrics("Couche 1.3: FillRect Alpha 500x500", 200, 500*500, 500*500*4, func(i int64) {
		p.DrawRect(100, 100, 500, 500, alphaCol)
	})
	results = append(results, m3)

	// 1.4 FillRoundedRect Subpixel Antialiased (R=32, 500x300)
	m4 := measureMetrics("Couche 1.4: RoundedRect R32 500x300", 200, 500*300, 500*300*4, func(i int64) {
		p.DrawRoundedRect(100, 100, 500, 300, 32, alphaCol)
	})
	results = append(results, m4)

	// 1.5 FillCircle Subpixel (R=120)
	m5 := measureMetrics("Couche 1.5: FillCircle R120 (45k px)", 500, 45239, 180956, func(i int64) {
		p.DrawCircle(500, 500, 120, alphaCol)
	})
	results = append(results, m5)

	// 1.6 DrawLine Diagonal (2000 px, width=4)
	m6 := measureMetrics("Couche 1.6: DrawLine Diag (2000px, w4)", 500, 2000*4, 2000*4*4, func(i int64) {
		p.DrawLine(50, 50, 1850, 950, 4, solidCol)
	})
	results = append(results, m6)

	// -------------------------------------------------------------------------
	// Couche 2 : Typographie & Rendu de Glyphes (tt55 / FontManager)
	// -------------------------------------------------------------------------
	// 2.1 Mesure et Rendu de chaîne courte (Cache Hit)
	sampleText := "Antigravity Sovereign Fyne55 Vector Engine - 1234567890"
	m7 := measureMetrics("Couche 2.1: DrawString 54 chars (Hit)", 1000, int64(len(sampleText)*8*16), int64(len(sampleText)), func(i int64) {
		c2fynedriver.DrawString(p, sampleText, 50, 50, 14, solidCol)
	})
	results = append(results, m7)

	// 2.2 Mesure de chaîne
	m8 := measureMetrics("Couche 2.2: MeasureString 54 chars", 5000, 0, 0, func(i int64) {
		_, _ = c2fynedriver.MeasureString(sampleText, 14)
	})
	results = append(results, m8)

	// -------------------------------------------------------------------------
	// Couche 3 : Grille Matricielle & Parseur ANSI (c2fyneterm / c2grid)
	// -------------------------------------------------------------------------
	term := c2fyneterm.NewTerminalWidget(120, 40)
	ansiPayload := []byte("\x1b[38;2;34;197;94m[OK]\x1b[0m Compilation CGO-Free sgoiter passée avec succès\r\n")

	// 3.1 Ingestion ANSI Feed (120x40 grid)
	m9 := measureMetrics("Couche 3.1: ANSI Stream Feed", 2000, int64(120*40), int64(len(ansiPayload)), func(i int64) {
		term.Feed(ansiPayload)
	})
	results = append(results, m9)

	// 3.2 Rasterisation Terminal Grille Complète (120x40 = 4800 cellules -> 960x640 pixels)
	termSurf := c2painter.NewSurface(960, 640)
	m10 := measureMetrics("Couche 3.2: Terminal RenderToSurface", 100, 960*640, 960*640*4, func(i int64) {
		term.RenderToSurface(termSurf)
	})
	results = append(results, m10)

	// -------------------------------------------------------------------------
	// Couche 4 : Graphe de Scène & Traversée (Canvas.Walk / Paint)
	// -------------------------------------------------------------------------
	drv, _ := c2fynedriver.NewHeadlessDriver(1280, 800)
	win, _ := drv.CreateWindow("Stratified Bench", 1280, 800)
	root, _, mutants := helperBuildDenseUI(drv, 500)
	win.SetContent(root)

	// 4.1 Mutation d'arbre (500 widgets)
	m11 := measureMetrics("Couche 4.1: Widget Mutations (500)", 1000, 500, 0, func(i int64) {
		helperMutateWidgets(mutants, int(i))
	})
	results = append(results, m11)

	// 4.2 Traversée d'arbre Walk (500 widgets)
	canvas := win.Canvas()
	m12 := measureMetrics("Couche 4.2: Canvas.Walk (500 widgets)", 500, 500, 0, func(i int64) {
		canvas.Walk(func(obj c2fynedriver.CanvasObject, pos c2fynedriver.Position, size c2fynedriver.Size) bool {
			return true
		})
	})
	results = append(results, m12)

	// 4.3 Rendu Complet Canvas.Paint (500 widgets)
	m13 := measureMetrics("Couche 4.3: Canvas.Paint (500 widgets)", 100, 1280*800, 1280*800*4, func(i int64) {
		win.RenderFrame()
	})
	results = append(results, m13)

	// -------------------------------------------------------------------------
	// Couche 5 : IPC Wire & Présentation (HeadlessWindow.Present)
	// -------------------------------------------------------------------------
	hDisp, _ := c2display.NewHeadlessDisplay(c2display.DisplayOptions{})
	hWin, _ := hDisp.CreateWindow(c2display.WindowOptions{Width: 1920, Height: 1080})
	m14 := measureMetrics("Couche 5.1: Headless Present 1080p", 200, 1920*1080, 1920*1080*4, func(i int64) {
		_ = hWin.Present(surf1080p)
	})
	results = append(results, m14)

	// Affichage du tableau de bord stratifié
	for _, res := range results {
		t.Log(res.String())
	}
	t.Log("====================================================================================================================")
}

// =============================================================================
// MODULE 2 : MATRICE MULTIDIMENSIONNELLE (CPU, RAM, CONCURRENCE & CONTENTION)
// =============================================================================

func TestStratified_ConcurrencyAndContention(t *testing.T) {
	t.Log("====================================================================================================================")
	t.Log("MATRICE MULTIDIMENSIONNELLE : MONTÉE EN CONCURRENCE & CONTENTION DE VERROUS (1 -> 32 GOROUTINES)")
	t.Log("====================================================================================================================")

	threadCounts := []int{1, 2, 4, 8, 16, 32}

	for _, threads := range threadCounts {
		term := c2fyneterm.NewTerminalWidget(120, 40)
		opsPerThread := int64(100)
		totalOps := int64(threads) * opsPerThread

		name := fmt.Sprintf("Contention Terminal (%2d threads)", threads)
		m := measureMetrics(name, totalOps, 0, 0, func(i int64) {
			// Cette fonction n'est appelée qu'une fois pour la mesure globale
		})

		// Exécution multithreadée réelle
		var wg sync.WaitGroup
		start := time.Now()

		sampleMutex := []metrics.Sample{{Name: "/sync/mutex/wait/total:seconds"}}
		metrics.Read(sampleMutex)
		var mutexWaitBefore float64
		if sampleMutex[0].Value.Kind() == metrics.KindFloat64 {
			mutexWaitBefore = sampleMutex[0].Value.Float64()
		}

		for th := 0; th < threads; th++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				localSurf := c2painter.NewSurface(960, 640)
				data := []byte(fmt.Sprintf("\x1b[32mThread-%02d stream line data\x1b[0m\r\n", id))
				for op := int64(0); op < opsPerThread; op++ {
					if op%5 == 0 {
						term.RenderToSurface(localSurf)
					} else if op%13 == 0 {
						term.Resize(120, 40)
					} else if op%7 == 0 {
						term.ClearSelection()
					} else {
						term.Feed(data)
					}
				}
			}(th)
		}
		wg.Wait()
		elapsed := time.Since(start)

		metrics.Read(sampleMutex)
		var mutexWaitAfter float64
		if sampleMutex[0].Value.Kind() == metrics.KindFloat64 {
			mutexWaitAfter = sampleMutex[0].Value.Float64()
		}

		m.Duration = elapsed
		m.NsPerOp = float64(elapsed.Nanoseconds()) / float64(totalOps)
		m.Ops = totalOps
		m.MutexWaitSec = mutexWaitAfter - mutexWaitBefore

		t.Log(m.String())
	}
	t.Log("====================================================================================================================")
}

// =============================================================================
// MODULE 3 : CRASH TESTING & LIMITES DE RUPTURE / CONFINEMENT ARCHITECTURAL
// =============================================================================

// TestCrashLimit_ResolutionStress teste la saturation jusqu'à la limite mémoire 32K pixels.
func TestCrashLimit_ResolutionStress(t *testing.T) {
	t.Log("====================================================================================================================")
	t.Log("CRASH TEST 1 : SATURATION DE RÉSOLUTION (1080p -> 4K -> 8K -> 16K -> 32K)")
	t.Log("====================================================================================================================")

	resolutions := []struct {
		Name   string
		Width  int
		Height int
	}{
		{"1080p Full HD", 1920, 1080},
		{"4K Ultra HD", 3840, 2160},
		{"8K Super Hi-Vision", 7680, 4320},
		{"16K Extreme Canvas", 15360, 8640},
		{"32K Boundary Canvas", 30720, 17280}, // ~530 MégaPixels, ~2.12 Go par frame !
	}

	for _, res := range resolutions {
		mpix := float64(res.Width*res.Height) / 1e6
		rawMemMB := float64(res.Width*res.Height*4) / (1024 * 1024)

		start := time.Now()
		surf := c2painter.NewSurface(res.Width, res.Height)
		if surf == nil || len(surf.Pixels) == 0 {
			t.Fatalf("[RUPTURE ATTEINTE] Échec d'allocation pour %s (%dx%d, %.1f MB)", res.Name, res.Width, res.Height, rawMemMB)
		}
		allocTime := time.Since(start)

		p := c2painter.NewPainter(surf)
		clearStart := time.Now()
		p.Clear(c2painter.PackRGBA(15, 23, 42, 255))
		clearTime := time.Since(clearStart)

		// Dessin de primitives extrêmes aux 4 coins et au centre
		drawStart := time.Now()
		p.DrawRoundedRect(100, 100, res.Width-200, res.Height-200, 64, c2painter.PackRGBA(239, 68, 68, 200))
		p.DrawCircle(res.Width/2, res.Height/2, res.Height/4, c2painter.PackRGBA(59, 130, 246, 220))
		p.DrawLine(0, 0, res.Width, res.Height, 8, c2painter.PackRGBA(245, 158, 11, 255))
		drawTime := time.Since(drawStart)

		totalTime := time.Since(start)
		fpsEq := 1.0 / totalTime.Seconds()

		t.Logf(
			"[%-20s] %5dx%5d | %5.1f MP | Mem: %7.1f MB | Alloc: %8.2fms | Clear: %8.2fms | Draw: %8.2fms | Total: %8.2fms (%5.1f FPS)",
			res.Name, res.Width, res.Height, mpix, rawMemMB,
			float64(allocTime.Microseconds())/1000.0,
			float64(clearTime.Microseconds())/1000.0,
			float64(drawTime.Microseconds())/1000.0,
			float64(totalTime.Microseconds())/1000.0,
			fpsEq,
		)

		// Libération immédiate pour ne pas saturer la RAM de la machine hôte
		surf.Pixels = nil
		runtime.GC()
	}
	t.Log("====================================================================================================================")
}

// TestCrashLimit_WidgetAvalanche teste la saturation d'arbre de widgets jusqu'à 250 000 widgets.
func TestCrashLimit_WidgetAvalanche(t *testing.T) {
	t.Log("====================================================================================================================")
	t.Log("CRASH TEST 2 : AVALANCHE D'ARBRE DE WIDGETS (1 000 -> 10 000 -> 50 000 -> 100 000 -> 250 000)")
	t.Log("====================================================================================================================")

	counts := []int{1000, 10000, 50000, 100000, 250000}

	for _, count := range counts {
		drv, _ := c2fynedriver.NewHeadlessDriver(1920, 1080)
		win, _ := drv.CreateWindow(fmt.Sprintf("Avalanche %d", count), 1920, 1080)

		startBuild := time.Now()
		root, _, mutants := helperBuildDenseUI(drv, count)
		win.SetContent(root)
		buildTime := time.Since(startBuild)

		// Mutation complète de tous les widgets
		startMutate := time.Now()
		helperMutateWidgets(mutants, 42)
		mutateTime := time.Since(startMutate)

		// Traversée Walk
		startWalk := time.Now()
		visited := 0
		win.Canvas().Walk(func(obj c2fynedriver.CanvasObject, pos c2fynedriver.Position, size c2fynedriver.Size) bool {
			visited++
			return true
		})
		walkTime := time.Since(startWalk)

		// Rendu Paint
		startPaint := time.Now()
		win.RenderFrame()
		paintTime := time.Since(startPaint)

		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		t.Logf(
			"[%6d Widgets] Noeuds: %6d | Heap: %5.1f MB | Build: %7.2fms | Mutate: %6.2fms | Walk: %6.2fms | Paint: %7.2fms | Cadence: %5.1f FPS",
			count, visited, float64(m.HeapAlloc)/(1024*1024),
			float64(buildTime.Microseconds())/1000.0,
			float64(mutateTime.Microseconds())/1000.0,
			float64(walkTime.Microseconds())/1000.0,
			float64(paintTime.Microseconds())/1000.0,
			1.0/paintTime.Seconds(),
		)

		win.Close()
		runtime.GC()
	}
	t.Log("====================================================================================================================")
}

// TestCrashLimit_ScrollbackStress teste la saturation d'historique terminal jusqu'à 5 000 000 de lignes.
func TestCrashLimit_ScrollbackStress(t *testing.T) {
	t.Log("====================================================================================================================")
	t.Log("CRASH TEST 3 : SATURATION HISTORIQUE SCROLLBACK (10k -> 100k -> 1M -> 5M LIGNES)")
	t.Log("====================================================================================================================")

	capacities := []int{10000, 100000, 1000000, 5000000}

	for _, capLines := range capacities {
		ring := c2grid.NewScrollbackRing(capLines)
		cellLine := make([]c2grid.C2_cell_t, 120)
		for i := range cellLine {
			cellLine[i] = c2grid.C2_cell_t{Rune: 'A' + uint32(i%26), Fg: 15, Bg: 0, Width: 1}
		}

		start := time.Now()
		// Remplissage complet du scrollback
		for i := 0; i < capLines; i++ {
			ring.Push(cellLine)
		}
		elapsed := time.Since(start)

		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		linesPerSec := float64(capLines) / elapsed.Seconds()

		t.Logf(
			"[%7d Lignes] Capacité: %7d | Ingestion: %7.2fms (%8.0f lig/s) | Heap: %6.1f MB | Tampon Cellules: %5.1f MB",
			capLines, ring.Count(),
			float64(elapsed.Microseconds())/1000.0,
			linesPerSec,
			float64(m.HeapAlloc)/(1024*1024),
			float64(capLines*120*8)/(1024*1024),
		)

		runtime.GC()
	}
	t.Log("====================================================================================================================")
}

// TestCrashLimit_MassiveConcurrencyStorm teste la saturation extrême sous 500 goroutines concurrentes.
func TestCrashLimit_MassiveConcurrencyStorm(t *testing.T) {
	t.Log("====================================================================================================================")
	t.Log("CRASH TEST 4 : OURAGAN DE CONCURRENCE MASSIF (500 GOROUTINES ACTIVES SIMULTANÉES)")
	t.Log("====================================================================================================================")

	drv, _ := c2fynedriver.NewHeadlessDriver(1920, 1080)
	win, _ := drv.CreateWindow("Storm Win", 1920, 1080)
	root, _, mutants := helperBuildDenseUI(drv, 1000)
	win.SetContent(root)

	term := c2fyneterm.NewTerminalWidget(120, 40)

	goroutines := 500
	_ = goroutines
	duration := 2 * time.Second
	stopChan := make(chan struct{})

	var opsCompleted uint64
	var wg sync.WaitGroup

	start := time.Now()

	// 100 goroutines de streaming ANSI PTY
	for g := 0; g < 100; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			payload := []byte(fmt.Sprintf("\x1b[32mGoroutine-%03d ANSI storm output\x1b[0m\r\n", id))
			for {
				select {
				case <-stopChan:
					return
				default:
					term.Feed(payload)
					atomic.AddUint64(&opsCompleted, 1)
					runtime.Gosched()
				}
			}
		}(g)
	}

	// 100 goroutines de mutations de widgets
	for g := 0; g < 100; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			frame := 0
			for {
				select {
				case <-stopChan:
					return
				default:
					helperMutateWidgets(mutants, frame)
					frame++
					atomic.AddUint64(&opsCompleted, 1)
					runtime.Gosched()
				}
			}
		}()
	}

	// 100 goroutines de rendu graphique Canvas.Paint
	for g := 0; g < 100; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stopChan:
					return
				default:
					win.RenderFrame()
					atomic.AddUint64(&opsCompleted, 1)
					runtime.Gosched()
				}
			}
		}()
	}

	// 100 goroutines de rendu Terminal RenderToSurface
	for g := 0; g < 100; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			localSurf := c2painter.NewSurface(960, 640)
			for {
				select {
				case <-stopChan:
					return
				default:
					term.RenderToSurface(localSurf)
					atomic.AddUint64(&opsCompleted, 1)
					runtime.Gosched()
				}
			}
		}()
	}

	// 100 goroutines de redimensionnement dynamique
	for g := 0; g < 100; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			w := 800 + (id % 400)
			h := 600 + (id % 300)
			for {
				select {
				case <-stopChan:
					return
				default:
					win.Resize(c2fynedriver.Size{Width: w, Height: h})
					term.Resize(80+(id%40), 24+(id%20))
					term.ClearSelection()
					atomic.AddUint64(&opsCompleted, 1)
					time.Sleep(100 * time.Microsecond)
				}
			}
		}(g)
	}

	time.Sleep(duration)
	close(stopChan)
	wg.Wait()
	totalTime := time.Since(start)

	totalOps := atomic.LoadUint64(&opsCompleted)
	opsPerSec := float64(totalOps) / totalTime.Seconds()

	t.Logf(
		"[RÉSULTAT OURAGAN] 500 goroutines simultanées : %d opérations concurrentes exécutées en %.2fs (Débit: %.0f ops/s) | 0 Panic | 0 Deadlock",
		totalOps, totalTime.Seconds(), opsPerSec,
	)
	t.Log("====================================================================================================================")
}
