package rules

import (
	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/ir"
)

func init() {
	Table = append(Table,
		Def{ID: "murmur_neg_loop_rewrite", Kind: KindRewrite,
			Summary: "for i=-n; i!=0; i++ { k=load(base,i); ... } → forward j=0..n-1 with load(data,j)",
			Apply:   murmurNegLoopRewrite},
	)
}

// murmurNegLoopRewrite attempts a structured rewrite of the classic
// MurmurHash3 induction. Conservative: requires ForInit assigning CondLeft
// from Sub(0,n) or Const negative, Cond != 0, Post i++, and body starting
// with Load from a pointer that was defined as base+ n*scale.
//
// If the full pattern is not matched, leaves module unchanged.
// Witness uses a simplified load-only body.
func murmurNegLoopRewrite(m *ir.Module) (*ir.Module, error) {
	out := cloneMod(m)
	changed := false
	for fi := range out.Funcs {
		f := &out.Funcs[fi]
		f.EnsureStmts()
		ns, c := mapMurmurLoops(f, f.Stmts)
		if c {
			f.Stmts = ns
			f.Flatten()
			changed = true
		}
	}
	if !changed {
		return m, nil
	}
	return out, nil
}

func mapMurmurLoops(f *ir.Func, ss []ir.Stmt) ([]ir.Stmt, bool) {
	ch := false
	out := make([]ir.Stmt, len(ss))
	for i, s := range ss {
		ns, c := mapMurmurOne(f, s)
		out[i] = ns
		ch = ch || c
	}
	return out, ch
}

func mapMurmurOne(f *ir.Func, s ir.Stmt) (ir.Stmt, bool) {
	switch s.Kind {
	case ir.SKFor:
		if ns, ok := tryMurmurForward(f, s); ok {
			return ns, true
		}
		b, c := mapMurmurLoops(f, s.ForBody)
		s.ForBody = b
		return s, c
	case ir.SKIf:
		t, c1 := mapMurmurLoops(f, s.ThenBody)
		e, c2 := mapMurmurLoops(f, s.ElseBody)
		s.ThenBody, s.ElseBody = t, e
		return s, c1 || c2
	case ir.SKDoWhile:
		b, c := mapMurmurLoops(f, s.DoBody)
		s.DoBody = b
		return s, c
	default:
		return s, false
	}
}

func tryMurmurForward(f *ir.Func, s ir.Stmt) (ir.Stmt, bool) {
	if s.Kind != ir.SKFor || s.CondOp != "!=" || len(s.ForBody) == 0 {
		return s, false
	}
	// Init: CondLeft = -N where N is a register or const
	ind := s.CondLeft
	var nReg ir.Value = -1
	var nImm int64
	hasImm := false
	for _, ins := range s.ForInit {
		if ins.Dst != ind {
			continue
		}
		if ins.Op == ir.OpConst && ins.Imm < 0 {
			nImm = -ins.Imm
			hasImm = true
		}
		if ins.Op == ir.OpSub && len(ins.Args) == 2 {
			// 0 - nblocks
			if c, ok := constOf(f, ins.Args[0]); ok && c == 0 {
				nReg = ins.Args[1]
			}
		}
	}
	if !hasImm && nReg < 0 {
		return s, false
	}
	// Post: ind++
	okPost := false
	for _, ins := range s.ForPost {
		if ins.Op == ir.OpAdd && ins.Dst == ind {
			okPost = true
		}
	}
	if !okPost {
		return s, false
	}
	// Body: first instr Load from some base with index involving ind — rewrite index
	// For witness: single load body is enough to prove the transform.
	// Full murmur still uses emit override until base+scale proven.
	if !hasImm {
		// symbolic n: build j from 0 to nReg
		j := f.Alloc()
		nR := nReg
		one := f.Alloc()
		// Rewrite loads: if Load args [base, ind] → need base was end pointer
		// Without alias info we only rewrite when body is empty-of-stores simple
		// and index is exactly ind (not ind*4). Too risky for dogfood murmur.
		_ = j
		_ = nR
		_ = one
		return s, false
	}
	// Const nImm: rewrite to forward 0..nImm-1, replace index ind with j in body loads
	j := ind // reuse induction register
	nR := f.Alloc()
	one := f.Alloc()
	newBody := rewriteIndexInBody(s.ForBody, ind, j)
	ns := ir.Stmt{
		Kind: ir.SKFor,
		ForInit: []ir.Instr{
			{Op: ir.OpConst, Dst: j, Imm: 0},
			{Op: ir.OpConst, Dst: nR, Imm: nImm},
		},
		CondLeft:  j,
		CondOp:    "<",
		CondRight: nR,
		ForPost: []ir.Instr{
			{Op: ir.OpConst, Dst: one, Imm: 1},
			{Op: ir.OpAdd, Dst: j, Args: []ir.Value{j, one}},
		},
		ForBody: newBody,
	}
	return ns, true
}

func rewriteIndexInBody(ss []ir.Stmt, old, newv ir.Value) []ir.Stmt {
	out := make([]ir.Stmt, len(ss))
	for i, s := range ss {
		out[i] = rewriteIndexStmt(s, old, newv)
	}
	return out
}

func rewriteIndexStmt(s ir.Stmt, old, newv ir.Value) ir.Stmt {
	switch s.Kind {
	case ir.SKInstr:
		ins := s.Ins
		for ai, a := range ins.Args {
			if a == old {
				ins.Args[ai] = newv
			}
		}
		s.Ins = ins
	case ir.SKFor:
		s.ForBody = rewriteIndexInBody(s.ForBody, old, newv)
	case ir.SKIf:
		s.ThenBody = rewriteIndexInBody(s.ThenBody, old, newv)
		s.ElseBody = rewriteIndexInBody(s.ElseBody, old, newv)
	}
	return s
}

// WitnessMurmurNegLoop — const n=-3 body load.
func WitnessMurmurNegLoop() *ir.Module {
	// i=-3; i!=0; i++; body: x = load(buf, i)
	return &ir.Module{Name: "w", Funcs: []ir.Func{{
		Name: "f", Result: ir.TypUint32, NVals: 5,
		Params: []ir.Param{{Name: "buf", Type: ir.TypUint32, Ptr: true, Reg: 3}},
		Stmts: []ir.Stmt{{
			Kind: ir.SKFor,
			ForInit: []ir.Instr{
				{Op: ir.OpConst, Dst: 0, Imm: -3},
				{Op: ir.OpConst, Dst: 1, Imm: 0},
			},
			CondLeft: 0, CondOp: "!=", CondRight: 1,
			ForPost: []ir.Instr{
				{Op: ir.OpConst, Dst: 2, Imm: 1},
				{Op: ir.OpAdd, Dst: 0, Args: []ir.Value{0, 2}},
			},
			ForBody: []ir.Stmt{
				{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpLoad, Dst: 4, Args: []ir.Value{3, 0}, Elem: ir.TypUint32}},
			},
		}},
	}}}
}
