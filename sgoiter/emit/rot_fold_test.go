package emit

import (
	"strings"
	"testing"
)

func TestFoldRotateExprs(t *testing.T) {
	cases := []string{
		`*d = (((*d) << uint8(16)) | ((*d) >> uint8(16)))`,
		`*b = (((*b) << uint8(12)) | ((*b) >> uint8(20)))`,
		`v18 = ((v10 << uint8(16)) | (v10 >> uint8(16)))`,
	}
	for _, in := range cases {
		out := foldRotateExprs(in)
		t.Logf("in =%s\nout=%s", in, out)
		if !strings.Contains(out, "bits.RotateLeft") {
			t.Errorf("no fold: %s", out)
		}
		// must be balanced
		if !balanced(out) {
			t.Errorf("unbalanced: %s", out)
		}
	}
}
