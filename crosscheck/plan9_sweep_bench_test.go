package crosscheck

// Balayage de tailles des moteurs pur-Go du dépôt contre l'assembleur Plan 9
// de golang.org/x/crypto (XChaCha20-Poly1305, AVX2 dans chacha20poly1305).
// Lancer épinglé P-cores : taskset -c 0-15 go test -bench Sweep -benchtime 2s -count 5 .
// (i9-14900K : E-cores 16-31 à 4,4 GHz rendent des mesures bimodales à ±15 %.)

import (
	"fmt"
	"testing"

	c2 "code.hazyhaar.fr/devhoros/c2simd"
	mc "code.hazyhaar.fr/devhoros/pkg/monocypher55"
	"golang.org/x/crypto/chacha20poly1305"
)

var sweepSizes = []int{32, 64, 256, 1024, 8192, 65536, 1 << 20}

func BenchmarkSweep_Monocypher55(b *testing.B) {
	key := make([]byte, 32)
	nonce := make([]byte, 24)
	for _, n := range sweepSizes {
		plain := make([]byte, n)
		ct := make([]byte, n)
		var mac [16]byte
		b.Run(fmt.Sprintf("n%d", n), func(b *testing.B) {
			b.SetBytes(int64(n))
			for i := 0; i < b.N; i++ {
				if err := mc.LockDst(ct, mac[:], key, nonce, nil, plain); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkSweep_C2simd(b *testing.B) {
	key := make([]byte, 32)
	nonce := make([]byte, 24)
	for _, n := range sweepSizes {
		plain := make([]byte, n)
		dst := make([]byte, n)
		var mac [16]byte
		b.Run(fmt.Sprintf("n%d", n), func(b *testing.B) {
			b.SetBytes(int64(n))
			for i := 0; i < b.N; i++ {
				if _, err := c2.AEADLockDst(dst, &mac, key, nonce, nil, plain); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkSweep_XCryptoPlan9(b *testing.B) {
	key := make([]byte, 32)
	nonce := make([]byte, 24)
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		b.Fatal(err)
	}
	for _, n := range sweepSizes {
		plain := make([]byte, n)
		dst := make([]byte, 0, n+16)
		b.Run(fmt.Sprintf("n%d", n), func(b *testing.B) {
			b.SetBytes(int64(n))
			for i := 0; i < b.N; i++ {
				dst = aead.Seal(dst[:0], nonce, plain, nil)
			}
		})
	}
}
