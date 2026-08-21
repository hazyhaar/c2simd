//go:build goexperiment.simd && amd64

package c2tuidiff

import (
	"math/rand"
	"simd/archsimd"
	"testing"
	"unsafe"
)

// TestAVX2Wiring vérifie que la sonde matérielle a bien remplacé la référence
// scalaire par la variante AVX2 dans init().
func TestAVX2Wiring(t *testing.T) {
	if !archsimd.X86.AVX2() {
		t.Skip("AVX2 non disponible sur cette machine")
	}
	if !diffSimdActive {
		t.Fatal("sonde AVX2 positive mais diffGridFn non remplacée (diffSimdActive=false)")
	}
}

// TestAVX2KATDirect confronte la variante AVX2 à la référence scalaire en
// appel direct, hors dispatch, sur des grilles aléatoires de toutes largeurs.
func TestAVX2KATDirect(t *testing.T) {
	if !archsimd.X86.AVX2() {
		t.Skip("AVX2 non disponible sur cette machine")
	}
	rng := rand.New(rand.NewSource(20260814))
	for _, sz := range []struct{ w, h, stride int }{
		{4, 1, 4}, {5, 3, 8}, {32, 8, 32}, {33, 8, 36}, {127, 9, 128},
	} {
		for _, pct := range []float64{0.0, 0.05, 0.5, 1.0} {
			front, back := randGrid(rng, sz.w, sz.h, sz.stride, pct)
			var a, b []Span
			want := diffGridScalar(front, back, sz.w, sz.h, sz.stride, &a)
			got := diffGridAVX2(front, back, sz.w, sz.h, sz.stride, &b)
			if want != got || !spansEqual(a, b) {
				t.Fatalf("AVX2 direct: size %dx%d stride %d pct %.2f: scalar=%d/%v avx2=%d/%v",
					sz.w, sz.h, sz.stride, pct, want, a, got, b)
			}
		}
	}
}

func TestChunkDirtyAVX2VsC2(t *testing.T) {
	if !archsimd.X86.AVX2() {
		t.Skip("AVX2 non disponible sur cette machine")
	}
	rng := rand.New(rand.NewSource(20260820))
	var fw, bw [4]uint64
	fb := unsafe.Slice((*byte)(unsafe.Pointer(&fw[0])), 32)
	bb := unsafe.Slice((*byte)(unsafe.Pointer(&bw[0])), 32)
	for i := 0; i < 200; i++ {
		for j := 0; j < 4; j++ {
			fw[j] = rng.Uint64()
			bw[j] = fw[j]
			if rng.Intn(3) == 0 {
				bw[j] = rng.Uint64()
			}
		}
		want := C2_chunk_dirty4(fw[:], bw[:])
		got := int(chunkDirtyAVX2(fb, bb, 0))
		if got != want {
			t.Fatalf("i=%d fw=%v bw=%v c2=%d avx2=%d", i, fw, bw, want, got)
		}
	}
}
