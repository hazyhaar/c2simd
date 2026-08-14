package emit

import (
	"strings"
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/ir"
)

// rotModule builds `return (x <shiftA> r) | (x <shiftB> (w - r))` over a typed
// parameter, the shape murmur3's rotl32 and tweetnacl's R take.
func rotModule(typ ir.TypeName, w int64, first, second ir.Op) *ir.Module {
	const (
		x   = ir.Value(0)
		r   = ir.Value(1)
		cw  = ir.Value(2)
		sub = ir.Value(3)
		s1  = ir.Value(4)
		s2  = ir.Value(5)
		or  = ir.Value(6)
	)
	return &ir.Module{Name: "rot", Funcs: []ir.Func{{
		Name:   "rot",
		Result: typ,
		NVals:  7,
		Params: []ir.Param{
			{Name: "x", Type: typ, Reg: x},
			{Name: "r", Type: ir.TypUint8, Reg: r},
		},
		Body: []ir.Instr{
			{Op: ir.OpConst, Dst: cw, Imm: w},
			{Op: ir.OpSub, Dst: sub, Args: []ir.Value{cw, r}},
			{Op: first, Dst: s1, Args: []ir.Value{x, r}},
			{Op: second, Dst: s2, Args: []ir.Value{x, sub}},
			{Op: ir.OpOr, Dst: or, Args: []ir.Value{s1, s2}},
			{Op: ir.OpReturn, Args: []ir.Value{or}},
		},
	}}}
}

func TestRotateWithVariableCount(t *testing.T) {
	cases := []struct {
		name  string
		typ   ir.TypeName
		w     int64
		first ir.Op
		want  string
	}{
		{"left32", ir.TypUint32, 32, ir.OpShl, "bits.RotateLeft32(x, int(r))"},
		{"left64", ir.TypUint64, 64, ir.OpShl, "bits.RotateLeft64(x, int(r))"},
		{"right64", ir.TypUint64, 64, ir.OpShr, "bits.RotateLeft64(x, 64-int(r))"},
	}
	for _, c := range cases {
		second := ir.OpShr
		if c.first == ir.OpShr {
			second = ir.OpShl
		}
		src, err := Emit(rotModule(c.typ, c.w, c.first, second), ProfileGo127)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if !strings.Contains(src, c.want) {
			t.Errorf("%s: want %q in\n%s", c.name, c.want, src)
		}
	}
}

// The width must come from the shifted value's type. Folding a 32-wide rotation
// over a 64-bit value would silently drop the upper half.
func TestVariableRotateRefusesMismatchedWidth(t *testing.T) {
	src, err := Emit(rotModule(ir.TypUint64, 32, ir.OpShl, ir.OpShr), ProfileGo127)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(src, "RotateLeft") {
		t.Errorf("folded a 32-count rotation over a uint64 value:\n%s", src)
	}
}
