package sgoiterbench

import (
	"bytes"
)

// Fixture represents a test payload.
type Fixture struct {
	Name string
	Data []byte
}

// DefaultFixtures returns standard test payloads ranging from empty up to 64KB, plus 1M, 10M, 100M for heavy testing.
func DefaultFixtures(heavy bool) []Fixture {
	fix := []Fixture{
		{Name: "empty", Data: []byte("")},
		{Name: "hello", Data: []byte("hello sgoiter benchmark")},
		{Name: "16b", Data: bytes.Repeat([]byte("0123456789abcdef"), 1)},
		{Name: "17b", Data: bytes.Repeat([]byte("0123456789abcdefX"), 1)},
		{Name: "64b", Data: bytes.Repeat([]byte("a"), 64)},
		{Name: "1k", Data: bytes.Repeat([]byte("Z"), 1024)},
		{Name: "64k", Data: bytes.Repeat([]byte("P"), 64*1024)},
	}
	if heavy {
		fix = append(fix,
			Fixture{Name: "1M", Data: bytes.Repeat([]byte("H"), 1*1024*1024)},
			Fixture{Name: "10M", Data: bytes.Repeat([]byte("X"), 10*1024*1024)},
			Fixture{Name: "100M", Data: bytes.Repeat([]byte("Z"), 100*1024*1024)},
		)
	}
	return fix
}

// GenerateHorosvec creates a 64-bit float/int vector payload of given size for heavy load.
func GenerateHorosvec(elements int) []byte {
	buf := make([]byte, elements*8)
	for i := 0; i < elements*8; i++ {
		buf[i] = byte((i * 37 + 13) & 0xFF)
	}
	return buf
}
