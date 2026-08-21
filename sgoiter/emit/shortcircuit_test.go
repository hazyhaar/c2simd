package emit_test

import (
	"strings"
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/emit"
	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/ir"
)

// TestShortCircuitAndInIf tests that when a condition has `A && B` where B contains
// a Load operation, the load is emitted inside the guard for A (short-circuit).
func TestShortCircuitAndInIf(t *testing.T) {
	// Equivalent to:
	// if (idx < width && row[idx] != 0) { return 1; } return 0;
	// Dst 2: idx < width
	// Dst 3: row[idx] (OpLoad)
	// Dst 4: Dst 3 != 0
	// Dst 5: Dst 2 && Dst 4 (__cmp_&&)
	m := &ir.Module{
		Name: "testpkg",
		Funcs: []ir.Func{{
			Name:   "check_short_circuit",
			Result: ir.TypInt,
			Params: []ir.Param{
				{Name: "row", Type: ir.TypUint8, Reg: 0},
				{Name: "idx", Type: ir.TypInt, Reg: 1},
				{Name: "width", Type: ir.TypInt, Reg: 2},
			},
			NVals: 10,
			Stmts: []ir.Stmt{
				{
					Kind: ir.SKIf,
					ForInit: []ir.Instr{
						{Op: ir.OpCall, Dst: 3, Args: []ir.Value{1, 2}, Sym: "__cmp_<"},
						{Op: ir.OpLoad, Dst: 4, Args: []ir.Value{0, 1}, Elem: ir.TypUint8},
						{Op: ir.OpCall, Dst: 5, Args: []ir.Value{4, 6}, Sym: "__cmp_!="},
						{Op: ir.OpCall, Dst: 7, Args: []ir.Value{3, 5}, Sym: "__cmp_&&"},
					},
					CondLeft:  7,
					CondOp:    "truthy",
					ThenBody: []ir.Stmt{
						{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpReturn, Args: []ir.Value{8}}},
					},
				},
				{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpReturn, Args: []ir.Value{9}}},
			},
		}},
	}
	src, err := emit.Emit(m, emit.ProfileGo127)
	if err != nil {
		t.Fatalf("Emit failed: %v", err)
	}
	// The generated Go code must short-circuit the load:
	// It should nest the check or gate the load behind the first condition.
	lines := strings.Split(src, "\n")
	foundFirstIf := false
	loadInsideFirstIf := false
	for _, l := range lines {
		if strings.Contains(l, "if") && strings.Contains(l, "<") {
			foundFirstIf = true
		}
		if foundFirstIf && strings.Contains(l, "row[") {
			loadInsideFirstIf = true
			break
		}
	}
	if !loadInsideFirstIf {
		t.Fatalf("Load row[idx] was emitted before the boundary check:\n%s", src)
	}
}
