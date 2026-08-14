package emit

import (
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/ir"
)

// closureModule: entry calls helper, helper reads a table; orphan is reachable
// from nothing.
func closureModule() *ir.Module {
	return &ir.Module{
		Name:    "m",
		Globals: []ir.Global{{Name: "used", Type: ir.TypUint8, InitCSV: "1,2"}, {Name: "unused", Type: ir.TypUint8, InitCSV: "3"}},
		Funcs: []ir.Func{
			{
				Name: "entry", Result: ir.TypVoid, NVals: 1,
				Stmts: []ir.Stmt{
					{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpCall, Sym: "helper", Dst: ir.NoVal}},
					{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpReturn}},
				},
			},
			{
				Name: "helper", Result: ir.TypVoid, NVals: 2, Static: true,
				Stmts: []ir.Stmt{
					{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpMov, Dst: 0, Sym: "global:used"}},
					{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpReturn}},
				},
			},
			{
				Name: "orphan", Result: ir.TypVoid, NVals: 1, Static: true,
				Stmts: []ir.Stmt{
					{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpMov, Dst: 0, Sym: "global:unused"}},
					{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpReturn}},
				},
			},
		},
	}
}

func names(m *ir.Module) []string {
	out := []string{}
	for _, f := range m.Funcs {
		out = append(out, f.Name)
	}
	return out
}

func TestRootClosureKeepsCalleesDropsOrphans(t *testing.T) {
	out := RootClosure(closureModule(), nil)
	got := names(out)
	if len(got) != 2 || got[0] != "entry" || got[1] != "helper" {
		t.Errorf("kept %v, want [entry helper]", got)
	}
	if len(out.Globals) != 1 || out.Globals[0].Name != "used" {
		t.Errorf("globals kept: %v, want only used", out.Globals)
	}
}

// A harness often exercises a function C declares static — tweetnacl's vn is one.
func TestRootClosureAcceptsNamedRoots(t *testing.T) {
	out := RootClosure(closureModule(), []string{"orphan"})
	got := names(out)
	if len(got) != 1 || got[0] != "orphan" {
		t.Errorf("kept %v, want [orphan]", got)
	}
}

// A module where nothing matches keeps everything: emitting an empty file would
// be worse than emitting too much.
func TestRootClosureKeepsAllWhenNothingMatches(t *testing.T) {
	out := RootClosure(closureModule(), []string{"absent"})
	if len(out.Funcs) != 3 {
		t.Errorf("kept %v, want the whole module", names(out))
	}
}
