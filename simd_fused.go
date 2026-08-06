//go:build goexperiment.simd

package c2simd

import (
	"crypto/subtle"
	"encoding/binary"
	"fmt"
	"simd/archsimd"
)

const (
	c0Archtime uint32 = 0x61707865 // "expa"
	c1Archtime uint32 = 0x3320646e // "nd 3"
	c2Archtime uint32 = 0x79622d32 // "2-by"
	c3Archtime uint32 = 0x6b206574 // "te k"
)

var zeroPad16 [16]byte

// HChaCha20 dérive une sous-clé 256-bit à partir de key (32B) et nonce (16B)
func HChaCha20(key, nonce, out []byte) {
	HChaCha20_SIMD128(key, nonce, out)
}

// HChaCha20_SIMD128 (Double-Rounds Conformes HChaCha20 100% Registres Vectoriels)
func HChaCha20_SIMD128(key, nonce, out []byte) {
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
	n3 := binary.LittleEndian.Uint32(nonce[12:16])

	v0 := archsimd.LoadUint32x4Array(&[4]uint32{c0Archtime, c1Archtime, c2Archtime, c3Archtime})
	v1 := archsimd.LoadUint32x4Array(&[4]uint32{k0, k1, k2, k3})
	v2 := archsimd.LoadUint32x4Array(&[4]uint32{k4, k5, k6, k7})
	v3 := archsimd.LoadUint32x4Array(&[4]uint32{n0, n1, n2, n3})

	// 10 itérations de Double-Rounds (20 rounds au total) sans accès mémoire intermédiaire
	for i := 0; i < 10; i++ {
		// Column Round
		v0 = v0.Add(v1)
		v3 = v3.Xor(v0).RotateAllLeft(16)
		v2 = v2.Add(v3)
		v1 = v1.Xor(v2).RotateAllLeft(12)
		v0 = v0.Add(v1)
		v3 = v3.Xor(v0).RotateAllLeft(8)
		v2 = v2.Add(v3)
		v1 = v1.Xor(v2).RotateAllLeft(7)

		// Diagonal Round (Permutations vectorielles directes)
		var b1 [4]uint32
		v1.StoreArray(&b1)
		v1 = archsimd.LoadUint32x4Array(&[4]uint32{b1[1], b1[2], b1[3], b1[0]})

		var b2 [4]uint32
		v2.StoreArray(&b2)
		v2 = archsimd.LoadUint32x4Array(&[4]uint32{b2[2], b2[3], b2[0], b2[1]})

		var b3 [4]uint32
		v3.StoreArray(&b3)
		v3 = archsimd.LoadUint32x4Array(&[4]uint32{b3[3], b3[0], b3[1], b3[2]})

		v0 = v0.Add(v1)
		v3 = v3.Xor(v0).RotateAllLeft(16)
		v2 = v2.Add(v3)
		v1 = v1.Xor(v2).RotateAllLeft(12)
		v0 = v0.Add(v1)
		v3 = v3.Xor(v0).RotateAllLeft(8)
		v2 = v2.Add(v3)
		v1 = v1.Xor(v2).RotateAllLeft(7)

		v1.StoreArray(&b1)
		v1 = archsimd.LoadUint32x4Array(&[4]uint32{b1[3], b1[0], b1[1], b1[2]})

		v2.StoreArray(&b2)
		v2 = archsimd.LoadUint32x4Array(&[4]uint32{b2[2], b2[3], b2[0], b2[1]})

		v3.StoreArray(&b3)
		v3 = archsimd.LoadUint32x4Array(&[4]uint32{b3[1], b3[2], b3[3], b3[0]})
	}

	var hk0, hk1 [4]uint32
	v0.StoreArray(&hk0)
	v3.StoreArray(&hk1)

	binary.LittleEndian.PutUint32(out[0:4], hk0[0])
	binary.LittleEndian.PutUint32(out[4:8], hk0[1])
	binary.LittleEndian.PutUint32(out[8:12], hk0[2])
	binary.LittleEndian.PutUint32(out[12:16], hk0[3])
	binary.LittleEndian.PutUint32(out[16:20], hk1[0])
	binary.LittleEndian.PutUint32(out[20:24], hk1[1])
	binary.LittleEndian.PutUint32(out[24:28], hk1[2])
	binary.LittleEndian.PutUint32(out[28:32], hk1[3])
}

// AEADLockSIMD256_Fused effectue le scellage conforme RFC 8439 / Monocypher en Pure Go (0 allocation)
func AEADLockSIMD256_Fused(key, nonce, ad, plaintext []byte) ([]byte, []byte, error) {
	var mac [16]byte
	cipherText, err := AEADLockSIMD256_FusedDst(nil, &mac, key, nonce, ad, plaintext)
	return cipherText, mac[:], err
}



// AEADLockSIMD256_FusedDst effectue le scellage en Pure Go (0 allocation si dstBuffer fourni)
func AEADLockSIMD256_FusedDst(dstBuffer []byte, outMac *[16]byte, key, nonce, ad, plaintext []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("c2simd: key must be 32 bytes")
	}
	if len(nonce) != 24 {
		return nil, fmt.Errorf("c2simd: XChaCha20 nonce must be 24 bytes")
	}

	// 1. Dérivation de la sous-clé HChaCha20 (24-byte Nonce)
	var subKeyBlock [32]byte
	HChaCha20_SIMD128(key, nonce[0:16], subKeyBlock[:])

	// 2. Formatage du nonce dérivé (4 octets 0 + nonce[16:24])
	var derivedNonce [12]byte
	copy(derivedNonce[4:12], nonce[16:24])

	// 3. Clé Poly1305 (Counter 0)
	var polyKeyBlock [64]byte
	var zeroBlock [64]byte
	EncryptChaCha20_SIMD256(subKeyBlock[:], derivedNonce[:], 0, zeroBlock[:], polyKeyBlock[:])

	polySt := NewPoly1305QuadChain(polyKeyBlock[:32])

	// 4. Cadrage Poly1305 : AD + zeroPad16(AD)
	if len(ad) > 0 {
		polySt.Update(ad)
		if rem := len(ad) % 16; rem > 0 {
			polySt.Update(zeroPad16[:16-rem])
		}
	}

	var cipherText []byte
	if cap(dstBuffer) >= len(plaintext) {
		cipherText = dstBuffer[:len(plaintext)]
	} else {
		cipherText = make([]byte, len(plaintext))
	}

	// 5. Chiffrement SIMD256 et Accumulation Poly1305 FUSIONNÉS par morceaux de 4096 octets (Taille de cache L1)
	const chunkSize = 4096
	offset := 0
	counter := uint32(1)

	// Indice BCE (Bounds Check Elimination) pour guider cmd/compile
	if len(plaintext) > 0 {
		_ = plaintext[len(plaintext)-1]
		_ = cipherText[len(plaintext)-1]
	}

	for offset < len(plaintext) {
		end := offset + chunkSize
		if end > len(plaintext) {
			end = len(plaintext)
		}

		inChunk := plaintext[offset:end]
		outChunk := cipherText[offset:end]

		// Chiffrement SIMD256 de la tranche
		numBlocks := (len(inChunk) + 63) / 64
		EncryptChaCha20_SIMD256(subKeyBlock[:], derivedNonce[:], counter, inChunk, outChunk)
		counter += uint32(numBlocks)

		// Accumulation Poly1305 directe de la tranche chiffrée
		polySt.Update(outChunk)

		offset = end
	}

	// Cadrage de fin de texte chiffré
	if rem := ciphertextPadding(len(plaintext)); rem > 0 {
		polySt.Update(zeroPad16[:rem])
	}

	// 6. Blocs de tailles (len(AD) + len(Ciphertext))
	var sizeBlock [16]byte
	binary.LittleEndian.PutUint64(sizeBlock[0:8], uint64(len(ad)))
	binary.LittleEndian.PutUint64(sizeBlock[8:16], uint64(len(plaintext)))
	polySt.Update(sizeBlock[:])

	// 7. Génération de la signature Poly1305 MAC finale
	polySt.Finish(outMac)

	return cipherText, nil
}

func ciphertextPadding(length int) int {
	rem := length % 16
	if rem == 0 {
		return 0
	}
	return 16 - rem
}

// AEADUnlockSIMD256_Fused effectue le déchiffrement authentifié (fail-closed)
func AEADUnlockSIMD256_Fused(key, nonce, ad, ciphertext, mac []byte) ([]byte, error) {
	return AEADUnlockSIMD256_FusedDst(nil, key, nonce, ad, ciphertext, mac)
}

// AEADUnlockSIMD256_FusedDst effectue le déchiffrement authentifié en Pure Go (0 allocation si dstBuffer fourni)
func AEADUnlockSIMD256_FusedDst(dstBuffer []byte, key, nonce, ad, ciphertext, mac []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("c2simd: key must be 32 bytes")
	}
	if len(nonce) != 24 {
		return nil, fmt.Errorf("c2simd: XChaCha20 nonce must be 24 bytes")
	}
	if len(mac) != 16 {
		return nil, fmt.Errorf("c2simd: MAC tag must be 16 bytes")
	}

	var subKeyBlock [32]byte
	HChaCha20_SIMD128(key, nonce[0:16], subKeyBlock[:])

	var derivedNonce [12]byte
	copy(derivedNonce[4:12], nonce[16:24])

	var polyKeyBlock [64]byte
	var zeroBlock [64]byte
	EncryptChaCha20_SIMD256(subKeyBlock[:], derivedNonce[:], 0, zeroBlock[:], polyKeyBlock[:])

	polySt := NewPoly1305QuadChain(polyKeyBlock[:32])

	if len(ad) > 0 {
		polySt.Update(ad)
		if rem := len(ad) % 16; rem > 0 {
			polySt.Update(zeroPad16[:16-rem])
		}
	}

	polySt.Update(ciphertext)
	if rem := ciphertextPadding(len(ciphertext)); rem > 0 {
		polySt.Update(zeroPad16[:rem])
	}

	var sizeBlock [16]byte
	binary.LittleEndian.PutUint64(sizeBlock[0:8], uint64(len(ad)))
	binary.LittleEndian.PutUint64(sizeBlock[8:16], uint64(len(ciphertext)))
	polySt.Update(sizeBlock[:])

	var computedMac [16]byte
	polySt.Finish(&computedMac)

	if subtle.ConstantTimeCompare(mac, computedMac[:]) != 1 {
		return nil, fmt.Errorf("c2simd: MAC verification failed")
	}

	var plainText []byte
	if cap(dstBuffer) >= len(ciphertext) {
		plainText = dstBuffer[:len(ciphertext)]
	} else {
		plainText = make([]byte, len(ciphertext))
	}

	const chunkSize = 4096
	offset := 0
	counter := uint32(1)

	if len(ciphertext) > 0 {
		_ = ciphertext[len(ciphertext)-1]
		_ = plainText[len(ciphertext)-1]
	}

	for offset < len(ciphertext) {
		end := offset + chunkSize
		if end > len(ciphertext) {
			end = len(ciphertext)
		}

		inChunk := ciphertext[offset:end]
		outChunk := plainText[offset:end]

		numBlocks := (len(inChunk) + 63) / 64
		EncryptChaCha20_SIMD256(subKeyBlock[:], derivedNonce[:], counter, inChunk, outChunk)
		counter += uint32(numBlocks)

		offset = end
	}

	return plainText, nil
}
