//go:build goexperiment.simd

package c2simd_test

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd"
	"golang.org/x/crypto/poly1305"
)

func v2Tag(key, msg []byte) [16]byte {
	var tag [16]byte
	c2simd.Poly1305SumV2(&tag, msg, key)
	return tag
}

func scalarTag(key, msg []byte) [16]byte {
	st := c2simd.NewPoly1305QuadChain(key)
	st.Update(msg)
	var tag [16]byte
	st.Finish(&tag)
	return tag
}

func xcryptoTag(key, msg []byte) [16]byte {
	var k [32]byte
	copy(k[:], key)
	var tag [16]byte
	poly1305.Sum(&tag, msg, &k)
	return tag
}

func TestPoly1305V2_Parity(t *testing.T) {
	keys := [][]byte{
		bytes.Repeat([]byte{0x00}, 32),
		bytes.Repeat([]byte{0xff}, 32),
		bytes.Repeat([]byte{0xaa}, 32),
		{
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		},
		{
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x80,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x80,
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		},
	}
	rnd := make([]byte, 32)
	rand.Read(rnd)
	keys = append(keys, rnd)

	sizes := make([]int, 0, 97+8)
	for i := 0; i <= 96; i++ {
		sizes = append(sizes, i)
	}
	sizes = append(sizes, 1024, 8*1024, 64*1024, 1024*1024)
	for _, extra := range []int{64 - 1, 64 + 1, 128 - 3, 128 + 5, 256 - 7, 256 + 11} {
		sizes = append(sizes, extra)
	}

	for ki, key := range keys {
		for _, sz := range sizes {
			msg := make([]byte, sz)
			if sz > 0 {
				for i := range msg {
					msg[i] = byte(i*31 + ki*17)
				}
			}
			got := v2Tag(key, msg)
			sc := scalarTag(key, msg)
			xc := xcryptoTag(key, msg)
			if got != sc {
				t.Fatalf("v2 vs scalaire key=%d sz=%d got %x exp %x", ki, sz, got, sc)
			}
			if got != xc {
				t.Fatalf("v2 vs x/crypto key=%d sz=%d got %x exp %x", ki, sz, got, xc)
			}
		}
	}
}

func TestPoly1305V2_ChunkedUpdate(t *testing.T) {
	key := bytes.Repeat([]byte{0x5a}, 32)
	msg := make([]byte, 1000)
	rand.Read(msg)
	want := v2Tag(key, msg)
	for _, chunk := range []int{1, 15, 16, 17, 63, 64, 65, 200} {
		st := c2simd.NewPoly1305V2(key)
		for off := 0; off < len(msg); off += chunk {
			end := off + chunk
			if end > len(msg) {
				end = len(msg)
			}
			st.Update(msg[off:end])
		}
		var got [16]byte
		st.Finish(&got)
		if got != want {
			t.Fatalf("chunk=%d got %x want %x", chunk, got, want)
		}
	}
}

func TestPoly1305V2_ZeroAlloc(t *testing.T) {
	key := make([]byte, 32)
	msg := make([]byte, 64*1024)
	rand.Read(key)
	rand.Read(msg)
	n := testing.AllocsPerRun(50, func() {
		st := c2simd.NewPoly1305V2(key)
		st.Update(msg)
		var tag [16]byte
		st.Finish(&tag)
	})
	if n != 0 {
		t.Fatalf("allocs = %v want 0", n)
	}
}

func BenchmarkPoly1305V2(b *testing.B) {
	for _, sz := range []int{1024, 8 * 1024, 64 * 1024, 1024 * 1024} {
		b.Run(fmt.Sprintf("%dB", sz), func(b *testing.B) {
			key, payload := spikePayload(sz)
			var mac [16]byte
			b.SetBytes(int64(sz))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				st := c2simd.NewPoly1305V2(key)
				st.Update(payload)
				st.Finish(&mac)
			}
		})
	}
}
