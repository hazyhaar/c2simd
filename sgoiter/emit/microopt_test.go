package emit

import "testing"

func TestDropRedundantAlignMask(t *testing.T) {
	in := "PutUint64(dst[int(v4 &^ uint64(7)):], x)"
	out := dropRedundantAlignMask(in)
	if out != "PutUint64(dst[int(v4):], x)" {
		t.Fatalf("%q", out)
	}
}

func TestBCEIndexLoopsVn(t *testing.T) {
	in := "\tvar v3 int\n\tvar v4 uint64\n\tv3 = 0\n\tfor v3 < n {\n\t\tv4 = v4 | uint64(x[v3] ^ y[v3])\n\t\tv3 = v3 + 1\n\t}\n"
	out := bceIndexLoops(in)
	if !containsAll(out, "_x := x[:n]", "_y := y[:n]", "v3++") {
		t.Fatalf("bce failed:\n%s", out)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !contains(s, p) {
			return false
		}
	}
	return true
}
func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
