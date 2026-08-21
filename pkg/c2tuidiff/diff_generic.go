package c2tuidiff

import (
	"unsafe"
)

// diffGridFn pointe l'implémentation active du diff de grilles. La valeur par
// défaut est la référence scalaire 64-bit (diffGridScalar). Les fichiers SIMD
// (diff_simd_amd64.go, diff_simd_arm64.go) remplacent cette valeur dans leur
// propre init(), chacun protégé par son propre //go:build.
var diffGridFn = diffGridScalar

// diffSimdActive signale qu'une variante vectorielle est câblée. Utilisé par
// les tests et benchmarks pour distinguer la voie SIMD du repli scalaire.
var diffSimdActive = false

// diffGridScalar délègue au noyau transpilé C2_diff_grid_scalar (sgoiter).
// La liste ordonnée des Span est bit-exacte avec les variantes SIMD.
func diffGridScalar(front, back []Cell, width, height, stride int, spans *[]Span) int {
	if width <= 0 || height <= 0 || stride < width {
		return 0
	}
	need := (height-1)*stride + width
	if len(front) < need || len(back) < need {
		return 0
	}
	n0 := len(*spans)
	max := cap(*spans) - n0
	if max <= 0 {
		extra := height * width
		if extra < 1 {
			extra = 1
		}
		buf := make([]Span, n0, n0+extra)
		copy(buf, *spans)
		*spans = buf
		max = cap(*spans) - n0
	}
	slot := (*spans)[n0 : n0+max]
	cf := unsafe.Slice((*C2_cell_t)(unsafe.Pointer(&front[0])), len(front))
	cb := unsafe.Slice((*C2_cell_t)(unsafe.Pointer(&back[0])), len(back))
	cs := unsafe.Slice((*C2_span_t)(unsafe.Pointer(&slot[0])), max)
	ns := C2_diff_grid_scalar(cf, cb, height*stride, stride, width, cs, max)
	if ns < 0 {
		extra := height * width
		if extra < max*2 {
			extra = max * 2
		}
		if extra < 1 {
			extra = 1
		}
		buf := make([]Span, n0, n0+extra)
		copy(buf, (*spans)[:n0])
		*spans = buf
		max = cap(*spans) - n0
		slot = (*spans)[n0 : n0+max]
		cs = unsafe.Slice((*C2_span_t)(unsafe.Pointer(&slot[0])), max)
		ns = C2_diff_grid_scalar(cf, cb, height*stride, stride, width, cs, max)
		if ns < 0 {
			return 0
		}
	}
	if ns > max {
		ns = max
	}
	*spans = (*spans)[:n0+ns]
	changed := 0
	for i := n0; i < n0+ns; i++ {
		changed += (*spans)[i].Length
	}
	return changed
}

// cellEqScalar compare deux cellules (8 octets) positionnées en off dans les
// vues octet des grilles.
func cellEqScalar(fb, bb []byte, off int) bool {
	return *(*uint64)(unsafe.Pointer(&fb[off])) == *(*uint64)(unsafe.Pointer(&bb[off]))
}
