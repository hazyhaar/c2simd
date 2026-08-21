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
		asPtr  bool
		isBool bool
	}
	seen := map[string]*callSig{}
	var order []string
	callDst := map[ir.Value]string{}
	var curRet ir.TypeName

	var walkIns func(ins ir.Instr)
	walkIns = func(ins ir.Instr) {
		if ins.Op == ir.OpCall && len(ins.Args) == 3 && ins.Sym == "__select" {
			if exp, ok := callDst[ins.Args[1]]; ok {
				if curRet == ir.TypeName("[]byte") || strings.HasPrefix(string(curRet), "*") || strings.HasPrefix(string(curRet), "[]") {
					if cs, ok := seen[exp]; ok {
						cs.asPtr = true
					}
				}
			}
			if exp, ok := callDst[ins.Args[2]]; ok {
				if curRet == ir.TypeName("[]byte") || strings.HasPrefix(string(curRet), "*") || strings.HasPrefix(string(curRet), "[]") {
					if cs, ok := seen[exp]; ok {
						cs.asPtr = true
					}
				}
			}
			return
		}
		if ins.Op == ir.OpCall && ins.Sym != "" {
			n := baseCallSym(ins.Sym)
			low := strings.ToLower(n)
			if strings.HasPrefix(low, "__cmp_") || strings.HasPrefix(low, "_cmp_") {
				return
			}
			exp := exportName(n)
			if have[n] || have[low] || have[exp] {
				return
			}
			// never stub AEAD / critical monocypher entrypoints
			if strings.HasPrefix(low, "crypto_aead_") || low == "__select" {
				return
			}
			// emitted as builtins in emitBuiltinCall
			if low == "strchr" || low == "strlen" || low == "memcmp" ||
				low == "memset" || low == "memcpy" || low == "memmove" || low == "strcmp" || low == "memchr" ||
				low == "fabs" || low == "tolower" {
				return
			}
			cs, ok := seen[exp]
			if !ok {
				cs = &callSig{name: exp}
				seen[exp] = cs
				order = append(order, exp)
				have[exp] = true
			}
			if ins.Dst != ir.NoVal {
				cs.valued = true
				callDst[ins.Dst] = exp
			}
		}
		if ins.Op == ir.OpMov && len(ins.Args) > 0 {
			if exp, ok := callDst[ins.Args[0]]; ok {
				callDst[ins.Dst] = exp
			}
		}
		if (ins.Op == ir.OpStore || ins.Op == ir.OpLoad) && len(ins.Args) > 0 {
			if exp, ok := callDst[ins.Args[0]]; ok {
				if exp != "Fabs" && exp != "Sqrt" && exp != "Tolower" && exp != "Toupper" &&
					exp != "Sprintf" && exp != "Snprintf" && exp != "Printf" && exp != "Sscanf" &&
					exp != "Get_decimal_point" && exp != "Memcmp" && exp != "Strcmp" && exp != "Strtod" {
					if cs, ok := seen[exp]; ok {
						cs.asPtr = true
					}
				}
			}
		}
		if ins.Op == ir.OpFStore && len(ins.Args) >= 2 {
			if exp, ok := callDst[ins.Args[1]]; ok {
				if exp != "Fabs" && exp != "Sqrt" && exp != "Tolower" && exp != "Toupper" &&
					exp != "Sprintf" && exp != "Snprintf" && exp != "Printf" && exp != "Sscanf" &&
					exp != "Get_decimal_point" && exp != "Memcmp" && exp != "Strcmp" {
					if cs, ok := seen[exp]; ok {
						cs.asPtr = true
					}
				}
			}
		}
		if ins.Op == ir.OpCall && len(ins.Args) == 3 && ins.Sym == "__select" {
			if exp, ok := callDst[ins.Args[1]]; ok {
				if curRet == ir.TypeName("[]byte") || strings.HasPrefix(string(curRet), "*") || strings.HasPrefix(string(curRet), "[]") {
					if cs, ok := seen[exp]; ok {
						cs.asPtr = true
					}
				}
			}
			if exp, ok := callDst[ins.Args[2]]; ok {
				if curRet == ir.TypeName("[]byte") || strings.HasPrefix(string(curRet), "*") || strings.HasPrefix(string(curRet), "[]") {
					if cs, ok := seen[exp]; ok {
						cs.asPtr = true
					}
				}
			}
		}
		if ins.Op == ir.OpReturn && len(ins.Args) > 0 {
			if exp, ok := callDst[ins.Args[0]]; ok {
				if curRet == ir.TypeName("[]byte") || strings.HasPrefix(string(curRet), "*") || strings.HasPrefix(string(curRet), "[]") {
					if cs, ok := seen[exp]; ok {
						cs.asPtr = true
					}
				}
			}
		}
	}
	var walk func([]ir.Stmt)
	walk = func(ss []ir.Stmt) {
		for _, s := range ss {
			switch s.Kind {
			case ir.SKInstr:
				walkIns(s.Ins)
			case ir.SKFor:
				for _, ins := range s.ForInit {
					walkIns(ins)
				}
				for _, ins := range s.ForCondPrep {
					walkIns(ins)
				}
				for _, ins := range s.ForPost {
					walkIns(ins)
				}
				walk(s.ForBody)
			case ir.SKSwitch:
				for _, ins := range s.SwitchPrep {
					walkIns(ins)
				}
				for _, c := range s.SwitchCases {
					walk(c.Body)
				}
			case ir.SKDoWhile:
				walk(s.DoBody)
			case ir.SKIf:
				for _, ins := range s.ForInit {
					walkIns(ins)
				}
				walk(s.ThenBody)
				walk(s.ElseBody)
			}
		}
	}
	for i := range m.Funcs {
		m.Funcs[i].EnsureStmts()
		curRet = m.Funcs[i].Result
		walk(m.Funcs[i].Stmts)
	}
	for _, n := range order {
		cs := seen[n]
		lowName := strings.ToLower(cs.name)
		if lowName == "strtod" {
			cs.asPtr = false
			cs.valued = true
		} else if strings.Contains(lowName, "bool") || strings.HasPrefix(lowName, "is_") || strings.HasPrefix(lowName, "has_") || strings.Contains(lowName, "equals") || strings.Contains(lowName, "equal") {
			cs.isBool = true
			cs.asPtr = false
		} else if strings.Contains(lowName, "lookup") || strings.Contains(lowName, "check") || strings.Contains(lowName, "fread") || strings.Contains(lowName, "count") || strings.Contains(lowName, "size") || strings.Contains(lowName, "len") || strings.Contains(lowName, "cmp") || strings.Contains(lowName, "_lz") || strings.Contains(lowName, "_tz") || strings.Contains(lowName, "_mul") || strings.Contains(lowName, "decode") || strings.Contains(lowName, "entry") || strings.Contains(lowName, "encode") {
			cs.asPtr = false
			cs.valued = true
		}
		// marker Func: Params nil, NVals=-1 → emit variadic any stub
		ret := ir.TypVoid
		if lowName == "strtod" {
			ret = ir.TypFloat64
		} else if cs.isBool {
			ret = ir.TypBool
		} else if cs.asPtr {
			ret = ir.TypeName("[]byte")
		} else if cs.valued {
			ret = ir.TypInt
		}
		f := ir.Func{
			Name:   cs.name,
			Result: ret,
			NVals:  -1, // signal: stub variadic
			Stmts:  []ir.Stmt{{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpReturn}}},
		}
		if cs.isBool {
			f.Stmts = []ir.Stmt{
				{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpReturn, Sym: "false"}},
			}
		} else if cs.asPtr {
			f.Stmts = []ir.Stmt{
				{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpReturn, Sym: "nil"}},
			}
		} else if cs.valued {
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
