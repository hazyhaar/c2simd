package c2simd_test

import (
	"bytes"
	"testing"

	"github.com/hazyhaar/c2simd"
	"golang.org/x/crypto/chacha20"
)

func TestHChaCha20_MatchesXCrypto(t *testing.T) {
	key := make([]byte, 32)
	nonce := make([]byte, 16)
	for i := range key {
		key[i] = byte(i)
	}
	for i := range nonce {
		nonce[i] = byte(i * 3)
	}

	want, err := chacha20.HChaCha20(key, nonce)
	if err != nil {
		t.Fatal(err)
	}

	var out [32]byte
	c2simd.HChaCha20(key, nonce, out[:])
	if !bytes.Equal(out[:], want) {
		t.Fatalf("HChaCha20 divergence\n got %x\nwant %x", out[:], want)
	}

	var out2 [32]byte
	c2simd.HChaCha20_SIMD128(key, nonce, out2[:])
	if !bytes.Equal(out2[:], want) {
		t.Fatalf("HChaCha20_SIMD128 divergence\n got %x\nwant %x", out2[:], want)
	}
}

func TestHChaCha20_RejectsInvalidLengths(t *testing.T) {
	cases := []struct {
		name  string
		key   int
		nonce int
		out   int
	}{
		{"short_key", 16, 16, 32},
		{"short_nonce", 32, 8, 32},
		{"short_out", 32, 16, 16},
		{"empty", 0, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("expected panic on invalid lengths")
				}
			}()
			c2simd.HChaCha20(make([]byte, tc.key), make([]byte, tc.nonce), make([]byte, tc.out))
		})
	}
}
