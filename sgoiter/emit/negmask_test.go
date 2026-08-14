package emit

import "testing"

func TestFoldNegatedMask(t *testing.T) {
	cases := []struct{ in, want string }{
		{
			"v2 = (v2 >> uint8(1)) ^ uint32(0xedb88320) & uint32(-int(v2 & uint32(1)))",
			"v2 = (v2 >> uint8(1)) ^ uint32(0xedb88320) & -(v2 & uint32(1))",
		},
		{"m := uint64(-int(x ^ uint64(1)))", "m := -(x ^ uint64(1))"},
		// a negated counter is not a mask: murmur3 walks from -nblocks
		{"v18 = uint32(-int(v7))", "v18 = uint32(-int(v7))"},
		{"v18 = uint32(-int(v7 + v8))", "v18 = uint32(-int(v7 + v8))"},
		// no evidence the inner expression is already 32 bits wide
		{"v9 = uint32(-int(a & b))", "v9 = uint32(-int(a & b))"},
		// width mismatch between the conversions: leave it
		{"v9 = uint32(-int(a & uint64(1)))", "v9 = uint32(-int(a & uint64(1)))"},
	}
	for _, c := range cases {
		if got := foldNegatedMask(c.in); got != c.want {
			t.Errorf("in  =%q\ngot =%q\nwant=%q", c.in, got, c.want)
		}
	}
}
