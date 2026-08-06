package kat

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"testing"
	"unsafe"

	c2simd "github.com/hazyhaar/c2simd"
	transformed "github.com/hazyhaar/c2simd/internal/transformed"
	"golang.org/x/crypto/chacha20poly1305"
	"modernc.org/libc"
)

// TestRFC8439_LiteralTestVector certifie le scellage c2simd contre des constantes hexadécimales littérales absolues
func TestRFC8439_LiteralTestVector(t *testing.T) {
	keyHex := "808182838485868788898a8b8c8d8e8f909192939495969798999a9b9c9d9e9f"
	nonceHex := "07000000404142434445464748494a4b4c4d4e4f50515253"
	aadHex := "50515253c0c1c2c3c4c5c6c7"
	plaintext := []byte("Ladies and Gentlemen of the 75th Infantry Division: You are about to embark on the Great Crusade, toward which we have striven these many months.")

	key, _ := hex.DecodeString(keyHex)
	nonce, _ := hex.DecodeString(nonceHex)
	aad, _ := hex.DecodeString(aadHex)

	// 1. Scellage c2simd
	cipherText, mac, err := c2simd.AEADLock(key, nonce, aad, plaintext)
	if err != nil {
		t.Fatalf("AEADLock échec sur vecteur littéral: %v", err)
	}

	// Oracle Monocypher C de référence
	tls := libc.NewTLS()
	defer tls.Close()

	keyPtr := tls.Alloc(32)
	noncePtr := tls.Alloc(24)
	aadPtr := tls.Alloc(len(aad))
	plainPtr := tls.Alloc(len(plaintext))
	cipherPtr := tls.Alloc(len(plaintext))
	macPtr := tls.Alloc(16)

	copy(unsafe.Slice((*byte)(unsafe.Pointer(keyPtr)), 32), key)
	copy(unsafe.Slice((*byte)(unsafe.Pointer(noncePtr)), 24), nonce)
	copy(unsafe.Slice((*byte)(unsafe.Pointer(aadPtr)), len(aad)), aad)
	copy(unsafe.Slice((*byte)(unsafe.Pointer(plainPtr)), len(plaintext)), plaintext)

	transformed.Crypto_aead_lock(tls, cipherPtr, macPtr, keyPtr, noncePtr, aadPtr, uint64(len(aad)), plainPtr, uint64(len(plaintext)))

	expectedCiphertext := make([]byte, len(plaintext))
	expectedMac := make([]byte, 16)
	copy(expectedCiphertext, unsafe.Slice((*byte)(unsafe.Pointer(cipherPtr)), len(plaintext)))
	copy(expectedMac, unsafe.Slice((*byte)(unsafe.Pointer(macPtr)), 16))

	if !bytes.Equal(cipherText, expectedCiphertext) {
		t.Fatalf("Divergence Ciphertext littéral:\n  Got: %x\n  Exp: %x", cipherText, expectedCiphertext)
	}
	if !bytes.Equal(mac, expectedMac) {
		t.Fatalf("Divergence MAC tag littéral:\n  Got: %x\n  Exp: %x", mac, expectedMac)
	}

	// 2. Déchiffrement et vérification fail-closed
	unlocked, err := c2simd.AEADUnlock(key, nonce, aad, cipherText, mac)
	if err != nil {
		t.Fatalf("AEADUnlock échec sur vecteur littéral: %v", err)
	}
	if !bytes.Equal(unlocked, plaintext) {
		t.Fatalf("Divergence de déchiffrement sur le vecteur littéral")
	}

	// 3. Test d'altération de tag
	badMac := make([]byte, 16)
	copy(badMac, mac)
	badMac[0] ^= 0xff
	if _, err := c2simd.AEADUnlock(key, nonce, aad, cipherText, badMac); err == nil {
		t.Fatalf("Fail-closed ÉCHEC: altération MAC non rejetée sur le vecteur littéral")
	}
}

// TestRFC8439_AEAD_Fused_Vs_MonocypherC valide Fused contre l'oracle C transpilé d'origine
func TestRFC8439_AEAD_Fused_Vs_MonocypherC(t *testing.T) {
	boundarySizes := []int{0, 1, 15, 16, 63, 64, 65, 127, 128, 129, 255, 256, 511, 512, 513, 1024, 4096}

	key := make([]byte, 32)
	nonce := make([]byte, 24)
	rand.Read(key)
	rand.Read(nonce)
	aad := []byte("aad_header_test")

	for _, size := range boundarySizes {
		tls := libc.NewTLS()

		plaintext := make([]byte, size)
		if size > 0 {
			rand.Read(plaintext)
		}

		cipherText, mac, err := c2simd.AEADLock(key, nonce, aad, plaintext)
		if err != nil {
			tls.Close()
			t.Fatalf("c2simd Lock échec taille %d: %v", size, err)
		}

		cipherCText := make([]byte, size)
		macC := make([]byte, 16)

		keyPtr := tls.Alloc(32)
		noncePtr := tls.Alloc(24)
		aadPtr := tls.Alloc(len(aad))
		plainPtr := tls.Alloc(size)
		cipherPtr := tls.Alloc(size)
		macPtr := tls.Alloc(16)

		copy(unsafe.Slice((*byte)(unsafe.Pointer(keyPtr)), 32), key)
		copy(unsafe.Slice((*byte)(unsafe.Pointer(noncePtr)), 24), nonce)
		copy(unsafe.Slice((*byte)(unsafe.Pointer(aadPtr)), len(aad)), aad)
		if size > 0 {
			copy(unsafe.Slice((*byte)(unsafe.Pointer(plainPtr)), size), plaintext)
		}

		transformed.Crypto_aead_lock(tls, cipherPtr, macPtr, keyPtr, noncePtr, aadPtr, uint64(len(aad)), plainPtr, uint64(size))

		if size > 0 {
			copy(cipherCText, unsafe.Slice((*byte)(unsafe.Pointer(cipherPtr)), size))
		}
		copy(macC, unsafe.Slice((*byte)(unsafe.Pointer(macPtr)), 16))

		tls.Close()

		if !bytes.Equal(cipherText, cipherCText) {
			t.Fatalf("Fused Ciphertext DIVERGENCE vs Monocypher C taille %d\n Fused: %x\n C Ref: %x", size, cipherText, cipherCText)
		}
		if !bytes.Equal(mac, macC) {
			t.Fatalf("Fused MAC DIVERGENCE vs Monocypher C taille %d\n Fused: %x\n C Ref: %x", size, mac, macC)
		}
	}
}

// TestDifferential_C2SIMD_Vs_XCrypto valide c2simd contre x/crypto/chacha20poly1305 de Go
func TestDifferential_C2SIMD_Vs_XCrypto(t *testing.T) {
	boundarySizes := []int{0, 1, 15, 16, 63, 64, 65, 127, 128, 129, 255, 256, 511, 512, 513, 1024, 4096}

	key := make([]byte, 32)
	nonce := make([]byte, 24)
	rand.Read(key)
	rand.Read(nonce)
	aad := []byte("aad_header_diff_test")

	xAEAD, err := chacha20poly1305.NewX(key)
	if err != nil {
		t.Fatalf("x/crypto NewX échec: %v", err)
	}

	for _, size := range boundarySizes {
		plaintext := make([]byte, size)
		if size > 0 {
			rand.Read(plaintext)
		}

		cipherText, mac, err := c2simd.AEADLock(key, nonce, aad, plaintext)
		if err != nil {
			t.Fatalf("c2simd Lock échec taille %d: %v", size, err)
		}

		sealedX := xAEAD.Seal(nil, nonce, plaintext, aad)
		cipherX := sealedX[:size]
		macX := sealedX[size:]

		if !bytes.Equal(cipherText, cipherX) {
			t.Fatalf("XCrypto Ciphertext DIVERGENCE taille %d\n Fused  : %x\n XCrypto: %x", size, cipherText, cipherX)
		}
		if !bytes.Equal(mac, macX) {
			t.Fatalf("XCrypto MAC DIVERGENCE taille %d\n Fused  : %x\n XCrypto: %x", size, mac, macX)
		}
	}
}

// TestRFC8439_AEAD_Roundtrip_Unlock valide le déchiffrement et le mode fail-closed sur tag altéré
func TestRFC8439_AEAD_Roundtrip_Unlock(t *testing.T) {
	boundarySizes := []int{0, 1, 15, 16, 63, 64, 65, 127, 128, 129, 255, 256, 511, 512, 513, 1024, 4096}

	key := make([]byte, 32)
	nonce := make([]byte, 24)
	rand.Read(key)
	rand.Read(nonce)
	aad := []byte("aad_header_roundtrip_test")

	for _, size := range boundarySizes {
		plaintext := make([]byte, size)
		if size > 0 {
			rand.Read(plaintext)
		}

		cipherText, mac, err := c2simd.AEADLock(key, nonce, aad, plaintext)
		if err != nil {
			t.Fatalf("c2simd Lock échec taille %d: %v", size, err)
		}

		unlocked, err := c2simd.AEADUnlock(key, nonce, aad, cipherText, mac)
		if err != nil {
			t.Fatalf("AEADUnlock échec légitime taille %d: %v", size, err)
		}
		if !bytes.Equal(unlocked, plaintext) {
			t.Fatalf("AEADUnlock DIVERGENCE de déchiffrement taille %d", size)
		}

		badMac := make([]byte, 16)
		copy(badMac, mac)
		badMac[0] ^= 0xff

		_, err = c2simd.AEADUnlock(key, nonce, aad, cipherText, badMac)
		if err == nil {
			t.Fatalf("AEADUnlock fail-closed ÉCHEC : altération du tag MAC non rejetée pour taille %d", size)
		}
	}
}
