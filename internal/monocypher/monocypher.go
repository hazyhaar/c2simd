package monocypher

import (
	"errors"
	"unsafe"

	"modernc.org/libc"
)

var (
	ErrAEADCheckFailed = errors.New("monocypher: AEAD authentication check failed")
)

func getPtr(b []byte) uintptr {
	if len(b) == 0 {
		return 0
	}
	return uintptr(unsafe.Pointer(unsafe.SliceData(b)))
}

// AEADLock encrypts plainText and produces cipherText and mac using XChaCha20-Poly1305 in Pure Go (0 allocation).
func AEADLock(key []byte, nonce []byte, ad []byte, plainText []byte) (cipherText []byte, mac []byte, err error) {
	if len(key) != 32 {
		return nil, nil, errors.New("monocypher: key must be 32 bytes")
	}
	if len(nonce) != 24 {
		return nil, nil, errors.New("monocypher: XChaCha20 nonce must be 24 bytes")
	}

	tls := libc.NewTLS()
	defer tls.Close()

	cKey := getPtr(key)
	cNonce := getPtr(nonce)
	cAD := getPtr(ad)
	cPlain := getPtr(plainText)

	mac = make([]byte, 16)
	cMAC := getPtr(mac)

	cipherText = make([]byte, len(plainText))
	cCipher := getPtr(cipherText)

	crypto_aead_lock(
		tls,
		cCipher,
		cMAC,
		cKey,
		cNonce,
		cAD,
		size_t(len(ad)),
		cPlain,
		size_t(len(plainText)),
	)

	return cipherText, mac, nil
}

// AEADUnlock decrypts cipherText and verifies mac using XChaCha20-Poly1305 in Pure Go.
func AEADUnlock(key []byte, nonce []byte, mac []byte, ad []byte, cipherText []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, errors.New("monocypher: key must be 32 bytes")
	}
	if len(nonce) != 24 {
		return nil, errors.New("monocypher: XChaCha20 nonce must be 24 bytes")
	}
	if len(mac) != 16 {
		return nil, errors.New("monocypher: mac must be 16 bytes")
	}

	tls := libc.NewTLS()
	defer tls.Close()

	cKey := getPtr(key)
	cNonce := getPtr(nonce)
	cMAC := getPtr(mac)
	cAD := getPtr(ad)
	cCipher := getPtr(cipherText)

	plainText := make([]byte, len(cipherText))
	cPlain := getPtr(plainText)

	res := crypto_aead_unlock(
		tls,
		cPlain,
		cMAC,
		cKey,
		cNonce,
		cAD,
		size_t(len(ad)),
		cCipher,
		size_t(len(cipherText)),
	)

	if res != 0 {
		return nil, ErrAEADCheckFailed
	}

	return plainText, nil
}
