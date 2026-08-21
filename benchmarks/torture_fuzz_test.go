package benchmarks_test

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	c2display "code.hazyhaar.fr/devhoros/pkg/c2display"
	c2fynedriver "code.hazyhaar.fr/devhoros/pkg/c2fynedriver"
	c2fyneterm "code.hazyhaar.fr/devhoros/pkg/c2fyneterm"
	c2painter "code.hazyhaar.fr/devhoros/pkg/c2painter"
	tt55 "code.hazyhaar.fr/devhoros/pkg/tt55"
)

// =============================================================================
// 1. Banc de Torture & Fuzzing : Géométrie Extrême (FuzzExtremeGeometry)
// =============================================================================

func FuzzExtremeGeometry(f *testing.F) {
	// Corpus initial diversifié
	f.Add(0, 0, 100, 100, uint8(255), uint8(0), uint8(0), uint8(255), 10, 2)
	f.Add(-500, -500, 100000, 100000, uint8(0), uint8(255), uint8(0), uint8(128), 50000, 100)
	f.Add(math.MinInt32, math.MaxInt32, math.MinInt32, math.MaxInt32, uint8(1), uint8(1), uint8(1), uint8(1), -10, -5)
	f.Add(0, 0, 0, 0, uint8(0), uint8(0), uint8(0), uint8(0), 0, 0)
	f.Add(1000, 1000, 1, 1, uint8(255), uint8(255), uint8(255), uint8(255), 100, 50)
	f.Add(math.MaxInt32/2, math.MaxInt32/2, 200, 200, uint8(50), uint8(100), uint8(150), uint8(200), 20, 4)

	f.Fuzz(func(t *testing.T, x, y, w, h int, r, g, b, a uint8, radius, stroke int) {
		surf := c2painter.NewSurface(320, 240)
		if surf == nil || len(surf.Pixels) == 0 {
			t.Fatalf("échec d'allocation surface")
		}
		p := c2painter.NewPainter(surf)
		col := c2painter.RGBA(r, g, b, a)

		// Épreuve de toutes les primitives sous valeurs brutes
		p.DrawRect(x, y, w, h, col)
		p.StrokeRect(x, y, w, h, stroke, col)
		p.DrawRoundedRect(x, y, w, h, radius, col)
		p.StrokeRoundedRect(x, y, w, h, radius, stroke, col)
		p.DrawCircle(x, y, radius, col)
		p.StrokeCircle(x, y, radius, stroke, col)
		p.DrawEllipse(x, y, w, h, col)
		p.StrokeEllipse(x, y, w, h, stroke, col)
		p.DrawLine(x, y, x+w, y+h, stroke, col)
		p.DrawLinearGradient(x, y, w, h, col, c2painter.RGBA(255-r, 255-g, 255-b, a), true)

		// Test de découpage arbitraire
		p.SetClip(x, y, w, h)
		p.DrawRect(0, 0, 320, 240, col)
		p.ResetClip()
	})
}

func TestTortureExtremeGeometry(t *testing.T) {
	t.Run("DegenerateCoordinatesAndDimensions", func(t *testing.T) {
		surf := c2painter.NewSurface(640, 480)
		p := c2painter.NewPainter(surf)
		col := c2painter.RGBA(200, 100, 50, 200)

		extremes := []int{
			math.MinInt32, math.MinInt32 + 1, -1000000, -1, 0, 1, 1000000,
			math.MaxInt32 - 1, math.MaxInt32,
		}

		for _, x := range extremes {
			for _, y := range extremes {
				p.DrawRect(x, y, 100, 100, col)
				p.StrokeRect(x, y, 100, 100, 2, col)
				p.DrawRoundedRect(x, y, 100, 100, 10, col)
				p.StrokeRoundedRect(x, y, 100, 100, 10, 2, col)
				p.DrawCircle(x, y, 50, col)
				p.StrokeCircle(x, y, 50, 5, col)
				p.DrawLine(x, y, 100, 100, 2, col)
				p.DrawLine(100, 100, x, y, 2, col)
			}
		}
	})

	t.Run("GiantRectangles", func(t *testing.T) {
		resolutions := [][2]int{
			{10, 10},
			{320, 240},
			{1920, 1080},
			{3840, 2160},
		}

		for _, res := range resolutions {
			surf := c2painter.NewSurface(res[0], res[1])
			p := c2painter.NewPainter(surf)
			col := c2painter.RGBA(255, 0, 0, 128)

			// Rectangle 100 000 x 100 000 px
			p.DrawRect(-50000, -50000, 100000, 100000, col)
			p.StrokeRect(-50000, -50000, 100000, 100000, 100, col)
			p.DrawRoundedRect(-50000, -50000, 100000, 100000, 5000, col)

			// Rectangle 1 000 000 x 1 000 000 px
			p.DrawRect(-500000, -500000, 1000000, 1000000, col)
		}
	})

	t.Run("TenThousandOverlappingAlphaLayers", func(t *testing.T) {
		surf := c2painter.NewSurface(400, 300)
		p := c2painter.NewPainter(surf)
		p.Clear(c2painter.RGBA(0, 0, 0, 255))

		start := time.Now()
		const layers = 10000
		for i := 0; i < layers; i++ {
			alpha := uint8((i % 254) + 1)
			col := c2painter.RGBA(uint8(i%256), uint8((i*7)%256), uint8((i*13)%256), alpha)
			x := (i % 200)
			y := (i % 150)
			p.DrawRect(x, y, 200, 150, col)
		}
		dur := time.Since(start)

		// Vérification intégrité du tampon
		nonZero := 0
		for _, pix := range p.Pixels() {
			if pix != 0 {
				nonZero++
			}
		}
		if nonZero == 0 {
			t.Fatalf("tampon vide après 10 000 couches alpha")
		}
		t.Logf("10 000 couches alpha transparentes superposées en %v (%d/%d pixels affectés)",
			dur, nonZero, len(p.Pixels()))
	})

	t.Run("SubpixelAndExtremeThickness", func(t *testing.T) {
		surf := c2painter.NewSurface(500, 500)
		p := c2painter.NewPainter(surf)
		col := c2painter.RGBA(10, 200, 100, 255)

		// Cercles avec rayons dégénérés
		p.DrawCircle(250, 250, 0, col)
		p.DrawCircle(250, 250, -100, col)
		p.DrawCircle(250, 250, 1000000, col)
		p.StrokeCircle(250, 250, 100, 0, col)
		p.StrokeCircle(250, 250, 100, -50, col)
		p.StrokeCircle(250, 250, 100, 10000, col)

		// Lignes avec points identiques ou extrêmes
		p.DrawLine(250, 250, 250, 250, 10, col)
		p.DrawLine(math.MinInt32, math.MinInt32, math.MaxInt32, math.MaxInt32, 5, col)
		p.DrawLine(0, 0, 500, 500, 0, col)
		p.DrawLine(0, 0, 500, 500, -10, col)
		p.DrawLine(0, 0, 500, 500, 5000, col)

		// Rectangles arrondis avec rayons excessifs
		p.DrawRoundedRect(50, 50, 100, 100, 50000, col)
		p.DrawRoundedRect(50, 50, 100, 100, -500, col)
		p.StrokeRoundedRect(50, 50, 100, 100, 500, 200, col)

		// Ellipses avec rayons asymétriques extrêmes
		p.DrawEllipse(250, 250, 100000, 1, col)
		p.DrawEllipse(250, 250, 1, 100000, col)
		p.StrokeEllipse(250, 250, 100000, 1, 10, col)
	})

	t.Run("DegenerateClipBounds", func(t *testing.T) {
		surf := c2painter.NewSurface(400, 400)
		p := c2painter.NewPainter(surf)
		col := c2painter.RGBA(255, 255, 255, 255)

		clips := [][4]int{
			{-100, -100, 50, 50},
			{500, 500, 100, 100},
			{0, 0, 0, 0},
			{100, 100, -50, -50},
			{math.MinInt32, math.MinInt32, math.MaxInt32, math.MaxInt32},
			{0, 0, 10000, 10000},
		}

		for _, cl := range clips {
			p.SetClip(cl[0], cl[1], cl[2], cl[3])
			p.DrawRect(0, 0, 400, 400, col)
			p.DrawCircle(200, 200, 100, col)
			p.DrawLine(0, 0, 400, 400, 5, col)
			p.ResetClip()
		}
	})
}

// =============================================================================
// 2. Banc de Torture & Fuzzing : Tempête ANSI Terminal (FuzzTerminalAnsiStorm)
// =============================================================================

func FuzzTerminalAnsiStorm(f *testing.F) {
	// Corpus initial
	f.Add([]byte("\x1b[31;1mHello World\x1b[0m\r\n"))
	f.Add([]byte("\x1b]52;c;SGVsbG8gV29ybGQ=\x07"))
	f.Add([]byte("\x1b]52;c;INVALID_BASE64_PAYLOAD_NOT_TERMINATED"))
	f.Add([]byte("\x1b[?1049h\x1b[2J\x1b[HAlternate Screen\x1b[?1049l"))
	f.Add([]byte("\x00\x00\x00\x1b[\x00\x0031m\x00\x00\r\n\x00"))
	f.Add([]byte("\x1b(0lqqqqk\r\nx    x\r\nmqqqqj\x1b(B"))
	f.Add([]byte("\x1b[99999999999999999999;99999999999999999999H\x1b[38;2;300;400;500mOOB"))

	f.Fuzz(func(t *testing.T, payload []byte) {
		term := c2fyneterm.NewTerminalWidget(80, 24)
		if term == nil {
			t.Fatalf("échec d'instanciation TerminalWidget")
		}

		// Ingestion du flux
		_, _ = term.Write(payload)

		// Opérations de sélection et scrollback
		term.ScrollUp(5)
		term.ScrollDown(2)
		term.StartSelection(2, 3)
		term.UpdateSelection(40, 15)
		_ = term.GetSelectedText()
		_ = term.CopySelection()
		term.EndSelection()
		term.ClearSelection()

		// Redimensionnement dynamique
		term.Resize(40, 12)
		term.Resize(120, 40)

		// Rendu vers surface
		surf := c2painter.NewSurface(640, 480)
		term.RenderToSurface(surf)
	})
}

func TestTortureTerminalAnsiStorm(t *testing.T) {
	t.Run("MassiveMalformedByteStream", func(t *testing.T) {
		term := c2fyneterm.NewTerminalWidget(100, 30)

		// Génération de 2 Mo d'octets aléatoires avec séquences ANSI malformées
		rng := rand.New(rand.NewSource(42))
		buf := make([]byte, 2*1024*1024)
		for i := range buf {
			buf[i] = byte(rng.Intn(256))
		}

		start := time.Now()
		const chunkSize = 4096
		for i := 0; i < len(buf); i += chunkSize {
			end := i + chunkSize
			if end > len(buf) {
				end = len(buf)
			}
			_, err := term.Write(buf[i:end])
			if err != nil {
				t.Fatalf("erreur inattendue sur Write PTY: %v", err)
			}
		}
		dur := time.Since(start)
		t.Logf("Ingestion de 2 Mo de bruit ANSI malformé terminée en %v (sans crash)", dur)
	})

	t.Run("TruncatedOSCAndCSIPayloads", func(t *testing.T) {
		term := c2fyneterm.NewTerminalWidget(80, 24)

		truncatedPatterns := [][]byte{
			[]byte("\x1b"),
			[]byte("\x1b]"),
			[]byte("\x1b]52"),
			[]byte("\x1b]52;"),
			[]byte("\x1b]52;c;"),
			[]byte("\x1b]52;c;QUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFB"), // sans terminateur
			[]byte("\x1b]0;Titre Fenetre Incomplet"),
			[]byte("\x1b["),
			[]byte("\x1b[?"),
			[]byte("\x1b[?1049"),
			[]byte("\x1b[38;2;"),
			[]byte("\x1b[48;5;"),
			[]byte("\x1b[;"),
			[]byte("\x1b[;;;;;;;;;;;;;;m"),
			[]byte("\x1b[-10;-20H"),
			[]byte("\x1b[999999999999999999999999999999999999999999999999999m"),
		}

		for _, pat := range truncatedPatterns {
			_, _ = term.Write(pat)
			surf := c2painter.NewSurface(640, 384)
			term.RenderToSurface(surf)
		}
	})

	t.Run("CascadeOneMillionLines", func(t *testing.T) {
		term := c2fyneterm.NewTerminalWidget(80, 24)

		start := time.Now()
		// Injection répétée de cascades de sauts de ligne
		chunk := bytes.Repeat([]byte("Horos55 Ligne de test torture ANSI\r\n"), 1000)
		for i := 0; i < 1000; i++ {
			_, _ = term.Write(chunk)
		}
		dur := time.Since(start)

		t.Logf("1 000 000 de lignes ingérées et défilées en %v", dur)
	})

	t.Run("IllegalDECAndAlternateScreenSwitching", func(t *testing.T) {
		term := c2fyneterm.NewTerminalWidget(80, 24)

		for i := 0; i < 5000; i++ {
			// Bascule Alternate Screen ultra rapide
			_, _ = term.Write([]byte("\x1b[?1049h"))
			_, _ = term.Write([]byte("\x1b(0lqqqqk\r\nx    x\r\nmqqqqj\x1b(B"))
			_, _ = term.Write([]byte("\x1b[?1049l"))
		}
	})

	t.Run("ConcurrentStreamingAndResize", func(t *testing.T) {
		term := c2fyneterm.NewTerminalWidget(80, 24)
		var stop atomic.Bool

		var wg sync.WaitGroup
		// Goroutine de flux PTY continu
		for g := 0; g < 4; g++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				data := []byte(fmt.Sprintf("\x1b[3%dmThread %d : streaming de donnees continues...\r\n\x1b[0m", id%7+1, id))
				for !stop.Load() {
					_, _ = term.Write(data)
				}
			}(g)
		}

		// Goroutine de redimensionnement de grille concurrent
		wg.Add(1)
		go func() {
			defer wg.Done()
			sizes := [][2]int{
				{80, 24}, {120, 40}, {40, 10}, {200, 60}, {1, 1}, {500, 200}, {80, 24},
			}
			for i := 0; i < 200; i++ {
				sz := sizes[i%len(sizes)]
				term.Resize(sz[0], sz[1])
				time.Sleep(100 * time.Microsecond)
			}
		}()

		// Goroutine de rendu concurrent
		wg.Add(1)
		go func() {
			defer wg.Done()
			surf := c2painter.NewSurface(800, 600)
			for !stop.Load() {
				term.RenderToSurface(surf)
				time.Sleep(500 * time.Microsecond)
			}
		}()

		time.Sleep(100 * time.Millisecond)
		stop.Store(true)
		wg.Wait()
		t.Logf("Concurrent streaming et resize validés sans deadlock ni panic")
	})
}

// =============================================================================
// 3. Banc de Torture & Fuzzing : Polices Corrompues (FuzzFontCorruptedPayloads)
// =============================================================================

func FuzzFontCorruptedPayloads(f *testing.F) {
	// Corpus initial
	f.Add([]byte{})
	f.Add([]byte{0x00, 0x01, 0x00, 0x00})
	f.Add([]byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x04, 0x00, 0x40, 0x00, 0x02, 0x00, 0x20})
	f.Add([]byte("OTTO\x00\x02\x00\x20\x00\x01\x00\x00"))
	f.Add([]byte("true\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"))

	f.Fuzz(func(t *testing.T, data []byte) {
		var font tt55.Tt55_font
		ret := tt55.Tt55_open(data, uint64(len(data)), &font)
		if ret == 0 {
			// Si le parseur accepte le format, tester la résolution de glyphes
			var gid uint16
			var aw uint16
			for cp := rune(0); cp < 256; cp++ {
				_ = tt55.Tt55_glyph(&font, uint32(cp), &gid)
				_ = tt55.Tt55_advance(&font, gid, &aw)
			}
		}

		// Injection dans FontManager
		fm := c2fynedriver.DefaultFontManager()
		if ret == 0 {
			fm.SetTTFont(&font)
		}

		// Requêtes de masques de glyphes avec tailles variées
		for _, sz := range []int{-5, 0, 1, 14, 72, 1000} {
			_ = fm.GetGlyphMask('A', sz)
			_ = fm.GetGlyphMask(rune(0x10FFFF), sz)
			_ = fm.GetGlyphMask(rune(-1), sz)
		}
	})
}

func TestTortureFontCorruptedPayloads(t *testing.T) {
	t.Run("TruncatedTrueTypeHeaders", func(t *testing.T) {
		for l := 0; l <= 12; l++ {
			payload := make([]byte, l)
			var font tt55.Tt55_font
			ret := tt55.Tt55_open(payload, uint64(l), &font)
			if ret == 0 {
				t.Fatalf("Tt55_open n'aurait pas du accepter un payload tronque de %d octets", l)
			}
		}
	})

	t.Run("FalsifiedTableDirectoryOffsets", func(t *testing.T) {
		// En-tête TTF valide revendiquant 4 tables, mais tables pointant hors limites
		buf := make([]byte, 12+4*16)
		// Magic 0x00010000
		buf[0], buf[1], buf[2], buf[3] = 0x00, 0x01, 0x00, 0x00
		// numTables = 4
		buf[4], buf[5] = 0x00, 0x04

		// Table 1 : cmap avec offset 0xFFFFFFFF
		copy(buf[12:16], []byte("cmap"))
		buf[20], buf[21], buf[22], buf[23] = 0xFF, 0xFF, 0xFF, 0xFF
		buf[24], buf[25], buf[26], buf[27] = 0x00, 0x00, 0x10, 0x00

		// Table 2 : head avec offset 0x7FFFFFFF
		copy(buf[28:32], []byte("head"))
		buf[36], buf[37], buf[38], buf[39] = 0x7F, 0xFF, 0xFF, 0xFF
		buf[40], buf[41], buf[42], buf[43] = 0x00, 0x00, 0x00, 0x36

		var font tt55.Tt55_font
		ret := tt55.Tt55_open(buf, uint64(len(buf)), &font)
		if ret == 0 {
			t.Fatalf("Tt55_open n'aurait pas du valider des tables pointant vers des offsets hors-bornes")
		}
	})

	t.Run("BrokenCmapSubtables", func(t *testing.T) {
		// Construction d'une structure minimale avec cmap présent mais subtable corrompue
		buf := make([]byte, 512)
		buf[0], buf[1], buf[2], buf[3] = 0x00, 0x01, 0x00, 0x00
		buf[4], buf[5] = 0x00, 0x01 // 1 table

		copy(buf[12:16], []byte("cmap"))
		buf[20], buf[21], buf[22], buf[23] = 0x00, 0x00, 0x00, 0x30 // offset 48
		buf[24], buf[25], buf[26], buf[27] = 0x00, 0x00, 0x01, 0x00 // len 256

		// cmap header
		buf[48], buf[49] = 0x00, 0x00 // version 0
		buf[50], buf[51] = 0x00, 0x01 // 1 subtable
		// encoding record: platform 3, encoding 1, offset 0x00000008 (relatif à cmap)
		buf[52], buf[53] = 0x00, 0x03
		buf[54], buf[55] = 0x00, 0x01
		buf[56], buf[57], buf[58], buf[59] = 0x00, 0x00, 0x00, 0x08

		// Format 4 subtable à l'offset 48 + 8 = 56
		buf[56], buf[57] = 0x00, 0x04 // format 4
		buf[58], buf[59] = 0x00, 0x20 // length 32
		buf[60], buf[61] = 0x00, 0x00 // language 0
		buf[62], buf[63] = 0x00, 0x00 // segCountX2 = 0 (dégénéré !)

		var font tt55.Tt55_font
		ret := tt55.Tt55_open(buf, uint64(len(buf)), &font)
		if ret == 0 {
			// Si open valide, Glyph_fmt4 doit retourner sans paniquer
			var gid uint16
			_ = tt55.Tt55_glyph(&font, 0x41, &gid)
		}
	})

	t.Run("FontManagerStringMeasurementAndRendering", func(t *testing.T) {
		fm := c2fynedriver.DefaultFontManager()
		fm.SetTTFont(nil) // Revenir au moteur bitmap natif 8x16

		surf := c2painter.NewSurface(640, 480)
		p := c2painter.NewPainter(surf)

		testStrings := []string{
			"",
			"\x00\x00\x00",
			"Bonjour le monde ! Éèàçôîû⚡✓•▶",
			string(make([]rune, 10000)),
			"\r\n\t\b\x1b[31mTexte Brut",
		}

		for _, str := range testStrings {
			w, h := c2fynedriver.MeasureString(str, 14)
			if w < 0 || h < 0 {
				t.Fatalf("MeasureString a produit des dimensions negatives: %dx%d", w, h)
			}
			c2fynedriver.DrawString(p, str, 10, 10, 14, c2painter.RGBA(255, 255, 255, 255))
		}
	})
}

// =============================================================================
// 4. Banc de Torture & Fuzzing : Arbre de Widgets Massif (FuzzWidgetTreeMassive)
// =============================================================================

func FuzzWidgetTreeMassive(f *testing.F) {
	f.Add(1, 10)
	f.Add(10, 5)
	f.Add(100, 2)
	f.Add(500, 1)

	f.Fuzz(func(t *testing.T, depth, breadth int) {
		if depth <= 0 {
			depth = 1
		}
		if depth > 500 {
			depth = 500
		}
		if breadth <= 0 {
			breadth = 1
		}
		if breadth > 10 {
			breadth = 10
		}

		root := buildNestedTree(depth, breadth)
		if root == nil {
			t.Fatalf("échec de génération de l'arbre")
		}

		surf := c2painter.NewSurface(800, 600)
		p := c2painter.NewPainter(surf)

		canvas := c2fynedriver.NewCanvas(nil, 800, 600)
		canvas.SetContent(root)
		canvas.Walk(func(obj c2fynedriver.CanvasObject, pos c2fynedriver.Position, size c2fynedriver.Size) bool {
			return true
		})
		canvas.Paint(p)
	})
}

func buildNestedTree(depth, breadth int) c2fynedriver.CanvasObject {
	if depth <= 1 {
		items := make([]c2fynedriver.CanvasObject, 0, breadth)
		for i := 0; i < breadth; i++ {
			items = append(items, c2fynedriver.NewLabel(fmt.Sprintf("Feuille %d", i)))
		}
		return c2fynedriver.NewHBox(2, items...)
	}

	children := make([]c2fynedriver.CanvasObject, 0, breadth)
	for i := 0; i < breadth; i++ {
		children = append(children, buildNestedTree(depth-1, 1))
	}
	return c2fynedriver.NewVBox(2, children...)
}

func TestTortureWidgetTreeMassive(t *testing.T) {
	t.Run("DeeplyNestedLinearTree_10000_Widgets", func(t *testing.T) {
		const targetDepth = 10000
		var current c2fynedriver.CanvasObject = c2fynedriver.NewButton("Feuille Terminale", nil)

		start := time.Now()
		for i := 0; i < targetDepth; i++ {
			current = c2fynedriver.NewVBox(0, current)
		}
		buildDur := time.Since(start)

		surf := c2painter.NewSurface(800, 600)
		p := c2painter.NewPainter(surf)

		canvas := c2fynedriver.NewCanvas(nil, 800, 600)
		canvas.SetContent(current)

		nodeCount := 0
		walkStart := time.Now()
		canvas.Walk(func(obj c2fynedriver.CanvasObject, pos c2fynedriver.Position, size c2fynedriver.Size) bool {
			nodeCount++
			return true
		})
		walkDur := time.Since(walkStart)

		paintStart := time.Now()
		canvas.Paint(p)
		paintDur := time.Since(paintStart)

		if nodeCount != targetDepth+1 {
			t.Fatalf("compte de nœuds inattendu: attendu %d, obtenu %d", targetDepth+1, nodeCount)
		}
		t.Logf("Arbre récursif linéaire de %d nœuds : build=%v, walk=%v, paint=%v",
			nodeCount, buildDur, walkDur, paintDur)
	})

	t.Run("WideContainer_10000_Children", func(t *testing.T) {
		const childCount = 10000
		children := make([]c2fynedriver.CanvasObject, childCount)

		for i := 0; i < childCount; i++ {
			switch i % 6 {
			case 0:
				children[i] = c2fynedriver.NewLabel(fmt.Sprintf("Label %d", i))
			case 1:
				children[i] = c2fynedriver.NewButton(fmt.Sprintf("Btn %d", i), nil)
			case 2:
				children[i] = c2fynedriver.NewRectangle(c2painter.RGBA(uint8(i), 50, 100, 255))
			case 3:
				children[i] = c2fynedriver.NewCircle(c2painter.RGBA(100, uint8(i), 50, 255))
			case 4:
				children[i] = c2fynedriver.NewBadge("ACTIF", c2painter.RGBA(34, 197, 94, 255), c2painter.RGBA(255, 255, 255, 255))
			case 5:
				children[i] = c2fynedriver.NewCard("Titre", "Sous-titre", c2fynedriver.NewLabel("Contenu"))
			}
		}

		container := c2fynedriver.NewVBox(2, children...)
		surf := c2painter.NewSurface(1920, 1080)
		p := c2painter.NewPainter(surf)

		canvas := c2fynedriver.NewCanvas(nil, 1920, 1080)
		canvas.SetContent(container)

		start := time.Now()
		canvas.Paint(p)
		dur := time.Since(start)

		t.Logf("Rendu conteneur large de %d widgets polymorphes en %v", childCount, dur)
	})

	t.Run("DynamicMutationsDuringWalkingAndRendering", func(t *testing.T) {
		root := c2fynedriver.NewVBox(4)
		for i := 0; i < 100; i++ {
			root.Add(c2fynedriver.NewButton(fmt.Sprintf("Item %d", i), nil))
		}

		surf := c2painter.NewSurface(800, 600)
		p := c2painter.NewPainter(surf)
		canvas := c2fynedriver.NewCanvas(nil, 800, 600)
		canvas.SetContent(root)

		var mu sync.Mutex
		var stop atomic.Bool
		var wg sync.WaitGroup

		// Goroutine de rendu continu
		wg.Add(1)
		go func() {
			defer wg.Done()
			for !stop.Load() {
				mu.Lock()
				canvas.Paint(p)
				mu.Unlock()
				time.Sleep(100 * time.Microsecond)
			}
		}()

		// Goroutine de mutation d'arbre dynamique
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				mu.Lock()
				if i%2 == 0 {
					root.Add(c2fynedriver.NewLabel(fmt.Sprintf("Dynamique %d", i)))
				} else if len(root.Objects) > 50 {
					root.Objects = root.Objects[:len(root.Objects)-1]
					root.LayoutChildren()
				}
				mu.Unlock()
				time.Sleep(50 * time.Microsecond)
			}
		}()

		time.Sleep(50 * time.Millisecond)
		stop.Store(true)
		wg.Wait()
		t.Logf("Mutations dynamiques sous rendu validées sans conflit")
	})
}

// =============================================================================
// 5. Banc de Torture & Fuzzing : Ouragan de Redimensionnements (FuzzResizeStorm)
// =============================================================================

func FuzzResizeStorm(f *testing.F) {
	f.Add(800, 600)
	f.Add(10, 10)
	f.Add(3840, 2160)
	f.Add(-100, -100)
	f.Add(0, 0)
	f.Add(1, 1)
	f.Add(8000, 8000)

	f.Fuzz(func(t *testing.T, width, height int) {
		drv, err := c2fynedriver.NewHeadlessDriver(800, 600)
		if err != nil {
			t.Fatalf("échec d'instanciation HeadlessDriver: %v", err)
		}
		defer drv.Quit()

		win, err := drv.CreateWindow("Fuzz Window", 800, 600)
		if err != nil {
			t.Fatalf("échec de création fenêtre: %v", err)
		}
		defer win.Close()

		term := c2fynedriver.NewTerminalComponent(80, 24)
		btn := c2fynedriver.NewButton("Action", func() {})
		vbox := c2fynedriver.NewVBox(4, term, btn)
		win.SetContent(vbox)

		// Redimensionnement
		win.Resize(c2fynedriver.Size{Width: width, Height: height})
		win.RenderFrame()

		// Événements injectés
		canvas := win.Canvas()
		canvas.HandleMouseEvent(c2display.MouseEvent{
			Action: c2display.EventMousePress,
			Button: c2display.ButtonLeft,
			X:      width / 2,
			Y:      height / 2,
		})
		canvas.HandleKeyEvent(c2display.KeyEvent{
			Action: c2display.EventKeyPress,
			Key:    c2display.KeyEnter,
			Rune:   '\n',
		})
	})
}

func TestTortureResizeStorm(t *testing.T) {
	t.Run("TenThousandRapidResizes", func(t *testing.T) {
		drv, err := c2fynedriver.NewHeadlessDriver(800, 600)
		if err != nil {
			t.Fatalf("échec HeadlessDriver: %v", err)
		}
		defer drv.Quit()

		win, err := drv.CreateWindow("Storm Win", 800, 600)
		if err != nil {
			t.Fatalf("échec CreateWindow: %v", err)
		}
		defer win.Close()

		term := c2fynedriver.NewTerminalComponent(80, 24)
		win.SetContent(c2fynedriver.NewVBox(4,
			c2fynedriver.NewLabel("Ouragan de redimensionnements"),
			term,
			c2fynedriver.NewButton("Valider", nil),
		))

		sizes := []c2fynedriver.Size{
			{Width: 10, Height: 10},
			{Width: 320, Height: 240},
			{Width: 800, Height: 600},
			{Width: 1280, Height: 720},
			{Width: 1920, Height: 1080},
			{Width: 2560, Height: 1440},
			{Width: 3840, Height: 2160},
			{Width: 0, Height: 0},
			{Width: -50, Height: -50},
			{Width: 1, Height: 1},
		}

		start := time.Now()
		const iterations = 10000
		for i := 0; i < iterations; i++ {
			sz := sizes[i%len(sizes)]
			win.Resize(sz)
			if i%100 == 0 {
				win.RenderFrame()
			}
		}
		dur := time.Since(start)

		rate := float64(iterations) / dur.Seconds()
		t.Logf("10 000 redimensionnements exécutés en %v (cadence: %.1f Resize/s)", dur, rate)
	})

	t.Run("MultiThreadedResizeAndEventStorm", func(t *testing.T) {
		drv, err := c2fynedriver.NewHeadlessDriver(1920, 1080)
		if err != nil {
			t.Fatalf("échec HeadlessDriver: %v", err)
		}
		defer drv.Quit()

		win, err := drv.CreateWindow("MultiStorm Win", 1920, 1080)
		if err != nil {
			t.Fatalf("échec CreateWindow: %v", err)
		}
		defer win.Close()

		termComp := c2fynedriver.NewTerminalComponent(80, 24)
		win.SetContent(c2fynedriver.NewVBox(4,
			c2fynedriver.NewLabel("Torture Multi-Thread"),
			termComp,
			c2fynedriver.NewFPSCounter(drv),
			c2fynedriver.NewButton("Pression", func() {}),
		))

		var stop atomic.Bool
		var wg sync.WaitGroup
		var totalResizes atomic.Uint64
		var totalFrames atomic.Uint64
		var totalEvents atomic.Uint64

		// Thread 1: Ouragan de redimensionnements
		wg.Add(1)
		go func() {
			defer wg.Done()
			resolutions := [][2]int{
				{800, 600}, {1024, 768}, {1280, 720}, {1920, 1080},
				{2560, 1440}, {3840, 2160}, {640, 480}, {100, 100},
			}
			idx := 0
			for !stop.Load() {
				res := resolutions[idx%len(resolutions)]
				win.Resize(c2fynedriver.Size{Width: res[0], Height: res[1]})
				totalResizes.Add(1)
				idx++
				time.Sleep(50 * time.Microsecond)
			}
		}()

		// Thread 2: Rendu continu de trames
		wg.Add(1)
		go func() {
			defer wg.Done()
			for !stop.Load() {
				win.RenderFrame()
				totalFrames.Add(1)
				time.Sleep(200 * time.Microsecond)
			}
		}()

		// Thread 3: Injection d'événements souris
		wg.Add(1)
		go func() {
			defer wg.Done()
			canvas := win.Canvas()
			for !stop.Load() {
				canvas.HandleMouseEvent(c2display.MouseEvent{
					Action: c2display.EventMouseMove,
					X:      rand.Intn(1920),
					Y:      rand.Intn(1080),
				})
				canvas.HandleMouseEvent(c2display.MouseEvent{
					Action: c2display.EventMousePress,
					Button: c2display.ButtonLeft,
					X:      rand.Intn(1920),
					Y:      rand.Intn(1080),
				})
				canvas.HandleMouseEvent(c2display.MouseEvent{
					Action: c2display.EventMouseRelease,
					Button: c2display.ButtonLeft,
					X:      rand.Intn(1920),
					Y:      rand.Intn(1080),
				})
				totalEvents.Add(3)
				time.Sleep(100 * time.Microsecond)
			}
		}()

		// Thread 4: Injection d'événements clavier
		wg.Add(1)
		go func() {
			defer wg.Done()
			canvas := win.Canvas()
			keys := []rune{'a', 'b', '\n', '\t', ' ', 'z', '1', '9'}
			for !stop.Load() {
				r := keys[rand.Intn(len(keys))]
				canvas.HandleKeyEvent(c2display.KeyEvent{
					Action: c2display.EventKeyPress,
					Rune:   r,
				})
				totalEvents.Add(1)
				time.Sleep(100 * time.Microsecond)
			}
		}()

		// Thread 5: Ingestion flux PTY dans le terminal
		wg.Add(1)
		go func() {
			defer wg.Done()
			stream := []byte("Streaming PTY concurrently with resize storm...\r\n")
			for !stop.Load() {
				_, _ = termComp.Term.Write(stream)
				time.Sleep(100 * time.Microsecond)
			}
		}()

		// Exécution du banc pendant 200 ms sous charge maximale
		time.Sleep(200 * time.Millisecond)
		stop.Store(true)
		wg.Wait()

		t.Logf("Ouragan concurrent achevé avec succès : %d redimensionnements, %d trames, %d événements traités sans défaillance",
			totalResizes.Load(), totalFrames.Load(), totalEvents.Load())
	})
}
