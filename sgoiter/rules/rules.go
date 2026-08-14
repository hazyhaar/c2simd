// Package rules — thésaurus de règles IR ARCHTIME (golden in→want).
package rules

import (
	"fmt"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/ir"
)

// Kind classifies a rule.
type Kind string

const (
	KindRewrite Kind = "rewrite"
	KindGuard   Kind = "guard" // documentation / reject pattern
)

// Def is one thesaurus entry.
type Def struct {
	ID      string
	Kind    Kind
	Summary string
	// Apply transforms a module; may return m unchanged.
	Apply func(m *ir.Module) (*ir.Module, error)
}

// Table is the closed list of landed rules (ARCHTIME).
var Table = []Def{
	{ID: "const_fold_add", Kind: KindRewrite, Summary: "const+const → const", Apply: constFoldAdd},
	{ID: "const_fold_mul", Kind: KindRewrite, Summary: "const*const → const", Apply: constFoldMul},
	{ID: "const_fold_mul0", Kind: KindRewrite, Summary: "x*0 → 0", Apply: constFoldMul0},
	{ID: "strength_mul1", Kind: KindRewrite, Summary: "x*1 → x", Apply: strengthMul1},
	{ID: "add_zero", Kind: KindRewrite, Summary: "x+0 → x", Apply: addZero},
	{ID: "xor_self", Kind: KindRewrite, Summary: "x^x → 0", Apply: xorSelf},
	{ID: "const_fold_sub", Kind: KindRewrite, Summary: "const-const → const", Apply: constFoldSub},
	{ID: "sub_zero", Kind: KindRewrite, Summary: "x-0 → x", Apply: subZero},
	{ID: "and_or_self", Kind: KindRewrite, Summary: "x&x → x ; x|x → x", Apply: andSelf},
	{ID: "guard_no_generic_simd", Kind: KindGuard, Summary: "forbid generic loop→simd rule id", Apply: nil},
}

// Applied returns rewrite rules only.
func Applied() []Def {
	var out []Def
	for _, d := range Table {
		if d.Kind == KindRewrite && d.Apply != nil {
			out = append(out, d)
		}
	}
	return out
}

// ApplyAll runs all rewrite rules in order until fixed point (max 8).
func ApplyAll(m *ir.Module) (*ir.Module, error) {
	if m == nil {
		return nil, fmt.Errorf("rules: nil module")
	}
	cur := cloneMod(m)
	for round := 0; round < 8; round++ {
		changed := false
		for _, r := range Applied() {
			next, err := r.Apply(cur)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", r.ID, err)
			}
			eq, err := ir.EqualJSON(cur, next)
			if err != nil {
				return nil, err
			}
			if !eq {
				cur = next
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	return cur, nil
}

func constFoldAdd(m *ir.Module) (*ir.Module, error) {
	return mapInstr(m, func(ins ir.Instr, f *ir.Func) (ir.Instr, bool) {
		if ins.Op != ir.OpAdd || len(ins.Args) != 2 {
			return ins, false
		}
		c0, ok0 := constOf(f, ins.Args[0])
		c1, ok1 := constOf(f, ins.Args[1])
		if !ok0 || !ok1 {
			return ins, false
		}
		return ir.Instr{Op: ir.OpConst, Dst: ins.Dst, Imm: c0 + c1}, true
	})
}

// constFoldMul folds const*const. blake2b sigma indexing emitted `uint64(2) * uint64(2)`
// per G-function because only the additive form was folded.
func constFoldMul(m *ir.Module) (*ir.Module, error) {
	return mapInstr(m, func(ins ir.Instr, f *ir.Func) (ir.Instr, bool) {
		if ins.Op != ir.OpMul || len(ins.Args) != 2 {
			return ins, false
		}
		c0, ok0 := constOf(f, ins.Args[0])
		c1, ok1 := constOf(f, ins.Args[1])
		if !ok0 || !ok1 {
			return ins, false
		}
		p := c0 * c1
		if c0 != 0 && (p/c0 != c1) {
			return ins, false // overflow: keep the runtime multiply
		}
		return ir.Instr{Op: ir.OpConst, Dst: ins.Dst, Imm: p}, true
	})
}

func constFoldMul0(m *ir.Module) (*ir.Module, error) {
	return mapInstr(m, func(ins ir.Instr, f *ir.Func) (ir.Instr, bool) {
		if ins.Op != ir.OpMul || len(ins.Args) != 2 {
			return ins, false
		}
		if c, ok := constOf(f, ins.Args[0]); ok && c == 0 {
			return ir.Instr{Op: ir.OpConst, Dst: ins.Dst, Imm: 0}, true
		}
		if c, ok := constOf(f, ins.Args[1]); ok && c == 0 {
			return ir.Instr{Op: ir.OpConst, Dst: ins.Dst, Imm: 0}, true
		}
		return ins, false
	})
}

func strengthMul1(m *ir.Module) (*ir.Module, error) {
	return mapInstr(m, func(ins ir.Instr, f *ir.Func) (ir.Instr, bool) {
		if ins.Op != ir.OpMul || len(ins.Args) != 2 {
			return ins, false
		}
		if c, ok := constOf(f, ins.Args[1]); ok && c == 1 {
			return ir.Instr{Op: ir.OpMov, Dst: ins.Dst, Args: []ir.Value{ins.Args[0]}}, true
		}
		if c, ok := constOf(f, ins.Args[0]); ok && c == 1 {
			return ir.Instr{Op: ir.OpMov, Dst: ins.Dst, Args: []ir.Value{ins.Args[1]}}, true
		}
		return ins, false
	})
}

func addZero(m *ir.Module) (*ir.Module, error) {
	return mapInstr(m, func(ins ir.Instr, f *ir.Func) (ir.Instr, bool) {
		if ins.Op != ir.OpAdd || len(ins.Args) != 2 {
			return ins, false
		}
		if c, ok := constOf(f, ins.Args[1]); ok && c == 0 {
			return ir.Instr{Op: ir.OpMov, Dst: ins.Dst, Args: []ir.Value{ins.Args[0]}}, true
		}
		if c, ok := constOf(f, ins.Args[0]); ok && c == 0 {
			return ir.Instr{Op: ir.OpMov, Dst: ins.Dst, Args: []ir.Value{ins.Args[1]}}, true
		}
		return ins, false
	})
}

func xorSelf(m *ir.Module) (*ir.Module, error) {
	return mapInstr(m, func(ins ir.Instr, f *ir.Func) (ir.Instr, bool) {
		if ins.Op != ir.OpXor || len(ins.Args) != 2 {
			return ins, false
		}
		if ins.Args[0] == ins.Args[1] {
			return ir.Instr{Op: ir.OpConst, Dst: ins.Dst, Imm: 0}, true
		}
		return ins, false
	})
}

func defines(op ir.Op) bool {
	switch op {
	case ir.OpReturn, ir.OpNop:
		return false
	default:
		return true
	}
}

func constOf(f *ir.Func, v ir.Value) (int64, bool) {
	// Single defining Const only. Taking the *last* def was unsound once Body
	// held a flattened tree: `if (c) { v5 = x } else { v5 = 3 }` would fold every
	// read of v5 to 3. A register written more than once is never folded.
	var imm int64
	defs := 0
	for _, ins := range f.Body {
		if !defines(ins.Op) || ins.Dst != v {
			continue
		}
		defs++
		if defs > 1 {
			return 0, false
		}
		if ins.Op != ir.OpConst {
			return 0, false
		}
		imm = ins.Imm
	}
	if defs != 1 {
		return 0, false
	}
	return imm, true
}

func mapInstr(m *ir.Module, fn func(ir.Instr, *ir.Func) (ir.Instr, bool)) (*ir.Module, error) {
	out := cloneMod(m)
	for fi := range out.Funcs {
		f := &out.Funcs[fi]
		f.EnsureStmts()
		// constOf scans the flat Body; without this the rewrite rules saw an empty
		// body on every front v0.3+ module and folded nothing outside unit tests.
		f.Flatten()
		var walk func(*[]ir.Stmt)
		walk = func(ss *[]ir.Stmt) {
			for i := range *ss {
				s := &(*ss)[i]
				switch s.Kind {
				case ir.SKInstr:
					if ni, ok := fn(s.Ins, f); ok {
						s.Ins = ni
					}
				case ir.SKFor:
					for j := range s.ForInit {
						if ni, ok := fn(s.ForInit[j], f); ok {
							s.ForInit[j] = ni
						}
					}
					for j := range s.ForCondPrep {
						if ni, ok := fn(s.ForCondPrep[j], f); ok {
							s.ForCondPrep[j] = ni
						}
					}
					for j := range s.ForPost {
						if ni, ok := fn(s.ForPost[j], f); ok {
							s.ForPost[j] = ni
						}
					}
					walk(&s.ForBody)
				case ir.SKSwitch:
					for j := range s.SwitchPrep {
						if ni, ok := fn(s.SwitchPrep[j], f); ok {
							s.SwitchPrep[j] = ni
						}
					}
					for ci := range s.SwitchCases {
						walk(&s.SwitchCases[ci].Body)
					}
				case ir.SKDoWhile:
					// C macros expand to do{…}while(0); blake2b holds its 8 G-functions
					// there, so skipping this arm hid most of the kernel from the rules.
					walk(&s.DoBody)
				case ir.SKIf:
					for j := range s.ForInit { // cond prep lives in ForInit
						if ni, ok := fn(s.ForInit[j], f); ok {
							s.ForInit[j] = ni
						}
					}
					walk(&s.ThenBody)
					walk(&s.ElseBody)
				}
			}
		}
		walk(&f.Stmts)
		f.Flatten()
	}
	return out, nil
}

func cloneMod(m *ir.Module) *ir.Module {
	b, _ := ir.Marshal(m)
	n, _ := ir.Unmarshal(b)
	return n
}

// HasGenericSIMDRule is the mechanical guard for F-sgoiter-q2.
func HasGenericSIMDRule() bool {
	for _, d := range Table {
		if d.ID == "generic_loop_simd" || d.ID == "loop_to_simd" {
			return true
		}
	}
	return false
}

func subZero(m *ir.Module) (*ir.Module, error) {
	return mapInstr(m, func(ins ir.Instr, f *ir.Func) (ir.Instr, bool) {
		if ins.Op != ir.OpSub || len(ins.Args) != 2 {
			return ins, false
		}
		if c, ok := constOf(f, ins.Args[1]); ok && c == 0 {
			return ir.Instr{Op: ir.OpMov, Dst: ins.Dst, Args: []ir.Value{ins.Args[0]}}, true
		}
		return ins, false
	})
}

func constFoldSub(m *ir.Module) (*ir.Module, error) {
	return mapInstr(m, func(ins ir.Instr, f *ir.Func) (ir.Instr, bool) {
		if ins.Op != ir.OpSub || len(ins.Args) != 2 {
			return ins, false
		}
		c0, ok0 := constOf(f, ins.Args[0])
		c1, ok1 := constOf(f, ins.Args[1])
		if !ok0 || !ok1 {
			return ins, false
		}
		return ir.Instr{Op: ir.OpConst, Dst: ins.Dst, Imm: c0 - c1}, true
	})
}

func andSelf(m *ir.Module) (*ir.Module, error) {
	return mapInstr(m, func(ins ir.Instr, f *ir.Func) (ir.Instr, bool) {
		if ins.Op != ir.OpAnd && ins.Op != ir.OpOr {
			return ins, false
		}
		if len(ins.Args) != 2 || ins.Args[0] != ins.Args[1] {
			return ins, false
		}
		return ir.Instr{Op: ir.OpMov, Dst: ins.Dst, Args: []ir.Value{ins.Args[0]}}, true
	})
}
