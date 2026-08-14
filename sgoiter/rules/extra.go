package rules

import (
	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/ir"
)

// Extra rewrites: or-self, xor-zero, shl/shr by 0, identity mov chains.
func init() {
	// Appended at package init so Table stays the single catalogue.
	Table = append(Table,
		Def{ID: "or_self", Kind: KindRewrite, Summary: "x|x → x", Apply: orSelf},
		Def{ID: "xor_zero", Kind: KindRewrite, Summary: "x^0 → x", Apply: xorZero},
		Def{ID: "or_zero", Kind: KindRewrite, Summary: "x|0 → x", Apply: orZero},
		Def{ID: "and_ones_u64", Kind: KindRewrite, Summary: "x & -1 → x (Imm all-bits)", Apply: andAllOnes},
		Def{ID: "shl_zero", Kind: KindRewrite, Summary: "x<<0 → x", Apply: shiftZero(ir.OpShl)},
		Def{ID: "shr_zero", Kind: KindRewrite, Summary: "x>>0 → x", Apply: shiftZero(ir.OpShr)},
		Def{ID: "mov_elim", Kind: KindRewrite, Summary: "v = mov v → nop fold via mov to same", Apply: movElim},
	)
}

func orSelf(m *ir.Module) (*ir.Module, error) {
	return mapInstr(m, func(ins ir.Instr, f *ir.Func) (ir.Instr, bool) {
		if ins.Op != ir.OpOr || len(ins.Args) != 2 || ins.Args[0] != ins.Args[1] {
			return ins, false
		}
		return ir.Instr{Op: ir.OpMov, Dst: ins.Dst, Args: []ir.Value{ins.Args[0]}}, true
	})
}

func xorZero(m *ir.Module) (*ir.Module, error) {
	return mapInstr(m, func(ins ir.Instr, f *ir.Func) (ir.Instr, bool) {
		if ins.Op != ir.OpXor || len(ins.Args) != 2 {
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

func orZero(m *ir.Module) (*ir.Module, error) {
	return mapInstr(m, func(ins ir.Instr, f *ir.Func) (ir.Instr, bool) {
		if ins.Op != ir.OpOr || len(ins.Args) != 2 {
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

func andAllOnes(m *ir.Module) (*ir.Module, error) {
	return mapInstr(m, func(ins ir.Instr, f *ir.Func) (ir.Instr, bool) {
		if ins.Op != ir.OpAnd || len(ins.Args) != 2 {
			return ins, false
		}
		// Identity only for a mask that is ALL ones in a full machine word.
		// 0xffffffff is NOT identity on uint64 — it clears the high 32 bits
		// (poly1305 limbs). Treating it as -1 broke monocypher AEAD MAC parity
		// (2026-08-13). Accept only -1 or 0xffffffffffffffff.
		check := func(v ir.Value) bool {
			c, ok := constOf(f, v)
			if !ok {
				return false
			}
			if c == -1 {
				return true
			}
			return uint64(c) == 0xffffffffffffffff
		}
		if check(ins.Args[1]) {
			return ir.Instr{Op: ir.OpMov, Dst: ins.Dst, Args: []ir.Value{ins.Args[0]}}, true
		}
		if check(ins.Args[0]) {
			return ir.Instr{Op: ir.OpMov, Dst: ins.Dst, Args: []ir.Value{ins.Args[1]}}, true
		}
		return ins, false
	})
}

func shiftZero(op ir.Op) func(*ir.Module) (*ir.Module, error) {
	return func(m *ir.Module) (*ir.Module, error) {
		return mapInstr(m, func(ins ir.Instr, f *ir.Func) (ir.Instr, bool) {
			if ins.Op != op || len(ins.Args) != 2 {
				return ins, false
			}
			if c, ok := constOf(f, ins.Args[1]); ok && c == 0 {
				return ir.Instr{Op: ir.OpMov, Dst: ins.Dst, Args: []ir.Value{ins.Args[0]}}, true
			}
			return ins, false
		})
	}
}

func movElim(m *ir.Module) (*ir.Module, error) {
	return mapInstr(m, func(ins ir.Instr, f *ir.Func) (ir.Instr, bool) {
		if ins.Op != ir.OpMov || len(ins.Args) != 1 {
			return ins, false
		}
		if ins.Dst == ins.Args[0] {
			return ir.Instr{Op: ir.OpNop}, true
		}
		return ins, false
	})
}
