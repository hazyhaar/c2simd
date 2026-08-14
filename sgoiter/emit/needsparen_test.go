package emit

import "testing"

func TestNeedsParenCastWithTrailingOp(t *testing.T) {
	// C3: inlining uint32(0xffffff) >> nb must keep parens
	if !needsParen("uint32(0xffffff) >> uint8(nb_mask)") {
		t.Fatal("cast with trailing shift needs parens")
	}
	if needsParen("uint32(0xffffff)") {
		t.Fatal("plain cast is primary")
	}
	if !needsParen("Load24_le(s[29:]) & (uint32(0xffffff) >> uint8(1))") {
		t.Fatal("and-expr needs parens when inlined into shift")
	}
}
