package emit

import "code.hazyhaar.fr/devhoros/c2simd/sgoiter/ir"

// allocaEscapes reports whether an Alloca vreg is passed to a call or taken
// in a way that requires []T. Pure indexed load/store → can stay [N]T.
func allocaEscapes(f *ir.Func, v ir.Value) bool {
	var walk func([]ir.Stmt) bool
	checkIns := func(ins ir.Instr) bool {
		switch ins.Op {
		case ir.OpCall:
			for _, a := range ins.Args {
				if a == v {
					return true
				}
			}
		case ir.OpMov:
			// copy of the pointer to another reg — conservative escape
			if len(ins.Args) == 1 && ins.Args[0] == v {
				return true
			}
		case ir.OpLoad:
			// load *v (no index) as whole — escape-ish
			if len(ins.Args) == 1 && ins.Args[0] == v {
				return true
			}
		case ir.OpStore:
			// store to *v without index
			if len(ins.Args) == 2 && ins.Args[0] == v {
				return true
			}
			// store of v as value
			if len(ins.Args) >= 2 && ins.Args[len(ins.Args)-1] == v {
				// might be storing pointer — escape
				if len(ins.Args) == 2 {
					return true
				}
			}
		case ir.OpReturn:
			for _, a := range ins.Args {
				if a == v {
					return true
				}
			}
		}
		return false
	}
	walk = func(ss []ir.Stmt) bool {
		for _, s := range ss {
			switch s.Kind {
			case ir.SKInstr:
				if checkIns(s.Ins) {
					return true
				}
			case ir.SKFor:
				for _, ins := range s.ForInit {
					if checkIns(ins) {
						return true
					}
				}
				for _, ins := range s.ForCondPrep {
					if checkIns(ins) {
						return true
					}
				}
				for _, ins := range s.ForPost {
					if checkIns(ins) {
						return true
					}
				}
				if walk(s.ForBody) {
					return true
				}
			case ir.SKIf:
				for _, ins := range s.ForInit {
					if checkIns(ins) {
						return true
					}
				}
				if walk(s.ThenBody) || walk(s.ElseBody) {
					return true
				}
			case ir.SKDoWhile:
				if walk(s.DoBody) {
					return true
				}
			case ir.SKSwitch:
				for _, ins := range s.SwitchPrep {
					if checkIns(ins) {
						return true
					}
				}
				for _, c := range s.SwitchCases {
					if walk(c.Body) {
						return true
					}
				}
			}
		}
		return false
	}
	f.EnsureStmts()
	return walk(f.Stmts)
}
