// Package fastxor offre une routine XOR vectorisée 64 bits ultra-rapide sans CGO.
package fastxor

import "encoding/binary"

// Bytes effectue l'opération XOR octet par octet : dst[i] = src1[i] ^ src2[i].
func Bytes(dst, src1, src2 []byte) {
	n := len(src1)
	if len(src2) < n {
		n = len(src2)
	}
	if len(dst) < n {
		return
	}
	FastXorBytes(dst, src1, src2, uint64(n))
}

func FastXorBytes(dst []byte, src1 []byte, src2 []byte, len_ uint64) {
	var v4 uint64
	var v7 uint64
	var v16 uint64
	var v32 uint8
	var v34 uint8
	var v14 uint64
	var v21 uint64
	var v24 uint64
	var v33 uint8
	v4 = uint64(0)
	for {
		v7 = v4 + uint64(8)
		if !(v7 <= len_) {
			break
		}
		v16 = v4 &^ uint64(7)
		v14 = binary.LittleEndian.Uint64(src1[int(v16):])
		v21 = binary.LittleEndian.Uint64(src2[int(v16):])
		v24 = v14 ^ v21
		binary.LittleEndian.PutUint64(dst[int(v16):], v24)
		v4 = v7
	}
	for v4 < len_ {
		v32 = src1[int(v4)]
		v33 = src2[int(v4)]
		v34 = v32 ^ v33
		dst[int(v4)] = v34
		v4 = v4 + uint64(1)
	}
}
