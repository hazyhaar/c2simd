package emit

import "testing"

func TestDropRedundantParens(t *testing.T) {
	cases := []struct{ in, want string }{
		{"\tv8[8] = (v8[8] + v8[12])", "\tv8[8] = v8[8] + v8[12]"},
		{"\tif !(((v3) < (len_))) { break }", "\tif !(v3 < len_) { break }"},
		{"\t*d = ((*d) ^ v7)", "\t*d = *d ^ v7"},
		{"\tif ((((s[0]) == (uint8(0x6e)))) && ((s[1]) == (uint8(0x75)))) {", "\tif (s[0] == uint8(0x6e)) && (s[1] == uint8(0x75)) {"},
		// grouping that carries meaning must survive
		{"\tv5 = (v5 * uint64(10)) + uint64(v15 - v17)", "\tv5 = (v5 * uint64(10)) + uint64(v15 - v17)"},
		{"\tv1 := (a + b) * c", "\tv1 := (a + b) * c"},
		{"\tv2 := !(a < b)", "\tv2 := !(a < b)"},
		// a call argument must keep its parens: int(v9) is not (v9)
		{"\tv7[int(v9)] = h[int(v9)]", "\tv7[int(v9)] = h[int(v9)]"},
		{"\tv18 := bits.RotateLeft64((v8[12] ^ v8[0]), 32)", "\tv18 := bits.RotateLeft64((v8[12] ^ v8[0]), 32)"},
		// a wrapped call must not lose the parens that belong to the call
		{"\tv1 := (binary.LittleEndian.Uint64(block))", "\tv1 := binary.LittleEndian.Uint64(block)"},
		{"\tv5 = uint64((v15 - v17))", "\tv5 = uint64(v15 - v17)"},
		// multiple arguments: each pair is not alone inside the call
		{"\tv6 = bits.RotateLeft64((a ^ b), 32)", "\tv6 = bits.RotateLeft64((a ^ b), 32)"},
	}
	for _, c := range cases {
		if got := dropRedundantParens(c.in); got != c.want {
			t.Errorf("in  =%q\ngot =%q\nwant=%q", c.in, got, c.want)
		}
	}
}

func TestStripOuterParensKeepsBalance(t *testing.T) {
	// (a) + (b) is fully parenthesized at both ends but not wrapped as a whole
	if got := stripOuterParens("(a) + (b)"); got != "(a) + (b)" {
		t.Errorf("over-stripped: %q", got)
	}
	if got := stripOuterParens("((a + b))"); got != "a + b" {
		t.Errorf("under-stripped: %q", got)
	}
	if got := stripOuterParens("f(x)"); got != "f(x)" {
		t.Errorf("call damaged: %q", got)
	}
}
