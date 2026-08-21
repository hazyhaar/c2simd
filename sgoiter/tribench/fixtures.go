package tribench

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Fixture is one named input vector (shared across backends).
type Fixture struct {
	Name  string
	Data  []byte // primary payload
	Data2 []byte // secondary (xor src2, key, etc.)
	Seed  uint32
}

// FixturesFor returns deterministic vectors for a kind.
func FixturesFor(k Kind) []Fixture {
	hello := []byte("hello")
	abc := []byte("0123456789abcdef")
	abcX := []byte("0123456789abcdefX")
	ones := bytesOf(64, 0xA5)
	kilo := bytesOf(1024, 0x5A)
	// 64 KiB for bench-ish I/O without multi-second runs
	big := bytesOf(64*1024, 0x3C)

	switch k {
	case KindHash64, KindHash64Seed, KindHash32, KindHash32Seed, KindUtf8Proc, KindFastlz1, KindMurmur128, KindLibInj:
		return []Fixture{
			{Name: "empty", Data: nil, Seed: 0},
			{Name: "hello", Data: hello, Seed: 0},
			{Name: "16b", Data: abc, Seed: 0x9747b28c},
			{Name: "17b", Data: abcX, Seed: 0x9747b28c},
			{Name: "64b", Data: ones, Seed: 42},
			{Name: "1k", Data: kilo, Seed: 42},
			{Name: "64k", Data: big, Seed: 42},
		}
	case KindBase64:
		return []Fixture{
			{Name: "pad0", Data: nil},
			{Name: "pad1", Data: []byte("f")},
			{Name: "pad2", Data: []byte("fo")},
			{Name: "pad3", Data: []byte("foo")},
			{Name: "bin_null", Data: []byte{0, 0, 0, 0}},
			{Name: "hello", Data: hello},
			{Name: "16b", Data: abc},
			{Name: "1k", Data: kilo},
		}
	case KindSipHash:
		key := []byte{
			0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
			0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
		}
		return []Fixture{
			{Name: "empty", Data: nil, Data2: key},
			{Name: "hello", Data: hello, Data2: key},
			{Name: "16b", Data: abc, Data2: key},
			{Name: "17b", Data: abcX, Data2: key},
			{Name: "64b", Data: ones, Data2: key},
			{Name: "1k", Data: kilo, Data2: key},
			{Name: "64k", Data: big, Data2: key},
		}
	case KindXor, KindDotF32, KindCjsonCore, KindStbiPng:
		return []Fixture{
			{Name: "empty", Data: nil, Data2: nil},
			{Name: "16b", Data: abc, Data2: bytesOf(16, 0xFF)},
			{Name: "17b", Data: abcX, Data2: bytesOf(17, 0x11)},
			{Name: "64b", Data: ones, Data2: bytesOf(64, 0x5A)},
			{Name: "1k", Data: kilo, Data2: bytesOf(1024, 0xA5)},
			{Name: "64k", Data: big, Data2: bytesOf(64*1024, 0xC3)},
		}
	case KindBlake2b, KindMD5, KindChaChaQR, KindPoly5, KindPolyDonna32, KindCurveDonna64, KindTweetHsalsa:
		return []Fixture{
			{Name: "zero", Data: nil},
			{Name: "pattern", Data: ones},
		}
	case KindTweetVer:
		a := bytesOf(16, 0x00)
		b := bytesOf(16, 0x00)
		c := bytesOf(16, 0x01)
		return []Fixture{
			{Name: "eq", Data: a, Data2: b},
			{Name: "neq", Data: a, Data2: c},
		}
	case KindYyjsonInt:
		return []Fixture{
			{Name: "zero", Seed: 0},
			{Name: "single", Seed: 7},
			{Name: "double", Seed: 42},
			{Name: "medium", Seed: 12345},
			{Name: "large", Seed: 123456789},
			{Name: "max", Seed: 0xFFFFFFFF},
		}
	default:
		return []Fixture{{Name: "empty", Data: nil}}
	}
}

func bytesOf(n int, v byte) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = v
	}
	return b
}

// DigestLine formats one result line (stable protocol).
func DigestLine(name, hexOut string) string {
	return fmt.Sprintf("%s %s", name, hexOut)
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func u64Hex(v uint64) string {
	return fmt.Sprintf("%016x", v)
}

func u32Hex(v uint32) string {
	return fmt.Sprintf("%08x", v)
}
