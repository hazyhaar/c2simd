//go:build goexperiment.simd

package c2simd

import (
	"crypto/subtle"
	"encoding/binary"
	"fmt"
)

// AEADSubkeyLockDst scelle en ChaCha20-IETF-Poly1305 (RFC 8439) avec une
// sous-clé DÉJÀ dérivée (HChaCha20) et un nonce IETF de 12 octets.
// C'est le corps de AEADLockSIMD256_FusedDst sans l'étape HChaCha20 : pour un
// stream dont les 16 premiers octets du nonce XChaCha sont constants, la
// sous-clé se dérive une seule fois et ce point d'entrée évite de la recalculer
// à chaque chunk. Équivalence : AEADLockSIMD256_FusedDst(key, nonce24) ==
// AEADSubkeyLockDst(HChaCha20(key, nonce24[0:16]), 0x00000000||nonce24[16:24]).
func AEADSubkeyLockDst(dstBuffer []byte, outMac *[16]byte, subkey, nonce12, ad, plaintext []byte) ([]byte, error) {
	if len(subkey) != 32 {
		return nil, fmt.Errorf("c2simd: subkey must be 32 bytes")
	}
	if len(nonce12) != 12 {
		return nil, fmt.Errorf("c2simd: IETF nonce must be 12 bytes")
	}

	// Clé Poly1305 (Counter 0)
	var polyKeyBlock [64]byte
	var zeroBlock [64]byte
	EncryptChaCha20_SIMD256(subkey, nonce12, 0, zeroBlock[:], polyKeyBlock[:])

	polySt := newPolyMAC(polyKeyBlock[:32], len(plaintext)+len(ad))

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

	const chunkSize = 4096
	offset := 0
	counter := uint32(1)

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

		numBlocks := (len(inChunk) + 63) / 64
		EncryptChaCha20_SIMD256(subkey, nonce12, counter, inChunk, outChunk)
		counter += uint32(numBlocks)

		polySt.Update(outChunk)

		offset = end
	}

	if rem := ciphertextPadding(len(plaintext)); rem > 0 {
		polySt.Update(zeroPad16[:rem])
	}

	var sizeBlock [16]byte
	binary.LittleEndian.PutUint64(sizeBlock[0:8], uint64(len(ad)))
	binary.LittleEndian.PutUint64(sizeBlock[8:16], uint64(len(plaintext)))
	polySt.Update(sizeBlock[:])

	polySt.Finish(outMac)

	return cipherText, nil
}

// AEADSubkeyUnlockDst vérifie puis déchiffre (fail-closed) avec sous-clé
// pré-dérivée et nonce IETF 12 octets. Miroir de AEADUnlockSIMD256_FusedDst
// sans l'étape HChaCha20.
func AEADSubkeyUnlockDst(dstBuffer []byte, subkey, nonce12, ad, ciphertext, mac []byte) ([]byte, error) {
	if len(subkey) != 32 {
		return nil, fmt.Errorf("c2simd: subkey must be 32 bytes")
	}
	if len(nonce12) != 12 {
		return nil, fmt.Errorf("c2simd: IETF nonce must be 12 bytes")
	}
	if len(mac) != 16 {
		return nil, fmt.Errorf("c2simd: MAC tag must be 16 bytes")
	}

	var polyKeyBlock [64]byte
	var zeroBlock [64]byte
	EncryptChaCha20_SIMD256(subkey, nonce12, 0, zeroBlock[:], polyKeyBlock[:])

	polySt := newPolyMAC(polyKeyBlock[:32], len(ciphertext)+len(ad))

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
		EncryptChaCha20_SIMD256(subkey, nonce12, counter, inChunk, outChunk)
		counter += uint32(numBlocks)

		offset = end
	}

	return plainText, nil
}
