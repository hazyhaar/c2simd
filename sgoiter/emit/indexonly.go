package emit

import "code.hazyhaar.fr/devhoros/c2simd/sgoiter/ir"

// indexOnlyRegs finds the locals that exist only to walk an array: a counter
// started at a constant, compared against a constant, stepped by a constant,
// and read as a slice index or to derive one.
//
// Such a register can be emitted as `int` from the start, which removes the
// conversion at every use — blake2b spends most of its int() calls on exactly
// two of them. Nothing about the code's shape changes, only the declared type,
// so this does not reintroduce the loop-shadowing that sank the earlier attempt.
//
// The analysis is deliberately narrow. A register is rejected as soon as it
// reaches anything that could care about its width: a bitwise operator, a call,
// a return, a stored value, or a comparison against something that is not a
// constant. `for i < len_` therefore keeps its unsigned counter, len_ being a
// parameter.
func indexOnlyRegs(f *ir.Func) map[ir.Value]bool {
	cand := map[ir.Value]bool{}
	consts := map[ir.Value]bool{}
	params := map[ir.Value]bool{}
	for _, p := range f.Params {
		params[p.Reg] = true
	}

	var instrs []ir.Instr
	collectInstrs(f.Stmts, &instrs)
	for _, ins := range instrs {
		if ins.Op == ir.OpConst {
			consts[ins.Dst] = true
		}
	}
	// Conditions live beside the instructions. A counter compared against
	// anything but a constant keeps its own type: `i < len_` reads a uint64
	// parameter, and retyping i would only move the conversion.
	var condVictims []ir.Value
	collectCondOperands(f.Stmts, consts, &condVictims)

	// seed: every register defined by a constant or by arithmetic on constants
	for _, ins := range instrs {
		switch ins.Op {
		case ir.OpConst, ir.OpAdd, ir.OpSub, ir.OpMul, ir.OpMov:
			if ins.Dst != ir.NoVal && !params[ins.Dst] {
				cand[ins.Dst] = true
			}
		}
	}

	for _, v := range condVictims {
		delete(cand, v)
	}

	// remove anything reached by a use that cares about the width
	changed := true
	for changed {
		changed = false
		for _, ins := range instrs {
			for _, v := range widthSensitiveArgs(ins, consts) {
				if cand[v] {
					delete(cand, v)
					changed = true
				}
			}
			// a candidate defined from a rejected value is rejected too
			if ins.Dst != ir.NoVal && cand[ins.Dst] {
				switch ins.Op {
				case ir.OpAdd, ir.OpSub, ir.OpMul, ir.OpMov:
					for _, a := range ins.Args {
						if !consts[a] && !cand[a] {
							delete(cand, ins.Dst)
							changed = true
						}
					}
				case ir.OpConst:
					// fine
				default:
					delete(cand, ins.Dst)
					changed = true
				}
			}
		}
	}

	// a bare constant is not worth retyping on its own
	for v := range cand {
		if consts[v] {
			delete(cand, v)
		}
	}
	return cand
}

// widthSensitiveArgs lists the operands of ins whose width the instruction
// depends on, so they must not be retyped.
func widthSensitiveArgs(ins ir.Instr, consts map[ir.Value]bool) []ir.Value {
	switch ins.Op {
	case ir.OpAnd, ir.OpOr, ir.OpXor, ir.OpShl, ir.OpShr, ir.OpNot, ir.OpDiv, ir.OpMod:
		return ins.Args
	case ir.OpCall, ir.OpReturn, ir.OpField, ir.OpFStore:
		return ins.Args
	case ir.OpLoad:
		if len(ins.Args) >= 1 {
			return ins.Args[:1] // the pointer, not the index
		}
	case ir.OpStore:
		switch len(ins.Args) {
		case 2:
			return ins.Args // *ptr = val: both
		case 3:
			return []ir.Value{ins.Args[0], ins.Args[2]} // ptr[idx] = val
		}
	case ir.OpAdd, ir.OpSub, ir.OpMul:
		// arithmetic is fine only against a constant
		if len(ins.Args) == 2 && !consts[ins.Args[0]] && !consts[ins.Args[1]] {
			return ins.Args
		}
	}
	return nil
}

// collectCondOperands gathers the operands of every loop/if condition whose
// other side is not a constant.
func collectCondOperands(ss []ir.Stmt, consts map[ir.Value]bool, out *[]ir.Value) {
	for _, s := range ss {
		switch s.Kind {
		case ir.SKFor, ir.SKIf, ir.SKDoWhile:
			l, r := s.CondLeft, s.CondRight
			if s.CondOp == "" {
				*out = append(*out, l) // truthiness: the value itself is read
			} else if !consts[r] {
				*out = append(*out, l, r)
			}
			collectCondOperands(s.ForBody, consts, out)
			collectCondOperands(s.ThenBody, consts, out)
			collectCondOperands(s.ElseBody, consts, out)
			collectCondOperands(s.DoBody, consts, out)
		case ir.SKSwitch:
			*out = append(*out, s.SwitchOn)
			for _, c := range s.SwitchCases {
				collectCondOperands(c.Body, consts, out)
			}
		}
	}
}

func collectInstrs(ss []ir.Stmt, out *[]ir.Instr) {
	for _, s := range ss {
		switch s.Kind {
		case ir.SKInstr:
			*out = append(*out, s.Ins)
		case ir.SKFor:
			*out = append(*out, s.ForInit...)
			*out = append(*out, s.ForCondPrep...)
			*out = append(*out, s.ForPost...)
			collectInstrs(s.ForBody, out)
		case ir.SKSwitch:
			*out = append(*out, s.SwitchPrep...)
			for _, c := range s.SwitchCases {
				collectInstrs(c.Body, out)
			}
		case ir.SKDoWhile:
			collectInstrs(s.DoBody, out)
		case ir.SKIf:
			*out = append(*out, s.ForInit...)
			collectInstrs(s.ThenBody, out)
			collectInstrs(s.ElseBody, out)
		}
	}
}
