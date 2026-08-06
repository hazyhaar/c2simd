package astmatch_test

import (
	"strings"
	"testing"

	"github.com/hazyhaar/c2simd/internal/astmatch"
)

func TestTransformRotations(t *testing.T) {
	input := `package sample

import "modernc.org/libc"

func chacha20_step(tls *libc.TLS, x uint32) uint32 {
	return rotl32(tls, x, uint32(16))
}
`

	outputBytes, err := astmatch.TransformRotations([]byte(input))
	if err != nil {
		t.Fatalf("TransformRotations a échoué : %v", err)
	}

	output := string(outputBytes)

	if !strings.Contains(output, `bits.RotateLeft32(x, int(16))`) {
		t.Errorf("Sortie ne contient pas bits.RotateLeft32. Obtenu :\n%s", output)
	}

	if !strings.Contains(output, `"math/bits"`) {
		t.Errorf("Import math/bits non ajouté. Obtenu :\n%s", output)
	}
}
