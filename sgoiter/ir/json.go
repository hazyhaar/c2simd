package ir

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

// Marshal serializes a Module to stable JSON (indent, sorted by encoding/json field order).
func Marshal(m *Module) ([]byte, error) {
	if m == nil {
		return nil, fmt.Errorf("ir: nil module")
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(m); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Unmarshal parses a Module from JSON.
func Unmarshal(data []byte) (*Module, error) {
	var m Module
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("ir: unmarshal: %w", err)
	}
	return &m, nil
}

// Load reads a Module from path.
func Load(path string) (*Module, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Unmarshal(b)
}

// EqualJSON reports whether two modules serialize identically.
func EqualJSON(a, b *Module) (bool, error) {
	ba, err := Marshal(a)
	if err != nil {
		return false, err
	}
	bb, err := Marshal(b)
	if err != nil {
		return false, err
	}
	return bytes.Equal(ba, bb), nil
}
