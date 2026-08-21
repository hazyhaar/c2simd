// Package crosscheck verrouille les invariants INTER-moteurs du pôle AEAD :
// les deux implémentations XChaCha20-Poly1305 du dépôt (c2simd fused et
// monocypher55) doivent produire un fil bit-identique et accepter mutuellement
// leurs sorties. Module de test dédié : aucun des deux moteurs ne peut dépendre
// de l'autre sans cycle, l'invariant vit donc au-dessus.
//
// Provenance : oracle forgé en session d'audit 2026-08-15 (scratchpad), promu
// au dépôt le 2026-08-15 — une preuve de gate non versionnée n'est pas une preuve.
package crosscheck

import (
	"bytes"
	"testing"

	c2 "code.hazyhaar.fr/devhoros/c2simd"
	mc "code.hazyhaar.fr/devhoros/pkg/monocypher55"
)

// TestCrossEngineWireCompat chiffre avec un moteur et déchiffre avec l'autre,
// dans les deux sens, et exige en outre l'égalité octet par octet du fil
// (ciphertext ET mac) entre les deux moteurs.
func TestCrossEngineWireCompat(t *testing.T) {
	key := make([]byte, 32)
	nonce := make([]byte, 24)
	for i := range key {
		key[i] = byte(i + 1)
	}
	for i := range nonce {
		nonce[i] = byte(0x30 + i)
	}
	sizes := []int{0, 1, 48, 64, 200, 255, 256, 257, 512, 65536, 1 << 20}
	ads := [][]byte{nil, []byte("ad-croise")}
	for _, n := range sizes {
		for _, ad := range ads {
			plain := make([]byte, n)
			for i := range plain {
				plain[i] = byte(i * 13)
			}

			// c2simd -> monocypher55
			var mac1 [16]byte
			ct1, err := c2.AEADLockDst(nil, &mac1, key, nonce, ad, plain)
			if err != nil {
				t.Fatalf("c2simd lock n=%d : %v", n, err)
			}
			dst1 := make([]byte, n)
			if err := mc.UnlockDst(dst1, key, nonce, mac1[:], ad, ct1); err != nil {
				t.Fatalf("c2simd->monocypher55 n=%d ad=%v : %v", n, ad != nil, err)
			}
			if !bytes.Equal(dst1, plain) {
				t.Fatalf("c2simd->monocypher55 n=%d : plaintext différent", n)
			}

			// monocypher55 -> c2simd
			ct2, mac2, err := mc.AEADLock(key, nonce, ad, plain)
			if err != nil {
				t.Fatalf("monocypher55 lock n=%d : %v", n, err)
			}
			dst2, err := c2.AEADUnlockDst(nil, key, nonce, ad, ct2, mac2)
			if err != nil {
				t.Fatalf("monocypher55->c2simd n=%d ad=%v : %v", n, ad != nil, err)
			}
			if !bytes.Equal(dst2, plain) {
				t.Fatalf("monocypher55->c2simd n=%d : plaintext différent", n)
			}

			// Fil bit-identique entre les deux moteurs.
			if !bytes.Equal(ct1, ct2) || !bytes.Equal(mac1[:], mac2) {
				t.Fatalf("fil divergent n=%d ad=%v : ct_egal=%v mac_egal=%v",
					n, ad != nil, bytes.Equal(ct1, ct2), bytes.Equal(mac1[:], mac2))
			}
		}
	}
}

// TestCrossEngineRejectForgery vérifie que les DEUX moteurs refusent le même
// fil corrompu (divergence de sémantique d'acceptation = défaut de gate).
func TestCrossEngineRejectForgery(t *testing.T) {
	key := make([]byte, 32)
	nonce := make([]byte, 24)
	plain := make([]byte, 512)
	var mac [16]byte
	ct, err := c2.AEADLockDst(nil, &mac, key, nonce, nil, plain)
	if err != nil {
		t.Fatal(err)
	}
	for _, flip := range []struct {
		name string
		mut  func(c []byte, m []byte)
	}{
		{"ct_bit", func(c, m []byte) { c[100] ^= 1 }},
		{"mac_bit", func(c, m []byte) { m[0] ^= 0x80 }},
	} {
		c := append([]byte(nil), ct...)
		m := append([]byte(nil), mac[:]...)
		flip.mut(c, m)
		if _, err := c2.AEADUnlockDst(nil, key, nonce, nil, c, m); err == nil {
			t.Fatalf("%s : c2simd accepte un fil corrompu", flip.name)
		}
		dst := make([]byte, len(c))
		if err := mc.UnlockDst(dst, key, nonce, m, nil, c); err == nil {
			t.Fatalf("%s : monocypher55 accepte un fil corrompu", flip.name)
		}
	}
}
