package emit

import (
	"strings"
	"testing"
)

func TestFoldTempCopiesOnceWithDefine(t *testing.T) {
	input := "\tv55 := bits.RotateLeft64(v20, 13)\n\tv20 = v55\n"
	got := foldTempCopies(input)
	if strings.Contains(got, "v20 = v55") {
		t.Errorf("expected identity assignment 'v20 = v55' to be folded, got:\n%s", got)
	}
	if !strings.Contains(got, "v20 = bits.RotateLeft64(v20, 13)") {
		t.Errorf("expected folded assignment 'v20 = bits.RotateLeft64(v20, 13)', got:\n%s", got)
	}
}
