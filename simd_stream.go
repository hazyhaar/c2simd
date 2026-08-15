//go:build goexperiment.simd

package c2simd

import (
	"encoding/binary"
	"simd/archsimd"
)

var chachaCtrInc4 = archsimd.LoadUint32x8Array(&[8]uint32{4, 0, 0, 0, 4, 0, 0, 0})
var chachaCtrInc2 = archsimd.LoadUint32x8Array(&[8]uint32{2, 0, 0, 0, 2, 0, 0, 0})

// EncryptChaCha20_SIMD256 chiffre un tampon en noyau SIMD 256-bit AVX2, 4 blocs
// par itération. Étage de sortie réécrit le 2026-08-15 sur le modèle du kernel
// monocypher55 (2 927 MB/s) : XOR directement depuis les registres
// (GetLo/GetHi → LoadUint8x16.Xor.Store), plus aucun StoreArray ni
// recomposition scalaire dans la boucle chaude ; compteur IETF 32-bit avancé
// par Add de constante en lane (pas de rechargement d'état par itération).
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

	st3_A := archsimd.LoadUint32x8Array(&[8]uint32{
		currCtr, n0, n1, n2,
		currCtr + 1, n0, n1, n2,
	})
	st3_B := archsimd.LoadUint32x8Array(&[8]uint32{
		currCtr + 2, n0, n1, n2,
		currCtr + 3, n0, n1, n2,
	})

	// 1. Blocs de 256 octets (4 blocs/itération, pipeline A/B)
	for offset+256 <= len(src) {
		_ = src[offset+255]
		_ = dst[offset+255]

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

		o := offset
		k0_0, k0_1, k0_2, k0_3 := v0_A.GetLo().AsUint8x16(), v1_A.GetLo().AsUint8x16(), v2_A.GetLo().AsUint8x16(), v3_A.GetLo().AsUint8x16()
		archsimd.LoadUint8x16(src[o : o+16]).Xor(k0_0).Store(dst[o : o+16])
		archsimd.LoadUint8x16(src[o+16 : o+32]).Xor(k0_1).Store(dst[o+16 : o+32])
		archsimd.LoadUint8x16(src[o+32 : o+48]).Xor(k0_2).Store(dst[o+32 : o+48])
		archsimd.LoadUint8x16(src[o+48 : o+64]).Xor(k0_3).Store(dst[o+48 : o+64])

		k1_0, k1_1, k1_2, k1_3 := v0_A.GetHi().AsUint8x16(), v1_A.GetHi().AsUint8x16(), v2_A.GetHi().AsUint8x16(), v3_A.GetHi().AsUint8x16()
		archsimd.LoadUint8x16(src[o+64 : o+80]).Xor(k1_0).Store(dst[o+64 : o+80])
		archsimd.LoadUint8x16(src[o+80 : o+96]).Xor(k1_1).Store(dst[o+80 : o+96])
		archsimd.LoadUint8x16(src[o+96 : o+112]).Xor(k1_2).Store(dst[o+96 : o+112])
		archsimd.LoadUint8x16(src[o+112 : o+128]).Xor(k1_3).Store(dst[o+112 : o+128])

		k2_0, k2_1, k2_2, k2_3 := v0_B.GetLo().AsUint8x16(), v1_B.GetLo().AsUint8x16(), v2_B.GetLo().AsUint8x16(), v3_B.GetLo().AsUint8x16()
		archsimd.LoadUint8x16(src[o+128 : o+144]).Xor(k2_0).Store(dst[o+128 : o+144])
		archsimd.LoadUint8x16(src[o+144 : o+160]).Xor(k2_1).Store(dst[o+144 : o+160])
		archsimd.LoadUint8x16(src[o+160 : o+176]).Xor(k2_2).Store(dst[o+160 : o+176])
		archsimd.LoadUint8x16(src[o+176 : o+192]).Xor(k2_3).Store(dst[o+176 : o+192])

		k3_0, k3_1, k3_2, k3_3 := v0_B.GetHi().AsUint8x16(), v1_B.GetHi().AsUint8x16(), v2_B.GetHi().AsUint8x16(), v3_B.GetHi().AsUint8x16()
		archsimd.LoadUint8x16(src[o+192 : o+208]).Xor(k3_0).Store(dst[o+192 : o+208])
		archsimd.LoadUint8x16(src[o+208 : o+224]).Xor(k3_1).Store(dst[o+208 : o+224])
		archsimd.LoadUint8x16(src[o+224 : o+240]).Xor(k3_2).Store(dst[o+224 : o+240])
		archsimd.LoadUint8x16(src[o+240 : o+256]).Xor(k3_3).Store(dst[o+240 : o+256])

		offset += 256
		currCtr += 4
		// Compteur IETF 32-bit en lane 0/4 : l'Add wrappe naturellement en
		// arithmétique de lane, aucune recharge d'état nécessaire.
		st3_A = st3_A.Add(chachaCtrInc4)
		st3_B = st3_B.Add(chachaCtrInc4)
	}

	// 2. Paire de blocs restante (128 octets)
	for offset+128 <= len(src) {
		_ = src[offset+127]
		_ = dst[offset+127]

		v0, v1, v2, v3 := st0, st1, st2, st3_A
		for i := 0; i < 10; i++ {
			v0, v1, v2, v3 = ChaCha20DoubleBlockSIMD256(v0, v1, v2, v3)
		}
		v0 = v0.Add(st0)
		v1 = v1.Add(st1)
		v2 = v2.Add(st2)
		v3 = v3.Add(st3_A)

		o := offset
		ka0, ka1, ka2, ka3 := v0.GetLo().AsUint8x16(), v1.GetLo().AsUint8x16(), v2.GetLo().AsUint8x16(), v3.GetLo().AsUint8x16()
		archsimd.LoadUint8x16(src[o : o+16]).Xor(ka0).Store(dst[o : o+16])
		archsimd.LoadUint8x16(src[o+16 : o+32]).Xor(ka1).Store(dst[o+16 : o+32])
		archsimd.LoadUint8x16(src[o+32 : o+48]).Xor(ka2).Store(dst[o+32 : o+48])
		archsimd.LoadUint8x16(src[o+48 : o+64]).Xor(ka3).Store(dst[o+48 : o+64])

		kb0, kb1, kb2, kb3 := v0.GetHi().AsUint8x16(), v1.GetHi().AsUint8x16(), v2.GetHi().AsUint8x16(), v3.GetHi().AsUint8x16()
		archsimd.LoadUint8x16(src[o+64 : o+80]).Xor(kb0).Store(dst[o+64 : o+80])
		archsimd.LoadUint8x16(src[o+80 : o+96]).Xor(kb1).Store(dst[o+80 : o+96])
		archsimd.LoadUint8x16(src[o+96 : o+112]).Xor(kb2).Store(dst[o+96 : o+112])
		archsimd.LoadUint8x16(src[o+112 : o+128]).Xor(kb3).Store(dst[o+112 : o+128])

		offset += 128
		currCtr += 2
		st3_A = st3_A.Add(chachaCtrInc2)
		st3_B = st3_B.Add(chachaCtrInc2)
	}

	// 3. Reliquat (< 128 octets) : keystream de la paire courante en pile
	if offset < len(src) {
		rem := len(src) - offset

		v0, v1, v2, v3 := st0, st1, st2, st3_A
		for i := 0; i < 10; i++ {
			v0, v1, v2, v3 = ChaCha20DoubleBlockSIMD256(v0, v1, v2, v3)
		}
		v0 = v0.Add(st0)
		v1 = v1.Add(st1)
		v2 = v2.Add(st2)
		v3 = v3.Add(st3_A)

		var ks [128]byte
		v0.GetLo().AsUint8x16().Store(ks[0:16])
		v1.GetLo().AsUint8x16().Store(ks[16:32])
		v2.GetLo().AsUint8x16().Store(ks[32:48])
		v3.GetLo().AsUint8x16().Store(ks[48:64])
		v0.GetHi().AsUint8x16().Store(ks[64:80])
		v1.GetHi().AsUint8x16().Store(ks[80:96])
		v2.GetHi().AsUint8x16().Store(ks[96:112])
		v3.GetHi().AsUint8x16().Store(ks[112:128])

		for i := 0; i < rem; i++ {
			dst[offset+i] = src[offset+i] ^ ks[i]
		}
	}
}
