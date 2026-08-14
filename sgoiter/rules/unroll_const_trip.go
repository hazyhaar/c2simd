package rules

import (
	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/ir"
)

func init() {
	Table = append(Table,
		Def{ID: "unroll_const_trip_load", Kind: KindRewrite,
			Summary: "for i=0;i<N;i++ { x=load(g,i) } N const small → unroll + fold global loads",
			Apply:   unrollConstTripLoad},
	)
}

// unrollConstTripLoad unrolls empty-ish counted loops with const trip count ≤ 16
// whose body is a single Load from a base+index. Combined with fold_load_global_const_idx
// this implements N6 table-fold for const trip loops.
func unrollConstTripLoad(m *ir.Module) (*ir.Module, error) {
	out := cloneMod(m)
	changed := false
	for fi := range out.Funcs {
		f := &out.Funcs[fi]
		f.EnsureStmts()
		f.Flatten()
		ns, c := unrollStmtList(f, f.Stmts)
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

func unrollStmtList(f *ir.Func, ss []ir.Stmt) ([]ir.Stmt, bool) {
	ch := false
	var out []ir.Stmt
	for _, s := range ss {
		ns, c, flat := unrollOne(f, s)
		ch = ch || c
		if flat != nil {
			out = append(out, flat...)
			continue
		}
		out = append(out, ns)
	}
	return out, ch
}

func unrollOne(f *ir.Func, s ir.Stmt) (ir.Stmt, bool, []ir.Stmt) {
	switch s.Kind {
	case ir.SKFor:
		if flat, ok := tryUnrollConstFor(f, s); ok {
			return s, true, flat
		}
		b, c := unrollStmtList(f, s.ForBody)
		s.ForBody = b
		return s, c, nil
	case ir.SKIf:
		t, c1 := unrollStmtList(f, s.ThenBody)
		e, c2 := unrollStmtList(f, s.ElseBody)
		s.ThenBody, s.ElseBody = t, e
		return s, c1 || c2, nil
	case ir.SKDoWhile:
		b, c := unrollStmtList(f, s.DoBody)
		s.DoBody = b
		return s, c, nil
	default:
		return s, false, nil
	}
}

func tryUnrollConstFor(f *ir.Func, s ir.Stmt) ([]ir.Stmt, bool) {
	if flat, ok := tryUnrollLoadLoop(f, s); ok {
		return flat, true
	}
	// General body unroll disabled: trip-8 CRC bit loop lost carried updates
	// (2026-08-12). RemapBodyFresh kept for a future guarded re-enable.
	return nil, false
}

// tripConstN extracts N from for i < N when N is a compile-time constant.
func tripConstN(f *ir.Func, s ir.Stmt) (int64, bool) {
	n, ok := constOf(f, s.CondRight)
	if ok {
		return n, true
	}
	for _, ins := range s.ForInit {
		if ins.Dst == s.CondRight && ins.Op == ir.OpConst {
			return ins.Imm, true
		}
	}
	for _, ins := range s.ForCondPrep {
		if ins.Dst == s.CondRight && ins.Op == ir.OpConst {
			return ins.Imm, true
		}
	}
	return 0, false
}

func forInitZero(s ir.Stmt) bool {
	// CondLeft = 0  or  CondLeft = mov(const0)
	zeros := map[ir.Value]bool{}
	for _, ins := range s.ForInit {
		if ins.Op == ir.OpConst && ins.Imm == 0 {
			zeros[ins.Dst] = true
		}
		if ins.Dst == s.CondLeft && ins.Op == ir.OpConst && ins.Imm == 0 {
			return true
		}
		if ins.Dst == s.CondLeft && ins.Op == ir.OpMov && len(ins.Args) == 1 && zeros[ins.Args[0]] {
			return true
		}
	}
	return false
}

// forPostPlus1: CondLeft = CondLeft + 1 (Add dst or Add+Mov pattern).
func forPostPlus1(f *ir.Func, s ir.Stmt) bool {
	if len(s.ForPost) == 0 {
		return false
	}
	ones := map[ir.Value]bool{}
	var addDst ir.Value
	var sawAdd bool
	for _, ins := range s.ForPost {
		if ins.Op == ir.OpConst && ins.Imm == 1 {
			ones[ins.Dst] = true
		}
		if ins.Op == ir.OpAdd && len(ins.Args) == 2 {
			a, b := ins.Args[0], ins.Args[1]
			if (a == s.CondLeft && ones[b]) || (b == s.CondLeft && ones[a]) {
				addDst, sawAdd = ins.Dst, true
				if ins.Dst == s.CondLeft {
					return true
				}
			}
		}
		if sawAdd && ins.Op == ir.OpMov && ins.Dst == s.CondLeft && len(ins.Args) == 1 && ins.Args[0] == addDst {
			return true
		}
	}
	return false
}

func tryUnrollLoadLoop(f *ir.Func, s ir.Stmt) ([]ir.Stmt, bool) {
	if s.Kind != ir.SKFor || s.CondOp != "<" || len(s.ForBody) != 1 {
		return nil, false
	}
	body := s.ForBody[0]
	if body.Kind != ir.SKInstr || body.Ins.Op != ir.OpLoad || len(body.Ins.Args) != 2 {
		return nil, false
	}
	if body.Ins.Args[1] != s.CondLeft {
		return nil, false
	}
	n, ok := tripConstN(f, s)
	if !ok || n <= 0 || n > 16 || !forInitZero(s) {
		return nil, false
	}
	base := body.Ins.Args[0]
	dst := body.Ins.Dst
	elem := body.Ins.Elem
	var flat []ir.Stmt
	for i := int64(0); i < n; i++ {
		idx := f.Alloc()
		flat = append(flat,
			ir.Stmt{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpConst, Dst: idx, Imm: i}},
			ir.Stmt{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpLoad, Dst: dst, Args: []ir.Value{base, idx}, Elem: elem}},
		)
	}
	return flat, true
}

// tryUnrollGeneralConstFor: for i=0; i<N; i++ { body } with N≤16 and modest body.
// Enables blake2b 12-round expand so sigma[const] folds.
func tryUnrollGeneralConstFor(f *ir.Func, s ir.Stmt) ([]ir.Stmt, bool) {
	if s.Kind != ir.SKFor || s.CondOp != "<" {
		return nil, false
	}
	n, ok := tripConstN(f, s)
	if !ok || n < 2 || n > 16 {
		return nil, false
	}
	if !forInitZero(s) || !forPostPlus1(f, s) {
		return nil, false
	}
	// body size guard — expanded weight cap keeps emit folds tractable
	// (blake G-macro body ~1k instr × 12 blew foldSingleUse to minutes).
	if len(s.ForBody) == 0 || len(s.ForBody) > 128 {
		return nil, false
	}
	w := bodyWeight(s.ForBody)
	if w < 1 || w*int(n) > 256 {
		return nil, false
	}
	// refuse nested FOR only; SKDoWhile (G-macro do{…}while(0)) and SKIf clone fine
	for _, b := range s.ForBody {
		if b.Kind == ir.SKFor {
			return nil, false
		}
	}
	var flat []ir.Stmt
	for i := int64(0); i < n; i++ {
		// Fresh SSA temps per iteration so emit single-use / copy-prop can fire.
		flat = append(flat, ir.Stmt{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpConst, Dst: s.CondLeft, Imm: i}})
		flat = append(flat, remapBodyFresh(f, cloneStmts(s.ForBody), s.CondLeft)...)
	}
	return flat, true
}

func bodyWeight(ss []ir.Stmt) int {
	n := 0
	for _, s := range ss {
		switch s.Kind {
		case ir.SKInstr:
			n++
		case ir.SKFor:
			n += bodyWeight(s.ForBody) + len(s.ForInit) + len(s.ForPost)
		case ir.SKDoWhile:
			n += bodyWeight(s.DoBody)
		case ir.SKIf:
			n += bodyWeight(s.ThenBody) + bodyWeight(s.ElseBody)
		default:
			n++
		}
	}
	return n
}

// remapBodyFresh allocates new value IDs for every destination defined in body
// except keep (loop index). Outside values (stack arrays, params) stay put.
func remapBodyFresh(f *ir.Func, body []ir.Stmt, keep ir.Value) []ir.Stmt {
	m := map[ir.Value]ir.Value{}
	var collect func([]ir.Stmt)
	collect = func(ss []ir.Stmt) {
		for _, s := range ss {
			mark := func(dst ir.Value) {
				if dst == 0 || dst == keep {
					return
				}
				if _, ok := m[dst]; !ok {
					m[dst] = f.Alloc()
				}
			}
			switch s.Kind {
			case ir.SKInstr:
				mark(s.Ins.Dst)
			case ir.SKFor:
				for _, ins := range s.ForInit {
					mark(ins.Dst)
				}
				for _, ins := range s.ForCondPrep {
					mark(ins.Dst)
				}
				for _, ins := range s.ForPost {
					mark(ins.Dst)
				}
				collect(s.ForBody)
			case ir.SKDoWhile:
				collect(s.DoBody)
			case ir.SKIf:
				for _, ins := range s.ForInit {
					mark(ins.Dst)
				}
				collect(s.ThenBody)
				collect(s.ElseBody)
			}
		}
	}
	collect(body)
	if len(m) == 0 {
		return body
	}
	sub := func(v ir.Value) ir.Value {
		if n, ok := m[v]; ok {
			return n
		}
		return v
	}
	subIns := func(ins ir.Instr) ir.Instr {
		ins.Dst = sub(ins.Dst)
		if len(ins.Args) > 0 {
			args := make([]ir.Value, len(ins.Args))
			for i, a := range ins.Args {
				args[i] = sub(a)
			}
			ins.Args = args
		}
		return ins
	}
	var rewrite func([]ir.Stmt) []ir.Stmt
	rewrite = func(ss []ir.Stmt) []ir.Stmt {
		out := make([]ir.Stmt, len(ss))
		for i, s := range ss {
			switch s.Kind {
			case ir.SKInstr:
				s.Ins = subIns(s.Ins)
			case ir.SKFor:
				for j := range s.ForInit {
					s.ForInit[j] = subIns(s.ForInit[j])
				}
				for j := range s.ForCondPrep {
					s.ForCondPrep[j] = subIns(s.ForCondPrep[j])
				}
				for j := range s.ForPost {
					s.ForPost[j] = subIns(s.ForPost[j])
				}
				s.CondLeft, s.CondRight = sub(s.CondLeft), sub(s.CondRight)
				s.ForBody = rewrite(s.ForBody)
			case ir.SKDoWhile:
				s.CondLeft, s.CondRight = sub(s.CondLeft), sub(s.CondRight)
				s.DoBody = rewrite(s.DoBody)
			case ir.SKIf:
				for j := range s.ForInit {
					s.ForInit[j] = subIns(s.ForInit[j])
				}
				s.CondLeft, s.CondRight = sub(s.CondLeft), sub(s.CondRight)
				s.ThenBody = rewrite(s.ThenBody)
				s.ElseBody = rewrite(s.ElseBody)
			}
			out[i] = s
		}
		return out
	}
	return rewrite(body)
}

func cloneStmts(ss []ir.Stmt) []ir.Stmt {
	if ss == nil {
		return nil
	}
	out := make([]ir.Stmt, len(ss))
	for i, s := range ss {
		out[i] = cloneStmtDeep(s)
	}
	return out
}

func cloneStmtDeep(s ir.Stmt) ir.Stmt {
	s.ForInit = append([]ir.Instr(nil), s.ForInit...)
	s.ForCondPrep = append([]ir.Instr(nil), s.ForCondPrep...)
	s.ForPost = append([]ir.Instr(nil), s.ForPost...)
	s.ForBody = cloneStmts(s.ForBody)
	s.DoBody = cloneStmts(s.DoBody)
	s.ThenBody = cloneStmts(s.ThenBody)
	s.ElseBody = cloneStmts(s.ElseBody)
	if s.Ins.Args != nil {
		s.Ins.Args = append([]ir.Value(nil), s.Ins.Args...)
	}
	return s
}

// WitnessUnrollConstTrip for rules_test.
func WitnessUnrollConstTrip() *ir.Module {
	// i=0; i<4; i++; x = load(g, i)
	return &ir.Module{
		Name: "w",
		Globals: []ir.Global{{
			Name: "T", Type: ir.TypUint8, InitCSV: "1,2,3,4",
		}},
		Funcs: []ir.Func{{
			Name: "f", Result: ir.TypUint8, NVals: 4,
			Stmts: []ir.Stmt{
				{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpMov, Dst: 0, Sym: "global:T"}},
				{
					Kind: ir.SKFor,
					ForInit: []ir.Instr{
						{Op: ir.OpConst, Dst: 1, Imm: 0},
						{Op: ir.OpConst, Dst: 2, Imm: 4},
					},
					CondLeft: 1, CondOp: "<", CondRight: 2,
					ForPost: []ir.Instr{
						{Op: ir.OpConst, Dst: 3, Imm: 1},
						{Op: ir.OpAdd, Dst: 1, Args: []ir.Value{1, 3}},
					},
					ForBody: []ir.Stmt{
						{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpLoad, Dst: 0, Args: []ir.Value{0, 1}, Elem: ir.TypUint8}},
					},
				},
			},
		}},
	}
}
