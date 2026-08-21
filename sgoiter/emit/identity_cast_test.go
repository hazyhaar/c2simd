package emit

import (
	"strings"
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/ir"
)

// T1 — a conversion to the type a value already has says nothing.
func TestDropIdentityCasts(t *testing.T) {
	params := map[string]string{"len_": "uint64"}
	cases := []struct{ in, ret, want string }{
		{"\tvar v2 uint64\n\treturn uint64(v2)", "uint64", "\tvar v2 uint64\n\treturn v2"},
		{"\tvar v2 uint32\n\treturn uint32(^v2)", "uint32", "\tvar v2 uint32\n\treturn ^v2"},
		{"\tvar v4 uint64\n\tv4 = v4 + uint64(1)", "uint64", "\tvar v4 uint64\n\tv4 = v4 + 1"},
		{"\tvar v2 uint64\n\tv2 = uint64(0xcbf29ce484222325)", "uint64", "\tvar v2 uint64\n\tv2 = 0xcbf29ce484222325"},
		{"\tvar v5 uint32\n\tfor v5 < uint32(8) {", "uint32", "\tvar v5 uint32\n\tfor v5 < 8 {"},
		// a real conversion stays: data[i] is a byte
		{"\tvar v2 uint64\n\tv2 = v2 ^ uint64(data[int(v4)])", "uint64", "\tvar v2 uint64\n\tv2 = v2 ^ uint64(data[int(v4)])"},
		// returning a different type keeps its conversion
		{"\tvar v9 uint32\n\treturn uint64(v9)", "uint64", "\tvar v9 uint32\n\treturn uint64(v9)"},
		// a short declaration needs its type: x := 1 would be an int
		{"\tv7 := uint64(3)", "uint64", "\tv7 := uint64(3)"},
		// a parameter's own type counts as declared
		{"\treturn uint64(len_)", "uint64", "\treturn len_"},
	}
	for _, c := range cases {
		if got := dropIdentityCasts(c.in, c.ret, params); got != c.want {
			t.Errorf("in  =%q\ngot =%q\nwant=%q", c.in, got, c.want)
		}
	}
}

// T2 — Go takes any integer type as a shift count.
func TestBareShiftCounts(t *testing.T) {
	cases := []struct{ in, want string }{
		{"v2 = (v2 >> uint8(1)) ^ x", "v2 = (v2 >> 1) ^ x"},
		{"h = h << uint8(16)", "h = h << 16"},
		{"v = v >> uint64(32)", "v = v >> 32"},
		// a variable count is not a literal
		{"v = v << uint8(r)", "v = v << uint8(r)"},
		// the shifted value keeps its own conversion
		{"v = uint64(x) << 8", "v = uint64(x) << 8"},
	}
	for _, c := range cases {
		if got := bareShiftCounts(c.in); got != c.want {
			t.Errorf("in  =%q\ngot =%q\nwant=%q", c.in, got, c.want)
		}
	}
}

// T6 — a printable C string table is a constant, and a numeric table has a
// known extent. Emitted for real, not asserted against a string built here.
func TestRodataTablesAreNotSlices(t *testing.T) {
	m := &ir.Module{
		Name: "rodata",
		Globals: []ir.Global{
			{Name: "b64_table", Data: "ABCDEF"},
			{Name: "blake2b_sigma", Type: ir.TypUint8, InitCSV: "0,1,2,3"},
			{Name: "K", Type: ir.TypUint64, InitCSV: "0x428a2f98d728ae22,0x7137449123ef65cd"},
		},
		Funcs: []ir.Func{{
			Name: "f", Result: ir.TypUint64, NVals: 1,
			Body: []ir.Instr{
				{Op: ir.OpConst, Dst: 0, Imm: 0},
				{Op: ir.OpReturn, Args: []ir.Value{0}},
			},
		}},
	}
	src, err := Emit(m, ProfileGo127)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`const B64_table = "ABCDEF"`,
		"var Blake2b_sigma = [4]byte{0, 1, 2, 3}",
		"var K = [2]uint64{0x428a2f98d728ae22, 0x7137449123ef65cd}",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("missing %q in:\n%s", want, src)
		}
	}
	if strings.Contains(src, "[]byte(") || strings.Contains(src, "= []uint64{") {
		t.Errorf("a table is still a slice:\n%s", src)
	}
}
