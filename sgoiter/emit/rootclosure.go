package emit

import (
	"sort"
	"strings"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/ir"
)

// RootClosure returns m reduced to its entry points and everything they reach.
// A C `static` function is an internal helper: it survives only when a reachable
// function calls it. Tables follow the same rule.
//
// This is opt-in. Emitting the whole harvest is what the bench harnesses expect,
// and a kernel's entry point is not always the C source's: switching the default
// would change the surface every harness links against. The flag exists so a
// reader can be handed the closure without that risk — an audit reading twelve
// helpers around one exercised function concludes the port is half-finished,
// which is a false reading of a dogfood module.
// A C file's entry points are not always the ones a harness exercises:
// tweetnacl_dogfood exposes only randombytes, while the bench calls vn, which C
// declares static. Naming roots explicitly is therefore part of the contract,
// not a convenience.
func RootClosure(m *ir.Module, roots []string) *ir.Module {
	if m == nil {
		return nil
	}
	named := map[string]bool{}
	for _, r := range roots {
		r = strings.TrimSpace(r)
		if r != "" {
			named[r] = true
			named[strings.ToLower(r)] = true
		}
	}
	byName := map[string]*ir.Func{}
	for i := range m.Funcs {
		byName[m.Funcs[i].Name] = &m.Funcs[i]
		byName[strings.ToLower(m.Funcs[i].Name)] = &m.Funcs[i]
	}

	keep := map[string]bool{}
	var visit func(f *ir.Func)
	visit = func(f *ir.Func) {
		if f == nil || keep[f.Name] {
			return
		}
		keep[f.Name] = true
		for _, callee := range calleesOf(f) {
			if g, ok := byName[callee]; ok {
				visit(g)
			} else if g, ok := byName[strings.ToLower(callee)]; ok {
				visit(g)
			}
		}
	}
	for i := range m.Funcs {
		f := &m.Funcs[i]
		if len(named) > 0 {
			if named[f.Name] || named[strings.ToLower(f.Name)] || named[exportName(f.Name)] {
				visit(f)
			}
			continue
		}
		if !f.Static {
			visit(f)
		}
	}
	// Nothing matched: keep the module whole rather than emit an empty file.
	if len(keep) == 0 {
		return m
	}

	out := &ir.Module{Name: m.Name, Structs: m.Structs}
	for i := range m.Funcs {
		if keep[m.Funcs[i].Name] {
			out.Funcs = append(out.Funcs, m.Funcs[i])
		}
	}
	used := globalsReached(out)
	for _, g := range m.Globals {
		if used[sanitizeIdent(g.Name)] || used[g.Name] {
			out.Globals = append(out.Globals, g)
		}
	}
	return out
}

// calleesOf lists the function names f calls.
func calleesOf(f *ir.Func) []string {
	seen := map[string]bool{}
	f.EnsureStmts()
	var instrs []ir.Instr
	collectInstrs(f.Stmts, &instrs)
	for _, ins := range instrs {
		if ins.Op == ir.OpCall && ins.Sym != "" {
			seen[baseCallSym(ins.Sym)] = true
		}
	}
	out := make([]string, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// globalsReached names the tables the kept functions bind.
func globalsReached(m *ir.Module) map[string]bool {
	out := map[string]bool{}
	for i := range m.Funcs {
		f := &m.Funcs[i]
		f.EnsureStmts()
		var instrs []ir.Instr
		collectInstrs(f.Stmts, &instrs)
		for _, ins := range instrs {
			if ins.Op == ir.OpMov && strings.HasPrefix(ins.Sym, "global:") {
				name := ins.Sym[len("global:"):]
				out[name] = true
				out[sanitizeIdent(name)] = true
			}
		}
	}
	return out
}
