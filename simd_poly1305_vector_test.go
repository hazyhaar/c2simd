//go:build goexperiment.simd

package c2simd_test

import (
	"bytes"
	"crypto/rand"
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd"
	"golang.org/x/crypto/poly1305"
)

func TestKAT_Poly1305_AVX2_Vs_XCrypto_ExtendedSizes(t *testing.T) {
	sizes := []int{0, 1, 15, 16, 17, 31, 32, 33, 63, 64, 65, 127, 128, 129, 255, 256, 511, 512, 1000, 1024, 4096}

	key := make([]byte, 32)
	rand.Read(key)

	var keyArr [32]byte
	copy(keyArr[:], key)

	for _, sz := range sizes {
		msg := make([]byte, sz)
		if sz > 0 {
			rand.Read(msg)
		}

		var expTag [16]byte
		poly1305.Sum(&expTag, msg, &keyArr)

		engine := c2simd.NewPoly1305AVX2(key)
		engine.Update(msg)

		var gotTag [16]byte
		engine.Finish(&gotTag)

		if !bytes.Equal(gotTag[:], expTag[:]) {
			t.Fatalf("KAT DIVERGENCE taille %d:\n Got: %x\n Exp: %x", sz, gotTag, expTag)
		}
	}
}

// TestKAT_Poly1305_AVX2_CanonicalBoundary Edge-case test provoquant H >= 2^130 - 5
func TestKAT_Poly1305_AVX2_CanonicalBoundary(t *testing.T) {
	var key [32]byte
	key[0] = 1

	msg := bytes.Repeat([]byte{0xff}, 16)

	var expTag [16]byte
	poly1305.Sum(&expTag, msg, &key)

	engine := c2simd.NewPoly1305AVX2(key[:])
	engine.Update(msg)

	var gotTag [16]byte
	engine.Finish(&gotTag)

	if !bytes.Equal(gotTag[:], expTag[:]) {
		t.Fatalf("DIVERGENCE CANONIQUE H >= 2^130-5:\n Got: %x\n Exp: %x", gotTag, expTag)
	}
}

func BenchmarkPoly1305_AVX2_VPMULUDQ_1MB(b *testing.B) {
	key := make([]byte, 32)
	payload := make([]byte, 1024*1024)
	rand.Read(key)
	rand.Read(payload)

	st := c2simd.NewPoly1305AVX2(key)

	b.SetBytes(int64(len(payload)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		st.Update(payload)
	}
}
