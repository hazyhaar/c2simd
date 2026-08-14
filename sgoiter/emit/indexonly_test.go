package emit

import (
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/ir"
)

// counterLoop builds `for i < bound { t[i] = 0; i++ }` where bound is either a
// constant or the function's parameter.
func counterLoop(boundIsParam bool) *ir.Module {
	const (
		p     = ir.Value(0) // parameter
		t     = ir.Value(1) // table
		i     = ir.Value(2)
		zero  = ir.Value(3)
		cst   = ir.Value(4)
		one   = ir.Value(5)
		nextI = ir.Value(6)
	)
	bound := cst
	if boundIsParam {
		bound = p
	}
	return &ir.Module{Name: "m", Funcs: []ir.Func{{
		Name:   "f",
		Result: ir.TypVoid,
		NVals:  7,
		Params: []ir.Param{
			{Name: "n", Type: ir.TypUint64, Reg: p},
			{Name: "t", Type: ir.TypUint64, Reg: t, Ptr: true},
		},
		Stmts: []ir.Stmt{
			{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpConst, Dst: zero, Imm: 0}},
			{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpConst, Dst: cst, Imm: 16}},
			{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpMov, Dst: i, Args: []ir.Value{zero}}},
			{
				Kind:      ir.SKFor,
				CondLeft:  i,
				CondOp:    "<",
				CondRight: bound,
				ForBody: []ir.Stmt{
					{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpStore, Args: []ir.Value{t, i, zero}}},
				},
				ForPost: []ir.Instr{
					{Op: ir.OpConst, Dst: one, Imm: 1},
					{Op: ir.OpAdd, Dst: nextI, Args: []ir.Value{i, one}},
					{Op: ir.OpMov, Dst: i, Args: []ir.Value{nextI}},
				},
			},
			{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpReturn}},
		},
	}}}
}

// A counter bounded by a constant only ever walks the table: it can be an int.
func TestIndexOnlyAcceptsConstantBoundedCounter(t *testing.T) {
	m := counterLoop(false)
	f := &m.Funcs[0]
	f.EnsureStmts()
	if got := indexOnlyRegs(f); !got[2] {
		t.Errorf("constant-bounded counter not recognised: %v", got)
	}
}

// A counter compared against a uint64 parameter keeps its own width; retyping it
// would only move the conversion to the comparison.
func TestIndexOnlyRejectsParameterBoundedCounter(t *testing.T) {
	m := counterLoop(true)
	f := &m.Funcs[0]
	f.EnsureStmts()
	if got := indexOnlyRegs(f); got[2] {
		t.Errorf("counter bounded by a parameter was retyped: %v", got)
	}
}

// A value that reaches a bitwise operator carries meaning in its width.
func TestIndexOnlyRejectsBitwiseUse(t *testing.T) {
	m := counterLoop(false)
	f := &m.Funcs[0]
	f.EnsureStmts()
	f.Stmts[3].ForBody = append(f.Stmts[3].ForBody, ir.Stmt{
		Kind: ir.SKInstr,
		Ins:  ir.Instr{Op: ir.OpXor, Dst: 6, Args: []ir.Value{2, 3}},
	})
	if got := indexOnlyRegs(f); got[2] {
		t.Errorf("a value used in a xor was retyped: %v", got)
	}
}

func TestDropSelfCasts(t *testing.T) {
	types := map[string]string{"v9": "int", "v59": "int", "h": "uint64"}
	cases := []struct{ in, want string }{
		{"v7[int(v9)] = 1", "v7[v9] = 1"},
		{"x = blake2b_sigma[int(v59 + 1)]", "x = blake2b_sigma[(v59 + 1)]"},
		// a real conversion stays
		{"x = uint64(v9)", "x = uint64(v9)"},
		{"x = int(h)", "x = int(h)"},
		// unknown name: no evidence of its type
		{"x = int(zz)", "x = int(zz)"},
	}
	for _, c := range cases {
		if got := dropSelfCasts(c.in, types); got != c.want {
			t.Errorf("in  =%q\ngot =%q\nwant=%q", c.in, got, c.want)
		}
	}
}
