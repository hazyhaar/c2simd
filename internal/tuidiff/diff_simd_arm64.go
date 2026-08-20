//go:build goexperiment.simd && arm64

package c2tuidiff

import (
	"simd/archsimd"
	"unsafe"
)

// init active la variante NEON 128-bit (Advanced SIMD est supporté par défaut
// sur arm64). La référence scalaire demeure la valeur initiale de diffGridFn.
func init() {
	diffGridFn = diffGridNEON
	diffSimdActive = true
}

// diffGridNEON compare les grilles par paquets de 2 cellules (16 octets) à
// l'aide de vecteurs Uint64x2 (NEON 128-bit, une cellule par lane 64-bit),
// avec une boucle de terminaison scalaire pour les reliquats (largeur impaire).
func diffGridNEON(front, back []Cell, width, height, stride int, spans *[]Span) int {
	if width <= 0 || height <= 0 || stride < width {
		return 0
	}
	need := (height-1)*stride + width
	if len(front) < need || len(back) < need {
		return 0
	}
	fb := unsafe.Slice((*byte)(unsafe.Pointer(&front[0])), len(front)*8)
	bb := unsafe.Slice((*byte)(unsafe.Pointer(&back[0])), len(back)*8)

	changed := 0
	for y := 0; y < height; y++ {
		rowBase := y * stride
		x := 0
		for x < width {
			// Avance rapide par paquets de 2 cellules (16 octets).
			for x+2 <= width {
				m := chunkEqNEON(fb, bb, (rowBase+x)*8)
				if m == 0 { // les 2 cellules sont identiques
					x += 2
					continue
				}
				if m&1 == 0 {
					x++ // seule la cellule 1 du paquet diffère
				}
				break
			}
			if x >= width {
				break
			}
			if cellEqScalar(fb, bb, (rowBase+x)*8) {
				x++
				continue
			}
			start := x
			for x < width && !cellEqScalar(fb, bb, (rowBase+x)*8) {
				x++
			}
			*spans = append(*spans, Span{X: start, Y: y, Length: x - start})
			changed += x - start
		}
	}
	return changed
}

// chunkEqNEON charge 2 cellules (16 octets) depuis front et back et renvoie
// un masque à 2 bits (bit j à 1 si la cellule j du paquet diffère).
func chunkEqNEON(fb, bb []byte, off int) uint8 {
	var fw, bw [2]uint64
	fw[0] = *(*uint64)(unsafe.Pointer(&fb[off]))
	fw[1] = *(*uint64)(unsafe.Pointer(&fb[off+8]))
	bw[0] = *(*uint64)(unsafe.Pointer(&bb[off]))
	bw[1] = *(*uint64)(unsafe.Pointer(&bb[off+8]))
	vf := archsimd.LoadUint64x2Array(&fw)
	vb := archsimd.LoadUint64x2Array(&bw)
	var out [2]int64
	vf.Equal(vb).ToInt64x2().StoreArray(&out)
	var m uint8
	if out[0] == 0 {
		m |= 1
	}
	if out[1] == 0 {
		m |= 2
	}
	return m
}