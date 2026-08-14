package emit

import "testing"

func TestCharLiteralsInCmp(t *testing.T) {
	cases := []struct{ in, want string }{
		{"\tif !(v15 <= uint8(57)) { break }", "\tif !(v15 <= '9') { break }"},
		{"\tif s[0] == uint8(0x6e) {", "\tif s[0] == 'n' {"},
		{"\tif d == uint8(61) {", "\tif d == '=' {"},
		// shift widths keep their numeric form even on a line that compares
		{"\tif (v1 >> uint8(56)) == uint8(48) {", "\tif (v1 >> uint8(56)) == '0' {"},
		// sizes and masks are not characters
		{"\tif v2 == uint8(255) {", "\tif v2 == uint8(255) {"},
		{"\tif v3 < uint8(32) {", "\tif v3 < uint8(32) {"},
		// no comparison anywhere: leave it alone
		{"\tv17 := uint8(48)", "\tv17 := uint8(48)"},
		// base64 padding: stored, never compared
		{"\t\tdst[int(v95)] = uint8(61)", "\t\tdst[int(v95)] = '='"},
		{"\t\tdst[0] = uint8(65)", "\t\tdst[0] = 'A'"},
		// a stored size is not a character
		{"\t\tbuf[0] = uint8(255)", "\t\tbuf[0] = uint8(255)"},
		{"\t\tbuf[0] = uint8(32)", "\t\tbuf[0] = uint8(32)"},
	}
	for _, c := range cases {
		if got := charLiteralsForBytes(c.in); got != c.want {
			t.Errorf("in  =%q\ngot =%q\nwant=%q", c.in, got, c.want)
		}
	}
}

func TestCharLiteralFollowsHoistedTemp(t *testing.T) {
	in := "\tv17 := uint8(48)\n\tif !(v15 >= v17) { break }\n\tv5 = uint64(v15 - v17)\n"
	want := "\tv17 := uint8('0')\n\tif !(v15 >= v17) { break }\n\tv5 = uint64(v15 - v17)\n"
	if got := charLiteralsForBytes(in); got != want {
		t.Errorf("got =%q\nwant=%q", got, want)
	}
}

func TestCharLiteralIgnoresTempNeverCompared(t *testing.T) {
	// a size bound to a name and only added must stay numeric
	in := "\tv20 := uint8(65)\n\tv21 = v20 + uint8(1)\n"
	if got := charLiteralsForBytes(in); got != in {
		t.Errorf("rewrote a temp that no comparison reads: %q", got)
	}
}
