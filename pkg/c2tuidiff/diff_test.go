package c2tuidiff

import (
	"math/rand"
	"testing"
	"unsafe"
)

// randGrid génère deux grilles de mêmes dimensions avec un taux de
// perturbation contrôlé : chaque cellule modifiée l'est sur un seul champ.
func randGrid(rng *rand.Rand, w, h, stride int, dirtyPct float64) ([]Cell, []Cell) {
	front := make([]Cell, stride*h)
	back := make([]Cell, stride*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := Cell{Rune: rune(rng.Intn(0x80)), Fg: uint8(rng.Intn(256)), Bg: uint8(rng.Intn(16)), Flags: uint8(rng.Intn(8))}
			idx := y*stride + x
			front[idx] = c
			if rng.Float64() < dirtyPct {
				c2 := c
				switch rng.Intn(4) {
				case 0:
					c2.Rune = rune(rng.Intn(0x80))
				case 1:
					c2.Fg = uint8(rng.Intn(256))
				case 2:
					c2.Bg = uint8(rng.Intn(256))
				case 3:
					c2.Flags = uint8(rng.Intn(8))
				}
				back[idx] = c2
			} else {
				back[idx] = c
			}
		}
	}
	return front, back
}

// fixedGrid construit une grille réaliste de type texte de terminal.
func fixedGrid(w, h int) ([]Cell, []Cell) {
	front := make([]Cell, w*h)
	back := make([]Cell, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r := rune('a' + (x+y*3)%26)
			c := Cell{Rune: r, Fg: uint8((x*7 + y) % 256), Bg: uint8(y % 16), Flags: uint8(x % 4)}
			front[y*w+x] = c
			back[y*w+x] = c
		}
	}
	// Quelques modifications ciblées : une ligne entière, un bloc central,
	// une cellule isolée.
	for x := 0; x < w; x++ {
		back[3*w+x] = Cell{Rune: 'Z', Fg: 196, Bg: 0, Flags: 1}
	}
	for y := 7; y < 9; y++ {
		for x := 10; x < 20; x++ {
			back[y*w+x] = Cell{Rune: rune('@' + x - 10), Fg: 21, Bg: 226, Flags: 4}
		}
	}
	back[15*w+4] = Cell{Rune: '!', Fg: 0, Bg: 0, Flags: 0}
	return front, back
}

func spansEqual(a, b []Span) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestDiffKATBitExact vérifie que l'implémentation active (diffGridFn, SIMD le
// cas échéant) produit strictement la même liste ordonnée de Span que la
// référence scalaire, sur des grilles réelles et aléatoires.
func TestDiffKATBitExact(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	sizes := []struct {
		w, h, stride int
	}{
		{8, 4, 8},      // multiple de 4 et de 8
		{10, 6, 10},    // reliquats (non multiple de 4)
		{80, 24, 80},   // terminal 80x24
		{40, 12, 64},   // padding de stride
		{3, 5, 8},      // très étroit
		{1, 1, 1},      // cellule unique
		{100, 30, 104}, // padding + reliquats
		{16, 2, 16},    // frontière de paquet
	}
	for _, sz := range sizes {
		for _, pct := range []float64{0.0, 0.01, 0.2, 0.5, 1.0} {
			front, back := randGrid(rng, sz.w, sz.h, sz.stride, pct)
			var refSpans, actSpans []Span
			want := diffGridScalar(front, back, sz.w, sz.h, sz.stride, &refSpans)
			got := DiffGrid(front, back, sz.w, sz.h, sz.stride, &actSpans)
			if want != got {
				t.Fatalf("size %dx%d stride %d pct %.2f: count mismatch: scalar=%d active=%d",
					sz.w, sz.h, sz.stride, pct, want, got)
			}
			if !spansEqual(refSpans, actSpans) {
				t.Fatalf("size %dx%d stride %d pct %.2f: span mismatch\nscalar=%+v\nactive=%+v",
					sz.w, sz.h, sz.stride, pct, refSpans, actSpans)
			}
			// Vérification du comptage indépendant : somme des longueurs.
			var sum int
			for _, s := range actSpans {
				sum += s.Length
			}
			if sum != got {
				t.Fatalf("size %dx%d: span lengths sum %d != count %d", sz.w, sz.h, sum, got)
			}
		}
	}
}

// TestDiffKATRealistic vérifie la parité sur une grille réaliste non aléatoire.
func TestDiffKATRealistic(t *testing.T) {
	front, back := fixedGrid(80, 24)
	var refSpans, actSpans []Span
	want := diffGridScalar(front, back, 80, 24, 80, &refSpans)
	got := DiffGrid(front, back, 80, 24, 80, &actSpans)
	if want != got || !spansEqual(refSpans, actSpans) {
		t.Fatalf("realistic grid mismatch: scalar count=%d active count=%d\nscalar=%+v\nactive=%+v",
			want, got, refSpans, actSpans)
	}
	// Sur cette grille, on attend au moins les trois zones modifiées.
	if got != 80+2*10+1 {
		t.Fatalf("unexpected changed count: got %d, want %d", got, 80+2*10+1)
	}
}

// TestDiffEmptyAndInvalid vérifie le comportement sur les layouts dégénérés.
func TestMemoryLayout(t *testing.T) {
	if unsafe.Sizeof(Cell{}) != 8 || unsafe.Sizeof(C2_cell_t{}) != 8 {
		t.Fatalf("Sizeof Cell=%d C2=%d", unsafe.Sizeof(Cell{}), unsafe.Sizeof(C2_cell_t{}))
	}
	if unsafe.Offsetof(Cell{}.Width) != 7 {
		t.Fatalf("Width offset %d want 7", unsafe.Offsetof(Cell{}.Width))
	}
}

func TestDiffEmptyAndInvalid(t *testing.T) {
	empty := []Cell(nil)
	var spans []Span
	if got := DiffGrid(empty, empty, 80, 24, 80, &spans); got != 0 {
		t.Fatalf("empty grids: got %d, want 0", got)
	}
	if got := DiffGrid(nil, nil, 0, 0, 0, &spans); got != 0 {
		t.Fatalf("zero layout: got %d, want 0", got)
	}
	// stride < width : layout invalide.
	small := make([]Cell, 8)
	if got := DiffGrid(small, small, 10, 1, 8, &spans); got != 0 {
		t.Fatalf("invalid stride: got %d, want 0", got)
	}
}

func TestC2DiffGridScalarVsHand(t *testing.T) {
	front, back := fixedGrid(80, 24)
	cf := make([]C2_cell_t, len(front))
	cb := make([]C2_cell_t, len(back))
	for i := range front {
		cf[i] = C2_cell_t{Rune_: uint32(front[i].Rune), Fg: front[i].Fg, Bg: front[i].Bg, Flags: front[i].Flags, Width: front[i].Width}
		cb[i] = C2_cell_t{Rune_: uint32(back[i].Rune), Fg: back[i].Fg, Bg: back[i].Bg, Flags: back[i].Flags, Width: back[i].Width}
	}
	csp := make([]C2_span_t, 80*24)
	n := C2_diff_grid_scalar(cf, cb, 80*24, 80, 80, csp, len(csp))
	var hand []Span
	DiffGrid(front, back, 80, 24, 80, &hand)
	if n != len(hand) {
		t.Fatalf("span count gen=%d hand=%d", n, len(hand))
	}
	for i := 0; i < n; i++ {
		if csp[i].X != hand[i].X || csp[i].Y != hand[i].Y || csp[i].Length != hand[i].Length {
			t.Fatalf("span %d gen=%+v hand=%+v", i, csp[i], hand[i])
		}
	}
}

// TestDiffGridNoAlloc vérifie le contrat strict 0 B/op, 0 allocs/op sur le
// chemin de rendu (spans pré-alloués par l'appelant).
func TestDiffGridNoAlloc(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	front, back := randGrid(rng, 80, 24, 80, 0.3)
	spans := make([]Span, 0, 80*24)
	allocs := testing.AllocsPerRun(200, func() {
		spans = spans[:0]
		DiffGrid(front, back, 80, 24, 80, &spans)
	})
	if allocs != 0 {
		t.Fatalf("DiffGrid allocated %v allocs per run (want 0)", allocs)
	}
}

// TestDiffGridNoAllocSIMD est identique mais force la voie SIMD lorsqu'elle
// est active, en vérifiant que le câblage est bien en place.
func TestDiffGridNoAllocSIMD(t *testing.T) {
	if !diffSimdActive {
		t.Skip("SIMD path inactive (scalar fallback build)")
	}
	rng := rand.New(rand.NewSource(9))
	front, back := randGrid(rng, 80, 24, 80, 0.3)
	spans := make([]Span, 0, 80*24)
	allocs := testing.AllocsPerRun(200, func() {
		spans = spans[:0]
		diffGridFn(front, back, 80, 24, 80, &spans)
	})
	if allocs != 0 {
		t.Fatalf("SIMD DiffGrid allocated %v allocs per run (want 0)", allocs)
	}
}

func benchGrid(w, h int) ([]Cell, []Cell) {
	rng := rand.New(rand.NewSource(1))
	return randGrid(rng, w, h, w, 0.15)
}

// refreshTrace génère une trace de rafraîchissement réaliste : la grille back
// est identique à front, à l'exception de quelques zones ciblées (curseur,
// ligne de statut, zone de scroll) — cas nominal d'un terminal qui ne
// redessine que les régions modifiées.
func refreshTrace(w, h, regions int) ([]Cell, []Cell) {
	rng := rand.New(rand.NewSource(3))
	front := make([]Cell, w*h)
	back := make([]Cell, w*h)
	for i := range front {
		c := Cell{Rune: rune(' ' + i%0x60), Fg: uint8((i * 13) % 256), Bg: uint8((i / w) % 16), Flags: uint8(i % 4)}
		front[i] = c
		back[i] = c
	}
	for r := 0; r < regions; r++ {
		rw := 1 + rng.Intn(20)
		rh := 1 + rng.Intn(3)
		rx := rng.Intn(w - rw)
		ry := rng.Intn(h - rh)
		for y := ry; y < ry+rh; y++ {
			for x := rx; x < rx+rw; x++ {
				idx := y*w + x
				back[idx] = Cell{Rune: rune('A' + rng.Intn(26)), Fg: uint8(rng.Intn(256)), Bg: uint8(rng.Intn(16)), Flags: uint8(rng.Intn(8))}
			}
		}
	}
	return front, back
}

// Préparation des jeux de benchmarks une seule fois (globals) : les
// allocations de setup ne sont ainsi jamais comptabilisées par le harness à
// chaque passage de calibration, ce qui garantit 0 B/op mesuré.
var (
	benchStd   [2][]Cell
	benchStdH  int
	benchWide  [2][]Cell
	benchDense [2][]Cell
)

func benchOnce() {
	if benchStd[0] != nil {
		return
	}
	benchStd[0], benchStd[1] = refreshTrace(160, 60, 6)
	benchStdH = 60
	benchWide[0], benchWide[1] = refreshTrace(512, 40, 8)
	benchDense[0], benchDense[1] = benchGrid(160, 60)
}

// BenchmarkDiffScalar mesure la référence scalaire 64-bit.
func BenchmarkDiffScalar(b *testing.B) {
	benchOnce()
	spans := make([]Span, 0, 160*60)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		spans = spans[:0]
		diffGridScalar(benchStd[0], benchStd[1], 160, benchStdH, 160, &spans)
	}
}

// BenchmarkDiffSIMD mesure l'implémentation active (AVX2/NEON sous
// GOEXPERIMENT=simd). Le benchmark est ignoré sur un build scalaire.
func BenchmarkDiffSIMD(b *testing.B) {
	if !diffSimdActive {
		b.Skip("SIMD inactive (scalar fallback build)")
	}
	benchOnce()
	spans := make([]Span, 0, 160*60)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		spans = spans[:0]
		diffGridFn(benchStd[0], benchStd[1], 160, benchStdH, 160, &spans)
	}
}

// BenchmarkDiffWide mesure le cas large (grandes lignes) sur une trace
// réaliste où le bénéfice SIMD est maximal.
func BenchmarkDiffWide(b *testing.B) {
	benchOnce()
	spans := make([]Span, 0, 512*40)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		spans = spans[:0]
		DiffGrid(benchWide[0], benchWide[1], 512, 40, 512, &spans)
	}
}

// BenchmarkDiffDense mesure le cas défavorable (fort taux de change) : il
// documente le comportement honnête de la variante SIMD face à la scalaire.
func BenchmarkDiffDense(b *testing.B) {
	benchOnce()
	spans := make([]Span, 0, 160*60)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		spans = spans[:0]
		DiffGrid(benchDense[0], benchDense[1], 160, 60, 160, &spans)
	}
}
