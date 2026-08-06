//go:build goexperiment.simd

package c2simd

import (
	"encoding/binary"
	"simd/archsimd"
	"unsafe"
)

// EncryptChaCha20_SIMD256 chiffre un tampon de données en utilisant le noyau SIMD 256-bit AVX2 (archsimd.Uint32x8)
// sur 4 blocs parallèles (256 octets par itération) avec masquage XOR 64-bit conforme à l'entrelacement YMM (0 allocation heap).
func EncryptChaCha20_SIMD256(key []byte, nonce []byte, counter uint32, src []byte, dst []byte) {
	if len(src) == 0 {
		return
	}

	c0 := binary.LittleEndian.Uint32([]byte("expa"))
	c1 := binary.LittleEndian.Uint32([]byte("nd 3"))
	c2 := binary.LittleEndian.Uint32([]byte("2-by"))
	c3 := binary.LittleEndian.Uint32([]byte("te k"))

	k0 := binary.LittleEndian.Uint32(key[0:4])
	k1 := binary.LittleEndian.Uint32(key[4:8])
	k2 := binary.LittleEndian.Uint32(key[8:12])
	k3 := binary.LittleEndian.Uint32(key[12:16])
	k4 := binary.LittleEndian.Uint32(key[16:20])
	k5 := binary.LittleEndian.Uint32(key[20:24])
	k6 := binary.LittleEndian.Uint32(key[24:28])
	k7 := binary.LittleEndian.Uint32(key[28:32])

	n0 := binary.LittleEndian.Uint32(nonce[0:4])
	n1 := binary.LittleEndian.Uint32(nonce[4:8])
	n2 := binary.LittleEndian.Uint32(nonce[8:12])

	st0 := archsimd.LoadUint32x8Array(&[8]uint32{c0, c1, c2, c3, c0, c1, c2, c3})
	st1 := archsimd.LoadUint32x8Array(&[8]uint32{k0, k1, k2, k3, k0, k1, k2, k3})
	st2 := archsimd.LoadUint32x8Array(&[8]uint32{k4, k5, k6, k7, k4, k5, k6, k7})

	offset := 0
	currCtr := counter

	// 1. Traitement des blocs de 256 octets via SIMD256 AVX2 (4 blocs par itération)
	for offset+256 <= len(src) {
		st3_A := archsimd.LoadUint32x8Array(&[8]uint32{
			currCtr, n0, n1, n2,
			currCtr + 1, n0, n1, n2,
		})
		st3_B := archsimd.LoadUint32x8Array(&[8]uint32{
			currCtr + 2, n0, n1, n2,
			currCtr + 3, n0, n1, n2,
		})

		v0_A, v1_A, v2_A, v3_A := st0, st1, st2, st3_A
		v0_B, v1_B, v2_B, v3_B := st0, st1, st2, st3_B

		for i := 0; i < 10; i++ {
			v0_A, v1_A, v2_A, v3_A = ChaCha20DoubleBlockSIMD256(v0_A, v1_A, v2_A, v3_A)
			v0_B, v1_B, v2_B, v3_B = ChaCha20DoubleBlockSIMD256(v0_B, v1_B, v2_B, v3_B)
		}

		v0_A = v0_A.Add(st0)
		v1_A = v1_A.Add(st1)
		v2_A = v2_A.Add(st2)
		v3_A = v3_A.Add(st3_A)

		v0_B = v0_B.Add(st0)
		v1_B = v1_B.Add(st1)
		v2_B = v2_B.Add(st2)
		v3_B = v3_B.Add(st3_B)

		var buf0_A, buf1_A, buf2_A, buf3_A [8]uint32
		var buf0_B, buf1_B, buf2_B, buf3_B [8]uint32
		v0_A.StoreArray(&buf0_A)
		v1_A.StoreArray(&buf1_A)
		v2_A.StoreArray(&buf2_A)
		v3_A.StoreArray(&buf3_A)

		v0_B.StoreArray(&buf0_B)
		v1_B.StoreArray(&buf1_B)
		v2_B.StoreArray(&buf2_B)
		v3_B.StoreArray(&buf3_B)

		// Écriture 64-bit vectorisée pour les 4 blocs (256 octets)
		for b := 0; b < 2; b++ {
			bOff := offset + b*64
			base := b * 4

			sPtr := (*[8]uint64)(unsafe.Pointer(&src[bOff]))
			dPtr := (*[8]uint64)(unsafe.Pointer(&dst[bOff]))

			u64_0 := uint64(buf0_A[base+0]) | (uint64(buf0_A[base+1]) << 32)
			u64_1 := uint64(buf0_A[base+2]) | (uint64(buf0_A[base+3]) << 32)
			u64_2 := uint64(buf1_A[base+0]) | (uint64(buf1_A[base+1]) << 32)
			u64_3 := uint64(buf1_A[base+2]) | (uint64(buf1_A[base+3]) << 32)
			u64_4 := uint64(buf2_A[base+0]) | (uint64(buf2_A[base+1]) << 32)
			u64_5 := uint64(buf2_A[base+2]) | (uint64(buf2_A[base+3]) << 32)
			u64_6 := uint64(buf3_A[base+0]) | (uint64(buf3_A[base+1]) << 32)
			u64_7 := uint64(buf3_A[base+2]) | (uint64(buf3_A[base+3]) << 32)

			dPtr[0] = sPtr[0] ^ u64_0
			dPtr[1] = sPtr[1] ^ u64_1
			dPtr[2] = sPtr[2] ^ u64_2
			dPtr[3] = sPtr[3] ^ u64_3
			dPtr[4] = sPtr[4] ^ u64_4
			dPtr[5] = sPtr[5] ^ u64_5
			dPtr[6] = sPtr[6] ^ u64_6
			dPtr[7] = sPtr[7] ^ u64_7
		}

		for b := 0; b < 2; b++ {
			bOff := offset + 128 + b*64
			base := b * 4

			sPtr := (*[8]uint64)(unsafe.Pointer(&src[bOff]))
			dPtr := (*[8]uint64)(unsafe.Pointer(&dst[bOff]))

			u64_0 := uint64(buf0_B[base+0]) | (uint64(buf0_B[base+1]) << 32)
			u64_1 := uint64(buf0_B[base+2]) | (uint64(buf0_B[base+3]) << 32)
			u64_2 := uint64(buf1_B[base+0]) | (uint64(buf1_B[base+1]) << 32)
			u64_3 := uint64(buf1_B[base+2]) | (uint64(buf1_B[base+3]) << 32)
			u64_4 := uint64(buf2_B[base+0]) | (uint64(buf2_B[base+1]) << 32)
			u64_5 := uint64(buf2_B[base+2]) | (uint64(buf2_B[base+3]) << 32)
			u64_6 := uint64(buf3_B[base+0]) | (uint64(buf3_B[base+1]) << 32)
			u64_7 := uint64(buf3_B[base+2]) | (uint64(buf3_B[base+3]) << 32)

			dPtr[0] = sPtr[0] ^ u64_0
			dPtr[1] = sPtr[1] ^ u64_1
			dPtr[2] = sPtr[2] ^ u64_2
			dPtr[3] = sPtr[3] ^ u64_3
			dPtr[4] = sPtr[4] ^ u64_4
			dPtr[5] = sPtr[5] ^ u64_5
			dPtr[6] = sPtr[6] ^ u64_6
			dPtr[7] = sPtr[7] ^ u64_7
		}

		offset += 256
		currCtr += 4
	}

	// 2. Traitement des blocs de 128 octets restants
	for offset+128 <= len(src) {
		st3 := archsimd.LoadUint32x8Array(&[8]uint32{
			currCtr, n0, n1, n2,
			currCtr + 1, n0, n1, n2,
		})

		v0, v1, v2, v3 := st0, st1, st2, st3
		for i := 0; i < 10; i++ {
			v0, v1, v2, v3 = ChaCha20DoubleBlockSIMD256(v0, v1, v2, v3)
		}

		v0 = v0.Add(st0)
		v1 = v1.Add(st1)
		v2 = v2.Add(st2)
		v3 = v3.Add(st3)

		var buf0, buf1, buf2, buf3 [8]uint32
		v0.StoreArray(&buf0)
		v1.StoreArray(&buf1)
		v2.StoreArray(&buf2)
		v3.StoreArray(&buf3)

		for b := 0; b < 2; b++ {
			bOff := offset + b*64
			base := b * 4

			sPtr := (*[8]uint64)(unsafe.Pointer(&src[bOff]))
			dPtr := (*[8]uint64)(unsafe.Pointer(&dst[bOff]))

			u64_0 := uint64(buf0[base+0]) | (uint64(buf0[base+1]) << 32)
			u64_1 := uint64(buf0[base+2]) | (uint64(buf0[base+3]) << 32)
			u64_2 := uint64(buf1[base+0]) | (uint64(buf1[base+1]) << 32)
			u64_3 := uint64(buf1[base+2]) | (uint64(buf1[base+3]) << 32)
			u64_4 := uint64(buf2[base+0]) | (uint64(buf2[base+1]) << 32)
			u64_5 := uint64(buf2[base+2]) | (uint64(buf2[base+3]) << 32)
			u64_6 := uint64(buf3[base+0]) | (uint64(buf3[base+1]) << 32)
			u64_7 := uint64(buf3[base+2]) | (uint64(buf3[base+3]) << 32)

			dPtr[0] = sPtr[0] ^ u64_0
			dPtr[1] = sPtr[1] ^ u64_1
			dPtr[2] = sPtr[2] ^ u64_2
			dPtr[3] = sPtr[3] ^ u64_3
			dPtr[4] = sPtr[4] ^ u64_4
			dPtr[5] = sPtr[5] ^ u64_5
			dPtr[6] = sPtr[6] ^ u64_6
			dPtr[7] = sPtr[7] ^ u64_7
		}

		offset += 128
		currCtr += 2
	}

	// 3. Traitement du reliquat (tail < 128 octets) sans allocation heap (tampon en pile)
	if offset < len(src) {
		rem := len(src) - offset
		st3 := archsimd.LoadUint32x8Array(&[8]uint32{
			currCtr, n0, n1, n2,
			currCtr + 1, n0, n1, n2,
		})

		v0, v1, v2, v3 := st0, st1, st2, st3
		for i := 0; i < 10; i++ {
			v0, v1, v2, v3 = ChaCha20DoubleBlockSIMD256(v0, v1, v2, v3)
		}
		v0 = v0.Add(st0)
		v1 = v1.Add(st1)
		v2 = v2.Add(st2)
		v3 = v3.Add(st3)

		var buf0, buf1, buf2, buf3 [8]uint32
		v0.StoreArray(&buf0)
		v1.StoreArray(&buf1)
		v2.StoreArray(&buf2)
		v3.StoreArray(&buf3)

		var ks [128]byte
		for b := 0; b < 2; b++ {
			bOff := b * 64
			base := b * 4
			binary.LittleEndian.PutUint32(ks[bOff+0:], buf0[base+0])
			binary.LittleEndian.PutUint32(ks[bOff+4:], buf0[base+1])
			binary.LittleEndian.PutUint32(ks[bOff+8:], buf0[base+2])
			binary.LittleEndian.PutUint32(ks[bOff+12:], buf0[base+3])

			binary.LittleEndian.PutUint32(ks[bOff+16:], buf1[base+0])
			binary.LittleEndian.PutUint32(ks[bOff+20:], buf1[base+1])
			binary.LittleEndian.PutUint32(ks[bOff+24:], buf1[base+2])
			binary.LittleEndian.PutUint32(ks[bOff+28:], buf1[base+3])

			binary.LittleEndian.PutUint32(ks[bOff+32:], buf2[base+0])
			binary.LittleEndian.PutUint32(ks[bOff+36:], buf2[base+1])
			binary.LittleEndian.PutUint32(ks[bOff+40:], buf2[base+2])
			binary.LittleEndian.PutUint32(ks[bOff+44:], buf2[base+3])

			binary.LittleEndian.PutUint32(ks[bOff+48:], buf3[base+0])
			binary.LittleEndian.PutUint32(ks[bOff+52:], buf3[base+1])
			binary.LittleEndian.PutUint32(ks[bOff+56:], buf3[base+2])
			binary.LittleEndian.PutUint32(ks[bOff+60:], buf3[base+3])
		}

		for i := 0; i < rem; i++ {
			dst[offset+i] = src[offset+i] ^ ks[i]
		}
	}
}
