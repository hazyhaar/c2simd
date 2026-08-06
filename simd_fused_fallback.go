//go:build !goexperiment.simd

package c2simd

import (
	"fmt"

	"golang.org/x/crypto/chacha20"
	"golang.org/x/crypto/chacha20poly1305"
)

// HChaCha20_SIMD128 (Fallback Scalaire HChaCha20)
func HChaCha20_SIMD128(key, nonce, out []byte) {
	subKey, err := chacha20.HChaCha20(key, nonce)
	if err != nil {
		return
	}
	copy(out, subKey)
}

// HChaCha20 (Fallback Scalaire HChaCha20)
func HChaCha20(key, nonce, out []byte) {
	HChaCha20_SIMD128(key, nonce, out)
}

// AEADLockSIMD256_Fused (Fallback Scalaire Sécurisé XChaCha20-Poly1305)
func AEADLockSIMD256_Fused(key, nonce, ad, plaintext []byte) ([]byte, []byte, error) {
	var mac [16]byte
	cipherText, err := AEADLockSIMD256_FusedDst(nil, &mac, key, nonce, ad, plaintext)
	return cipherText, mac[:], err
}

// AEADLockSIMD256_FusedDst (Fallback Scalaire Sécurisé XChaCha20-Poly1305 avec réutilisation Dst)
func AEADLockSIMD256_FusedDst(dstBuffer []byte, outMac *[16]byte, key, nonce, ad, plaintext []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("c2simd: key must be 32 bytes")
	}
	if len(nonce) != 24 {
		return nil, fmt.Errorf("c2simd: XChaCha20 nonce must be 24 bytes")
	}

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, fmt.Errorf("c2simd: failed to initialize XChaCha20-Poly1305: %w", err)
	}

	sealed := aead.Seal(nil, nonce, plaintext, ad)
	cipherLen := len(sealed) - 16
	if cipherLen < 0 {
		return nil, fmt.Errorf("c2simd: invalid sealed payload length")
	}

	var cipherText []byte
	if cap(dstBuffer) >= cipherLen {
		cipherText = dstBuffer[:cipherLen]
	} else {
		cipherText = make([]byte, cipherLen)
	}

	copy(cipherText, sealed[:cipherLen])
	copy(outMac[:], sealed[cipherLen:])
	return cipherText, nil
}

// AEADUnlockSIMD256_Fused (Fallback Scalaire Déchiffrement Authentifié fail-closed)
func AEADUnlockSIMD256_Fused(key, nonce, ad, ciphertext, mac []byte) ([]byte, error) {
	return AEADUnlockSIMD256_FusedDst(nil, key, nonce, ad, ciphertext, mac)
}

// AEADUnlockSIMD256_FusedDst (Fallback Scalaire Déchiffrement Authentifié fail-closed Dst)
func AEADUnlockSIMD256_FusedDst(dstBuffer []byte, key, nonce, ad, ciphertext, mac []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("c2simd: key must be 32 bytes")
	}
	if len(nonce) != 24 {
		return nil, fmt.Errorf("c2simd: XChaCha20 nonce must be 24 bytes")
	}
	if len(mac) != 16 {
		return nil, fmt.Errorf("c2simd: Poly1305 MAC must be 16 bytes")
	}

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, fmt.Errorf("c2simd: failed to initialize XChaCha20-Poly1305: %w", err)
	}

	sealed := make([]byte, len(ciphertext)+16)
	copy(sealed[:len(ciphertext)], ciphertext)
	copy(sealed[len(ciphertext):], mac)

	plainText, err := aead.Open(nil, nonce, sealed, ad)
	if err != nil {
		return nil, fmt.Errorf("c2simd: authentication tag verification failed: %w", err)
	}

	var outText []byte
	if cap(dstBuffer) >= len(plainText) {
		outText = dstBuffer[:len(plainText)]
	} else {
		outText = make([]byte, len(plainText))
	}
	copy(outText, plainText)
	return outText, nil
}
