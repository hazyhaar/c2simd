package rules

import (
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/ir"
)

// andAllOnes must not treat 0xffffffff as identity on a 64-bit and
// (low-32 mask used by poly1305). Only -1 / full u64 ones.
func TestAndAllOnesKeepsLow32Mask(t *testing.T) {
	m := &ir.Module{Name: "t", Funcs: []ir.Func{{
		Name: "f", NVals: 3,
		Body: []ir.Instr{
			{Op: ir.OpConst, Dst: 0, Imm: 0x1_0000_0001},
			{Op: ir.OpConst, Dst: 1, Imm: 0xffffffff},
			{Op: ir.OpAnd, Dst: 2, Args: []ir.Value{0, 1}},
		},
	}}}
	out, err := andAllOnes(m)
	if err != nil {
		t.Fatal(err)
	}
	ins := out.Funcs[0].Body[2]
	if ins.Op != ir.OpAnd {
		t.Fatalf("expected And preserved, got %v (would break poly1305 MAC)", ins.Op)
	}
}

func TestAndAllOnesFoldsTrueOnes(t *testing.T) {
	m := &ir.Module{Name: "t", Funcs: []ir.Func{{
		Name: "f", NVals: 3,
		Body: []ir.Instr{
			{Op: ir.OpConst, Dst: 0, Imm: 42},
			{Op: ir.OpConst, Dst: 1, Imm: -1},
			{Op: ir.OpAnd, Dst: 2, Args: []ir.Value{0, 1}},
		},
	}}}
	out, err := andAllOnes(m)
	if err != nil {
		t.Fatal(err)
	}
	ins := out.Funcs[0].Body[2]
	if ins.Op != ir.OpMov || ins.Args[0] != 0 {
		t.Fatalf("expected Mov v0, got %+v", ins)
	}
}
