//go:build goexperiment.simd

package c2simd

import (
	"bytes"
	"testing"
)

// Le point d'entrée à sous-clé pré-dérivée doit produire exactement le même
// couple (ciphertext, mac) que le chemin XChaCha complet, pour toute taille.
func TestAEADSubkey_MatchesFusedXChaCha(t *testing.T) {
	var key [32]byte
	var nonce24 [24]byte
	for i := range key {
		key[i] = byte(i*7 + 3)
	}
	for i := range nonce24 {
		nonce24[i] = byte(0xA0 ^ i)
	}
	ad := []byte("secretstream55-ad")

	var subkey [32]byte
	HChaCha20(key[:], nonce24[0:16], subkey[:])
	var nonce12 [12]byte
	copy(nonce12[4:12], nonce24[16:24])

	for _, n := range []int{0, 1, 15, 16, 63, 64, 65, 4095, 4096, 4097, 65536, 1 << 20} {
		plain := make([]byte, n)
		for i := range plain {
			plain[i] = byte(i * 31)
		}

		var macRef, macSub [16]byte
		ctRef, err := AEADLockSIMD256_FusedDst(nil, &macRef, key[:], nonce24[:], ad, plain)
		if err != nil {
			t.Fatalf("n=%d: fused lock: %v", n, err)
		}
		ctSub, err := AEADSubkeyLockDst(nil, &macSub, subkey[:], nonce12[:], ad, plain)
		if err != nil {
			t.Fatalf("n=%d: subkey lock: %v", n, err)
		}
		if !bytes.Equal(ctRef, ctSub) {
			t.Fatalf("n=%d: ciphertext mismatch", n)
		}
		if macRef != macSub {
			t.Fatalf("n=%d: mac mismatch: %x vs %x", n, macRef, macSub)
		}

		pt, err := AEADSubkeyUnlockDst(nil, subkey[:], nonce12[:], ad, ctSub, macSub[:])
		if err != nil {
			t.Fatalf("n=%d: subkey unlock: %v", n, err)
		}
		if !bytes.Equal(pt, plain) {
			t.Fatalf("n=%d: roundtrip mismatch", n)
		}

		if n > 0 {
			bad := append([]byte(nil), macSub[:]...)
			bad[0] ^= 1
			if _, err := AEADSubkeyUnlockDst(nil, subkey[:], nonce12[:], ad, ctSub, bad); err == nil {
				t.Fatalf("n=%d: corrupted mac accepted", n)
			}
		}
	}
}
