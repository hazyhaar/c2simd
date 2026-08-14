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
// BENCHMARK SCOPÉ N°1 : ChaCha20 Pure Stream (ASM .s vs Pure Go SIMD256)
// -----------------------------------------------------------------------------

func BenchmarkScoped_ChaCha20_XCrypto_ASM(b *testing.B) {
	key := make([]byte, 32)
	nonce := make([]byte, 12)
	payload := make([]byte, 1024*1024)
	dst := make([]byte, 1024*1024)

	rand.Read(key)
	rand.Read(nonce)
	rand.Read(payload)

	c, err := chacha20.NewUnauthenticatedCipher(key, nonce)
	if err != nil {
		b.Fatal(err)
	}

	b.SetBytes(int64(len(payload)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c.XORKeyStream(dst, payload)
	}
}

func BenchmarkScoped_ChaCha20_C2SIMD_PureGo(b *testing.B) {
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

// -----------------------------------------------------------------------------
// BENCHMARK SCOPÉ N°2 : Poly1305 MAC Isolé (OctaChain 8-Way vs ASM)
// -----------------------------------------------------------------------------

func BenchmarkScoped_Poly1305_C2SIMD_OriginalScalar(b *testing.B) {
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

func BenchmarkScoped_Poly1305_XCrypto_ASM(b *testing.B) {
	key := make([]byte, 32)
	payload := make([]byte, 1024*1024)
	var mac [16]byte

	rand.Read(key)
	rand.Read(payload)

	var polyKey [32]byte
	copy(polyKey[:], key)

	b.SetBytes(int64(len(payload)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		poly1305.Sum(&mac, payload, &polyKey)
	}
}
