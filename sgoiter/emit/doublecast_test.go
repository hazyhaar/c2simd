package emit

import "testing"

func TestDropDoubleCasts(t *testing.T) {
	cases := []struct{ in, want string }{
		{"v2 = uint32(-int(int(v2 & uint32(1))))", "v2 = uint32(-int(v2 & uint32(1)))"},
		{"x := int(int(int(y)))", "x := int(y)"},
		{"a[int(v9)] = b[int(v9)]", "a[int(v9)] = b[int(v9)]"},
		// different types must stay: the conversion is real
		{"x := int(uint32(y))", "x := int(uint32(y))"},
		// a narrowing cast makes the intermediate int() irrelevant
		{"v2 = -uint32(int(v2 & uint32(1)))", "v2 = -uint32(v2 & uint32(1))"},
		{"b := uint8(int(x))", "b := uint8(x)"},
		// uint64 is not narrower than int: keep the conversion visible
		{"q := uint64(int(x))", "q := uint64(int(x))"},
		// a name that merely ends with a type name is not a cast
		{"x := myint(myint(y))", "x := myint(myint(y))"},
		// the inner cast covers one operand of a larger expression: keep both
		{"return uint64(uint64(v1) << uint8(8) | uint64(x[0]))", "return uint64(uint64(v1) << uint8(8) | uint64(x[0]))"},
		{"v3 = uint64(uint64(a) + b)", "v3 = uint64(uint64(a) + b)"},
	}
	for _, c := range cases {
		if got := dropDoubleCasts(c.in); got != c.want {
			t.Errorf("in  =%q\ngot =%q\nwant=%q", c.in, got, c.want)
		}
	}
}
