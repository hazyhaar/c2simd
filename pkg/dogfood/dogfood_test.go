package dogfood_test

import (
	"bytes"
	"testing"

	c2tuidiff "code.hazyhaar.fr/devhoros/c2simd/pkg/c2tuidiff"
	"code.hazyhaar.fr/devhoros/c2simd/pkg/dogfood/checksum"
	"code.hazyhaar.fr/devhoros/c2simd/pkg/dogfood/fastxor"
	"code.hazyhaar.fr/devhoros/c2simd/pkg/dogfood/libinjection"
	"code.hazyhaar.fr/devhoros/c2simd/pkg/dogfood/tuidiff"
)

func TestDogfoodPackages(t *testing.T) {
	// 1. LibInjection / Strlenspn test
	hello := []byte("hello")
	n := libinjection.Strlenspn(hello, uint64(len(hello)), []byte{'h', 'e', 'l', 0})
	if n != 4 {
		t.Errorf("expected Strlenspn=4 for 'hello', got %d", n)
	}

	world := []byte("world")
	n2 := libinjection.Strlenspn(world, uint64(len(world)), []byte{'h', 'e', 'l', 0})
	if n2 != 0 {
		t.Errorf("expected Strlenspn=0 for 'world', got %d", n2)
	}

	// 2. FastXOR test
	s1 := []byte("hello world 1234")
	s2 := []byte("1234567890abcdef")
	dst := make([]byte, len(s1))
	fastxor.Bytes(dst, s1, s2)

	// XOR twice restores original
	dst2 := make([]byte, len(s1))
	fastxor.Bytes(dst2, dst, s2)
	if !bytes.Equal(dst2, s1) {
		t.Errorf("fastxor mismatch: got %s, want %s", dst2, s1)
	}

	// 3. Checksum FNV1a & CRC32 test
	data := []byte("testing dogfooding checksums")
	h1 := checksum.FNV1a64(data)
	if h1 == 0 {
		t.Errorf("expected non-zero FNV1a64 hash")
	}

	c1 := checksum.CRC32IEEE(data)
	if c1 == 0 {
		t.Errorf("expected non-zero CRC32IEEE checksum")
	}

	// 4. TUIDiff test (Dogfooding sgoiter SIMD diffing)
	front := make([]c2tuidiff.Cell, 80*24)
	back := make([]c2tuidiff.Cell, 80*24)
	back[10] = c2tuidiff.Cell{Rune: 'X', Fg: 1, Bg: 2, Flags: 3}

	diffWrap := tuidiff.NewDiffWrapper(16)
	diffCount := diffWrap.Diff(front, back, 80, 24, 80)
	if diffCount != 1 {
		t.Errorf("expected 1 changed cell, got %d", diffCount)
	}
	if len(diffWrap.Spans) != 1 || diffWrap.Spans[0].X != 10 || diffWrap.Spans[0].Length != 1 {
		t.Errorf("unexpected spans: %+v", diffWrap.Spans)
	}
}
