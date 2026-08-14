package emit

import (
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/ir"
)

func TestAllocaEscapes_IndexOnly(t *testing.T) {
	f := &ir.Func{
		Name: "f", NVals: 4,
		Stmts: []ir.Stmt{
			{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpAlloca, Dst: 0, Imm: 16, Elem: ir.TypUint32}},
			{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpConst, Dst: 1, Imm: 0}},
			{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpLoad, Dst: 2, Args: []ir.Value{0, 1}, Elem: ir.TypUint32}},
			{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpReturn, Args: []ir.Value{2}}},
		},
	}
	if allocaEscapes(f, 0) {
		t.Fatal("index-only alloca should not escape")
	}
}

func TestAllocaEscapes_CallArg(t *testing.T) {
	f := &ir.Func{
		Name: "f", NVals: 3,
		Stmts: []ir.Stmt{
			{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpAlloca, Dst: 0, Imm: 16, Elem: ir.TypUint32}},
			{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpCall, Dst: 1, Sym: "foo", Args: []ir.Value{0}}},
		},
	}
	if !allocaEscapes(f, 0) {
		t.Fatal("call arg must escape")
	}
}
