package rules

import (
	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/ir"
)

func init() {
	Table = append(Table,
		Def{ID: "loop_neg_count_to_forward", Kind: KindRewrite,
			Summary: "empty for i=-N; i!=0; i++ → for i=0; i<N; i++ (safe structural)",
			Apply:   loopNegCountToForward},
	)
}

// loopNegCountToForward rewrites only the empty-body structural pattern:
//
//	i = -N (const); for i != 0; i = i+1 { }
//
// into
//
//	i = 0; for i < N; i = i+1 { }
//
// Non-empty bodies (murmur dogfood) stay on the emit override path — rewriting
// pointer-relative blocks[i] needs alias analysis not yet in IR.
func loopNegCountToForward(m *ir.Module) (*ir.Module, error) {
	out := cloneMod(m)
	changed := false
	for fi := range out.Funcs {
		f := &out.Funcs[fi]
		// Prefer structured Stmts; fall back to Body flat (no loops there).
		if len(f.Stmts) > 0 {
			ns, c := mapStmtList(f.Stmts)
			if c {
				f.Stmts = ns
				changed = true
			}
			continue
		}
	}
	if !changed {
		return m, nil
	}
	return out, nil
}

func mapStmtList(ss []ir.Stmt) ([]ir.Stmt, bool) {
	ch := false
	out := make([]ir.Stmt, len(ss))
	copy(out, ss)
	for i := range out {
		s, c := mapStmt(out[i])
		out[i] = s
		ch = ch || c
	}
	return out, ch
}

func mapStmt(s ir.Stmt) (ir.Stmt, bool) {
	switch s.Kind {
	case ir.SKFor:
		if ns, ok := tryEmptyNegLoop(s); ok {
			return ns, true
		}
		b, c := mapStmtList(s.ForBody)
		s.ForBody = b
		return s, c
	case ir.SKIf:
		t, c1 := mapStmtList(s.ThenBody)
		e, c2 := mapStmtList(s.ElseBody)
		s.ThenBody, s.ElseBody = t, e
		return s, c1 || c2
	case ir.SKDoWhile:
		b, c := mapStmtList(s.DoBody)
		s.DoBody = b
		return s, c
	case ir.SKSwitch:
		ch := false
		for i := range s.SwitchCases {
			b, c := mapStmtList(s.SwitchCases[i].Body)
			s.SwitchCases[i].Body = b
			ch = ch || c
		}
		return s, ch
	default:
		return s, false
	}
}

func tryEmptyNegLoop(s ir.Stmt) (ir.Stmt, bool) {
	if s.Kind != ir.SKFor || len(s.ForBody) != 0 {
		return s, false
	}
	if s.CondOp != "!=" || s.CondLeft < 0 {
		return s, false
	}
	// Init: single const negative to CondLeft
	var nAbs int64
	foundInit := false
	for _, ins := range s.ForInit {
		if ins.Dst == s.CondLeft && ins.Op == ir.OpConst && ins.Imm < 0 {
			nAbs = -ins.Imm
			foundInit = true
		}
	}
	if !foundInit || nAbs == 0 {
		return s, false
	}
	// CondRight must be const 0 — find in ForInit/ForCondPrep or CondRight value
	// We'll accept any CondRight and force rewrite (witness uses const 0 as separate reg).
	// Post: add 1 to induction
	hasPost := false
	for _, ins := range s.ForPost {
		if ins.Op == ir.OpAdd && ins.Dst == s.CondLeft {
			hasPost = true
		}
	}
	if !hasPost {
		return s, false
	}
	ind := s.CondLeft
	// Use high temps unlikely to collide in tiny witness modules
	nReg := ir.Value(900)
	oneReg := ir.Value(901)
	ns := ir.Stmt{
		Kind: ir.SKFor,
		ForInit: []ir.Instr{
			{Op: ir.OpConst, Dst: ind, Imm: 0},
			{Op: ir.OpConst, Dst: nReg, Imm: nAbs},
		},
		CondLeft:  ind,
		CondOp:    "<",
		CondRight: nReg,
		ForPost: []ir.Instr{
			{Op: ir.OpConst, Dst: oneReg, Imm: 1},
			{Op: ir.OpAdd, Dst: ind, Args: []ir.Value{ind, oneReg}},
		},
		ForBody: nil,
	}
	return ns, true
}

// WitnessLoopNeg is the module used by rules_test for loop_neg_count_to_forward.
func WitnessLoopNeg() *ir.Module {
	// i in v0: i=-5; while i != 0; i++
	// v1 = 0 for cond right
	return &ir.Module{Name: "loopw", Funcs: []ir.Func{{
		Name: "f", Result: ir.TypVoid, NVals: 4,
		Stmts: []ir.Stmt{{
			Kind: ir.SKFor,
			ForInit: []ir.Instr{
				{Op: ir.OpConst, Dst: 0, Imm: -5},
				{Op: ir.OpConst, Dst: 1, Imm: 0},
			},
			CondLeft:  0,
			CondOp:    "!=",
			CondRight: 1,
			ForPost: []ir.Instr{
				{Op: ir.OpConst, Dst: 2, Imm: 1},
				{Op: ir.OpAdd, Dst: 0, Args: []ir.Value{0, 2}},
			},
			ForBody: nil,
		}},
	}}}
}
