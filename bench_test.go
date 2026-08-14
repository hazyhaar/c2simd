//go:build goexperiment.simd

package c2simd_test

import (
	"crypto/rand"
	"runtime/debug"
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd"
	"code.hazyhaar.fr/devhoros/c2simd/internal/monocypher"
	"golang.org/x/crypto/chacha20poly1305"
)

func generateBenchData(size int) ([]byte, []byte, []byte, []byte) {
	key := make([]byte, 32)
	nonce := make([]byte, 24)
	ad := []byte("header_metadata_ad")
	payload := make([]byte, size)

	rand.Read(key)
	rand.Read(nonce)
	rand.Read(payload)

	return key, nonce, ad, payload
}

// -----------------------------------------------------------------------------
// 1. Benchmark Monocypher Transpiled Baseline (ccgo) - Lock Only
// -----------------------------------------------------------------------------

func BenchmarkMonocypher_Lock_1MB(b *testing.B) {
	key, nonce, ad, payload := generateBenchData(1024 * 1024)
	b.SetBytes(int64(len(payload)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _, err := monocypher.AEADLock(key, nonce, ad, payload)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// -----------------------------------------------------------------------------
// 2. Benchmark Moteur c2simd Transformé (bits.RotateLeft32) - Lock Only
// -----------------------------------------------------------------------------

func BenchmarkC2SIMD_Lock_1MB(b *testing.B) {
	key, nonce, ad, payload := generateBenchData(1024 * 1024)
	b.SetBytes(int64(len(payload)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _, err := c2simd.AEADLock(key, nonce, ad, payload)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// -----------------------------------------------------------------------------
// 3. Benchmark Fused SIMD256 Engine (0 libc.TLS - Fused AEAD) - Lock Only
// -----------------------------------------------------------------------------

func BenchmarkC2SIMD_FusedSIMD256_Lock_1MB(b *testing.B) {
	key, nonce, ad, payload := generateBenchData(1024 * 1024)
	dstBuffer := make([]byte, len(payload))
	var mac [16]byte
	b.SetBytes(int64(len(payload)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := c2simd.AEADLockSIMD256_FusedDst(dstBuffer, &mac, key, nonce, ad, payload)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkC2SIMD_FusedSIMD256_Lock_1MB_ZeroAlloc_NoGC(b *testing.B) {
	key, nonce, ad, payload := generateBenchData(1024 * 1024)
	dstBuffer := make([]byte, len(payload))
	var mac [16]byte

	// Contrôle explicite du GC pour éliminer tout balayage résiduel
	oldGC := debug.SetGCPercent(-1)
	defer debug.SetGCPercent(oldGC)

	b.SetBytes(int64(len(payload)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c2simd.AEADLockSIMD256_FusedDst(dstBuffer, &mac, key, nonce, ad, payload)
	}
}

// -----------------------------------------------------------------------------
// 4. Benchmark Chiffrement SIMD256 Pur AVX2 (archsimd.Uint32x8 - ChaCha20 Seul)
// -----------------------------------------------------------------------------

func BenchmarkC2SIMD_SIMD256_PureStream_1MB(b *testing.B) {
	key, nonce, _, payload := generateBenchData(1024 * 1024)
	dst := make([]byte, len(payload))
	b.SetBytes(int64(len(payload)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c2simd.EncryptChaCha20_SIMD256(key, nonce[:12], 1, payload, dst)
	}
}

// -----------------------------------------------------------------------------
// 5. Benchmark Native Go SIMD (golang.org/x/crypto/chacha20poly1305) - Seal Only
// -----------------------------------------------------------------------------

func BenchmarkNativeGo_Seal_1MB(b *testing.B) {
	key, nonce, ad, payload := generateBenchData(1024 * 1024)
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		b.Fatal(err)
	}

	b.SetBytes(int64(len(payload)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = aead.Seal(nil, nonce, payload, ad)
	}
}
