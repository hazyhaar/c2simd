package rules

import (
	"strings"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/ir"
)



func init() {
	Table = append(Table,
		Def{ID: "mark_const_call_args", Kind: KindRewrite,
			Summary: "annotate calls where trailing args are const (prep for emit specialize)",
			Apply:   markConstCallArgs},
	)
}

// markConstCallArgs rewrites Call.Sym to include a suffix when the last arg is
// a single-assign const, e.g. Call vn(..., 16) → Sym "vn#c16".
// Emit can special-case these without hardcoding only "Vn" body prefix forever.
// If no call matches, module unchanged — witness forces one rewrite.
func markConstCallArgs(m *ir.Module) (*ir.Module, error) {
	out := cloneMod(m)
	changed := false
	for fi := range out.Funcs {
		f := &out.Funcs[fi]
		f.EnsureStmts()
		f.Flatten()
		var walk func(*[]ir.Stmt)
		walk = func(ss *[]ir.Stmt) {
			for i := range *ss {
				s := &(*ss)[i]
				switch s.Kind {
				case ir.SKInstr:
					if s.Ins.Op == ir.OpCall && len(s.Ins.Args) >= 1 && s.Ins.Sym != "" {
						if strings.Contains(s.Ins.Sym, "#c") {
							continue
						}
						// Only mark CT-compare style helpers (Vn); broad marking
						// polluted monocypher (FillStubs / type names with '#').
						base := s.Ins.Sym
						low := strings.ToLower(base)
						if low != "vn" && low != "crypto_verify_16" && low != "crypto_verify_32" {
							continue
						}
						last := s.Ins.Args[len(s.Ins.Args)-1]
						if c, ok := constOf(f, last); ok && (c == 16 || c == 32) {
							s.Ins.Sym = base + "#c" + itoa(c)
							changed = true
						}
					}
				case ir.SKFor:
					walk(&s.ForBody)
				case ir.SKIf:
					walk(&s.ThenBody)
					walk(&s.ElseBody)
				case ir.SKDoWhile:
					walk(&s.DoBody)
				case ir.SKSwitch:
					for ci := range s.SwitchCases {
						walk(&s.SwitchCases[ci].Body)
					}
				}
			}
		}
		walk(&f.Stmts)
		f.Flatten()
	}
	if !changed {
		return m, nil
	}
	return out, nil
}

func itoa(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [32]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// WitnessMarkConstCall for rules_test.
func WitnessMarkConstCall() *ir.Module {
	return &ir.Module{Name: "w", Funcs: []ir.Func{{
		Name: "f", Result: ir.TypInt, NVals: 4,
		Body: []ir.Instr{
			{Op: ir.OpConst, Dst: 0, Imm: 16},
			{Op: ir.OpCall, Dst: 1, Sym: "vn", Args: []ir.Value{2, 3, 0}},
			{Op: ir.OpReturn, Args: []ir.Value{1}},
		},
	}}}
}
