package sgoiter

import (
	"bufio"
	"crypto/hmac"
	"encoding/hex"
	"encoding/json"
	"hash"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/chacha20"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/poly1305"
)

type vector struct {
	ID       string `json:"id"`
	Alg      string `json:"alg"`
	KeyHex   string `json:"key_hex"`
	NonceHex string `json:"nonce_hex"`
	ADHex    string `json:"ad_hex"`
	PTHex    string `json:"pt_hex"`
	CTHex    string `json:"ct_hex"`
	MACHex   string `json:"mac_hex"`
}

func TestMonocypherRFCPack(t *testing.T) {
	path := filepath.Join("testdata", "monocypher_rfc", "vectors.jsonl")
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("unable to open vectors pack: %v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	count := 0
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var vec vector
		if err := json.Unmarshal(line, &vec); err != nil {
			t.Fatalf("invalid json line %d: %v", count+1, err)
		}
		count++

		key, _ := hex.DecodeString(vec.KeyHex)
		nonce, _ := hex.DecodeString(vec.NonceHex)
		ad, _ := hex.DecodeString(vec.ADHex)
		pt, _ := hex.DecodeString(vec.PTHex)
		ctExpected, _ := hex.DecodeString(vec.CTHex)
		macExpected, _ := hex.DecodeString(vec.MACHex)

		switch vec.Alg {
		case "aead_ietf":
			aead, err := chacha20poly1305.New(key)
			if err != nil {
				t.Fatalf("[%s] chacha20poly1305.New failed: %v", vec.ID, err)
			}
			out := aead.Seal(nil, nonce, pt, ad)
			gotCT := out[:len(pt)]
			gotMAC := out[len(pt):]
			if !bytesEqual(gotCT, ctExpected) {
				t.Errorf("[%s] CT mismatch: got %x, want %s", vec.ID, gotCT, vec.CTHex)
			}
			if !bytesEqual(gotMAC, macExpected) {
				t.Errorf("[%s] MAC mismatch: got %x, want %s", vec.ID, gotMAC, vec.MACHex)
			}

		case "chacha20":
			c, err := chacha20.NewUnauthenticatedCipher(key, nonce)
			if err != nil {
				t.Fatalf("[%s] chacha20.NewUnauthenticatedCipher failed: %v", vec.ID, err)
			}
			gotCT := make([]byte, len(pt))
			c.XORKeyStream(gotCT, pt)
			if !bytesEqual(gotCT, ctExpected) {
				t.Errorf("[%s] ChaCha20 CT mismatch: got %x, want %s", vec.ID, gotCT, vec.CTHex)
			}

		case "poly1305":
			var keyArr [32]byte
			copy(keyArr[:], key)
			var gotMAC [16]byte
			poly1305.Sum(&gotMAC, pt, &keyArr)
			if !bytesEqual(gotMAC[:], macExpected) {
				t.Errorf("[%s] Poly1305 MAC mismatch: got %x, want %s", vec.ID, gotMAC[:], vec.MACHex)
			}

		case "blake2b":
			var h hash.Hash
			var err error
			if len(key) > 0 {
				h, err = blake2b.New512(key)
			} else {
				h, err = blake2b.New512(nil)
			}
			if err != nil {
				t.Fatalf("[%s] blake2b.New512 failed: %v", vec.ID, err)
			}
			h.Write(pt)
			gotMAC := h.Sum(nil)
			if !bytesEqual(gotMAC, macExpected) {
				t.Errorf("[%s] BLAKE2b mismatch: got %x, want %s", vec.ID, gotMAC, vec.MACHex)
			}

		default:
			t.Fatalf("[%s] unknown algorithm: %s", vec.ID, vec.Alg)
		}
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("scanner error: %v", err)
	}

	if count < 10 {
		t.Errorf("expected >= 10 vectors in pack, got %d", count)
	}
}

func bytesEqual(a, b []byte) bool {
	return hmac.Equal(a, b)
}
