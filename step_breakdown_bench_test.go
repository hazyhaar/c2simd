//go:build goexperiment.simd

package c2simd_test

import (
	"crypto/rand"
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd"
	"golang.org/x/crypto/chacha20"
	"golang.org/x/crypto/poly1305"
)

// -----------------------------------------------------------------------------
// BENCHMARK PAS À PAS : Décomposition fine de chaque étape c2simd vs x/crypto
// -----------------------------------------------------------------------------

func BenchmarkStep1_HChaCha20_Subkey(b *testing.B) {
	key := make([]byte, 32)
	nonce := make([]byte, 24)
	out := make([]byte, 32)
	rand.Read(key)
	rand.Read(nonce)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c2simd.HChaCha20_SIMD128(key, nonce[:16], out)
	}
}

func BenchmarkStep2_PolyKeyGen_ChaCha20Block0(b *testing.B) {
	key := make([]byte, 32)
	nonce := make([]byte, 12)
	zeroBlock := make([]byte, 64)
	polyKeyBlock := make([]byte, 64)
	rand.Read(key)
	rand.Read(nonce)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c2simd.EncryptChaCha20_SIMD256(key, nonce, 0, zeroBlock, polyKeyBlock)
	}
}

func BenchmarkStep3_PayloadEncrypt_ChaCha20_C2SIMD(b *testing.B) {
	key := make([]byte, 32)
	nonce := make([]byte, 12)
	payload := make([]byte, 1024*1024)
	dst := make([]byte, 1024*1024)
	rand.Read(key)
	rand.Read(nonce)
	rand.Read(payload)

	b.SetBytes(int64(len(payload)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c2simd.EncryptChaCha20_SIMD256(key, nonce, 1, payload, dst)
	}
}

func BenchmarkStep3_PayloadEncrypt_ChaCha20_XCrypto_ASM(b *testing.B) {
	key := make([]byte, 32)
	nonce := make([]byte, 12)
	payload := make([]byte, 1024*1024)
	dst := make([]byte, 1024*1024)
	rand.Read(key)
	rand.Read(nonce)
	rand.Read(payload)

	c, _ := chacha20.NewUnauthenticatedCipher(key, nonce)
	b.SetBytes(int64(len(payload)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.XORKeyStream(dst, payload)
	}
}

func BenchmarkStep4_MACTag_Poly1305_C2SIMD_DualChain(b *testing.B) {
	key := make([]byte, 32)
	payload := make([]byte, 1024*1024)
	var mac [16]byte
	rand.Read(key)
	rand.Read(payload)

	b.SetBytes(int64(len(payload)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		st := c2simd.NewPoly1305QuadChain(key)
		st.Update(payload)
		st.Finish(&mac)
	}
}

func BenchmarkStep4_MACTag_Poly1305_XCrypto_ASM(b *testing.B) {
	key := make([]byte, 32)
	payload := make([]byte, 1024*1024)
	var mac [16]byte
	var polyKey [32]byte
	rand.Read(key)
	rand.Read(payload)
	copy(polyKey[:], key)

	b.SetBytes(int64(len(payload)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		poly1305.Sum(&mac, payload, &polyKey)
	}
}
