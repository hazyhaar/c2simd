//go:build goexperiment.simd

// Micro-spike de MESURE (2026-08-15) : Poly1305 vectoriel du dépôt (Poly1305_AVX2_Engine,
// donna-26) contre le scalaire émis de référence (Poly1305QuadChain) et l'assembleur
// x/crypto. Parité de tag vérifiée avant toute mesure. Aucun câblage.
package c2simd_test

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd"
	"golang.org/x/crypto/poly1305"
)

var spikeSizes = []int{1024, 8 * 1024, 64 * 1024, 1024 * 1024}

// TestSpike_Poly1305_Parity vérifie la parité du tag des deux voies du dépôt
// contre x/crypto sur des tailles variées (alignées et non alignées).
func TestSpike_Poly1305_Parity(t *testing.T) {
	sizes := []int{0, 1, 15, 16, 17, 63, 64, 65, 127, 128, 129, 255, 256, 257,
		1023, 1024, 1025, 8*1024 - 3, 8 * 1024, 64 * 1024, 64*1024 + 7, 1024 * 1024}

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

		vec := c2simd.NewPoly1305AVX2(key)
		vec.Update(msg)
		var vecTag [16]byte
		vec.Finish(&vecTag)
		if !bytes.Equal(vecTag[:], expTag[:]) {
			t.Fatalf("PARITE FAIL voie vectorielle, taille %d: got %x exp %x", sz, vecTag, expTag)
		}

		sc := c2simd.NewPoly1305QuadChain(key)
		sc.Update(msg)
		var scTag [16]byte
		sc.Finish(&scTag)
		if !bytes.Equal(scTag[:], expTag[:]) {
			t.Fatalf("PARITE FAIL scalaire émis, taille %d: got %x exp %x", sz, scTag, expTag)
		}
	}
}

func spikePayload(sz int) ([]byte, []byte) {
	key := make([]byte, 32)
	payload := make([]byte, sz)
	rand.Read(key)
	rand.Read(payload)
	return key, payload
}

func BenchmarkSpike_Poly1305_Vector(b *testing.B) {
	for _, sz := range spikeSizes {
		b.Run(fmt.Sprintf("%dB", sz), func(b *testing.B) {
			key, payload := spikePayload(sz)
			var mac [16]byte
			b.SetBytes(int64(sz))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				st := c2simd.NewPoly1305AVX2(key)
				st.Update(payload)
				st.Finish(&mac)
			}
		})
	}
}

func BenchmarkSpike_Poly1305_Scalar(b *testing.B) {
	for _, sz := range spikeSizes {
		b.Run(fmt.Sprintf("%dB", sz), func(b *testing.B) {
			key, payload := spikePayload(sz)
			var mac [16]byte
			b.SetBytes(int64(sz))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				st := c2simd.NewPoly1305QuadChain(key)
				st.Update(payload)
				st.Finish(&mac)
			}
		})
	}
}

func BenchmarkSpike_Poly1305_XCryptoASM(b *testing.B) {
	for _, sz := range spikeSizes {
		b.Run(fmt.Sprintf("%dB", sz), func(b *testing.B) {
			key, payload := spikePayload(sz)
			var polyKey [32]byte
			copy(polyKey[:], key)
			var mac [16]byte
			b.SetBytes(int64(sz))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				poly1305.Sum(&mac, payload, &polyKey)
			}
		})
	}
}
