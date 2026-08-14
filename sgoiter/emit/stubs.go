package emit

import (
	"strings"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/ir"
)

// StubParamType marks the single variadic parameter of a stub. The call sites
// carry types the front could not recover, so the signature stays variadic;
// what changed is the body, which panics instead of returning a plausible zero.
const StubParamType = ir.TypeName("...any")

// IsStub reports whether f is a stub standing in for an unharvested C symbol.
func IsStub(f *ir.Func) bool {
	return len(f.Params) == 1 && f.Params[0].Type == StubParamType
}

// FillStubs appends stub funcs for called names missing from m.
// Stubs keep a variadic signature so go build succeeds regardless of call-site
// types, but their body panics: a partial harvest must not look like a result.
// It returns the names it stubbed, so a caller can report the harvest gap
// instead of shipping a module whose holes are only visible at runtime.
func FillStubs(m *ir.Module) []string {
	if m == nil {
		return nil
	}
	have := map[string]bool{}
	for _, f := range m.Funcs {
		have[f.Name] = true
		have[strings.ToLower(f.Name)] = true
		have[exportName(f.Name)] = true
	}
	type callSig struct {
		name   string
		valued bool
	}
	seen := map[string]*callSig{}
	var order []string
	var walk func([]ir.Stmt)
	walk = func(ss []ir.Stmt) {
		for _, s := range ss {
			switch s.Kind {
			case ir.SKInstr:
				if s.Ins.Op == ir.OpCall && s.Ins.Sym != "" {
					n := baseCallSym(s.Ins.Sym)
					low := strings.ToLower(n)
					if strings.HasPrefix(low, "__cmp_") || strings.HasPrefix(low, "_cmp_") {
						continue
					}
					exp := exportName(n)
					if have[n] || have[low] || have[exp] {
						continue
					}
					// never stub AEAD / critical monocypher entrypoints
					if strings.HasPrefix(low, "crypto_aead_") || low == "__select" {
						continue
					}
					// emitted as builtins in emitBuiltinCall
					if low == "strchr" || low == "strlen" || low == "memcmp" ||
						low == "memset" || low == "memcpy" || low == "memmove" || low == "strcmp" {
						continue
					}
					cs, ok := seen[exp]
					if !ok {
						cs = &callSig{name: exp}
						seen[exp] = cs
						order = append(order, exp)
						have[exp] = true
					}
					if s.Ins.Dst != ir.NoVal {
						cs.valued = true
					}
				}
			case ir.SKFor:
				walk(s.ForBody)
			case ir.SKSwitch:
				for _, c := range s.SwitchCases {
					walk(c.Body)
				}
			case ir.SKDoWhile:
				walk(s.DoBody)
			case ir.SKIf:
				walk(s.ThenBody)
				walk(s.ElseBody)
			}
		}
	}
	for i := range m.Funcs {
		m.Funcs[i].EnsureStmts()
		walk(m.Funcs[i].Stmts)
	}
	for _, n := range order {
		cs := seen[n]
		// marker Func: Params nil, NVals=-1 → emit variadic any stub
		ret := ir.TypVoid
		if cs.valued {
			ret = ir.TypInt
		}
		f := ir.Func{
			Name:   cs.name,
			Result: ret,
			NVals:  -1, // signal: stub variadic
			Stmts:  []ir.Stmt{{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpReturn}}},
		}
		if cs.valued {
			zero := ir.Value(0)
			f.NVals = 1
			f.Stmts = []ir.Stmt{
				{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpConst, Dst: zero, Imm: 0}},
				{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpReturn, Args: []ir.Value{zero}}},
			}
		}
		// keep a dummy so Name is unique; Params empty + flag via Elem on a sentinel?
		// Use Params with single "args" and special Type
		f.Params = []ir.Param{{Name: "args", Type: StubParamType}}
		m.Funcs = append(m.Funcs, f)
		have[cs.name] = true
	}
	return order
}
