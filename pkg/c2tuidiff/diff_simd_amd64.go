//go:build goexperiment.simd && amd64

package c2tuidiff

import (
	"math/bits"
	"simd/archsimd"
	"unsafe"
)

// init active la variante AVX2 256-bit quand la sonde matérielle le confirme.
// Sans AVX2, la référence scalaire demeure active (pas d'échec silencieux).
func init() {
	if archsimd.X86.AVX2() {
		diffGridFn = diffGridAVX2
		diffSimdActive = true
	}
}

// diffGridAVX2 compare les grilles par paquets de 4 cellules (32 octets) à
// l'aide de vecteurs Uint64x4 (AVX2 256-bit, une cellule par lane 64-bit),
// avec une boucle de terminaison scalaire pour les reliquats.
func diffGridAVX2(front, back []Cell, width, height, stride int, spans *[]Span) int {
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
			// Avance rapide par paquets de 4 cellules : masque 4-bit des
			// cellules modifiées, directement issu de la comparaison SIMD.
			for x+4 <= width {
				m := chunkDirtyAVX2(fb, bb, (rowBase+x)*8)
				if m == 0 {
					x += 4
					continue
				}
				x += bits.TrailingZeros8(m)
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

// chunkDirtyAVX2 charge 4 cellules (32 octets) directement depuis front et
// back et renvoie un masque à 4 bits (bit j à 1 si la cellule j du paquet
// diffère). Le chargement est effectué sans copie intermédiaire depuis le
// tampon mémoire des grilles.
func chunkDirtyAVX2(fb, bb []byte, off int) uint8 {
	fw := (*[4]uint64)(unsafe.Pointer(&fb[off]))
	bw := (*[4]uint64)(unsafe.Pointer(&bb[off]))
	vf := archsimd.LoadUint64x4Array(fw)
	vb := archsimd.LoadUint64x4Array(bw)
	// ToBits code l'égalité (bit à 1 si identique) : on inverse pour obtenir
	// le masque des cellules modifiées sur les 4 lanes utiles.
	return ^vf.Equal(vb).ToBits() & 0x0f
}