package rules

import (
	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/ir"
)

func init() {
	Table = append(Table,
		Def{ID: "dce_if_const_eq", Kind: KindRewrite,
			Summary: "if (const==const) keep then or else only",
			Apply:   dceIfConstEq},
	)
}

// dceIfConstEq: if CondLeft and CondRight are both compile-time consts,
// replace the If with Then or Else body (flattened into parent).
func dceIfConstEq(m *ir.Module) (*ir.Module, error) {
	out := cloneMod(m)
	changed := false
	for fi := range out.Funcs {
		f := &out.Funcs[fi]
		f.EnsureStmts()
		f.Flatten()
		ns, c := dceStmtList(f, f.Stmts)
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

func dceStmtList(f *ir.Func, ss []ir.Stmt) ([]ir.Stmt, bool) {
	ch := false
	var out []ir.Stmt
	for _, s := range ss {
		ns, c, flat := dceOne(f, s)
		ch = ch || c
		if flat != nil {
			out = append(out, flat...)
			continue
		}
		out = append(out, ns)
	}
	return out, ch
}

// returns (stmt, changed, flattenIntoParent)
func dceOne(f *ir.Func, s ir.Stmt) (ir.Stmt, bool, []ir.Stmt) {
	switch s.Kind {
	case ir.SKIf:
		// Cond prep may define consts
		f.Flatten()
		// Evaluate const condition: need CondLeft/Right as consts
		// CondOp == or empty truthiness
		lv, okL := constOf(f, s.CondLeft)
		rv, okR := constOf(f, s.CondRight)
		if s.CondOp == "" || s.CondOp == "!=" || s.CondOp == "==" || s.CondOp == "<" || s.CondOp == ">" {
			if okL && (s.CondOp == "" || okR) {
				takeThen := false
				switch s.CondOp {
				case "", "!=":
					// truthiness of left, or !=
					if s.CondOp == "" {
						takeThen = lv != 0
					} else {
						takeThen = lv != rv
					}
				case "==":
					takeThen = lv == rv
				case "<":
					takeThen = lv < rv
				case ">":
					takeThen = lv > rv
				}
				if s.CondNot {
					takeThen = !takeThen
				}
				body := s.ElseBody
				if takeThen {
					body = s.ThenBody
				}
				// include cond prep as dead? skip ForInit (pure)
				return s, true, body
			}
		}
		th, c1 := dceStmtList(f, s.ThenBody)
		el, c2 := dceStmtList(f, s.ElseBody)
		s.ThenBody, s.ElseBody = th, el
		return s, c1 || c2, nil
	case ir.SKFor:
		b, c := dceStmtList(f, s.ForBody)
		s.ForBody = b
		return s, c, nil
	case ir.SKDoWhile:
		b, c := dceStmtList(f, s.DoBody)
		s.DoBody = b
		return s, c, nil
	case ir.SKSwitch:
		ch := false
		for i := range s.SwitchCases {
			b, c := dceStmtList(f, s.SwitchCases[i].Body)
			s.SwitchCases[i].Body = b
			ch = ch || c
		}
		return s, ch, nil
	default:
		return s, false, nil
	}
}

// WitnessDceIfConstEq for rules_test.
func WitnessDceIfConstEq() *ir.Module {
	// if (1 == 1) return 7; else return 9;
	return &ir.Module{Name: "w", Funcs: []ir.Func{{
		Name: "f", Result: ir.TypInt, NVals: 4,
		Stmts: []ir.Stmt{
			{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpConst, Dst: 0, Imm: 1}},
			{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpConst, Dst: 1, Imm: 1}},
			{
				Kind:      ir.SKIf,
				CondLeft:  0,
				CondOp:    "==",
				CondRight: 1,
				ThenBody: []ir.Stmt{
					{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpConst, Dst: 2, Imm: 7}},
					{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpReturn, Args: []ir.Value{2}}},
				},
				ElseBody: []ir.Stmt{
					{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpConst, Dst: 3, Imm: 9}},
					{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpReturn, Args: []ir.Value{3}}},
				},
			},
		},
	}}}
}
