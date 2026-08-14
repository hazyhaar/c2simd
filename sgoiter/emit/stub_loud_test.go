package emit

import (
	"strings"
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/ir"
)

// A stub stands in for a C symbol the front could not harvest. It must never
// look like a result: Par25519 read a 32-byte buffer that the Pack25519 stub
// left untouched, and returned 0 without a word.
func TestStubBodyPanics(t *testing.T) {
	m := &ir.Module{Name: "t", Funcs: []ir.Func{{
		Name:   "caller",
		Result: ir.TypVoid,
		NVals:  1,
		Stmts: []ir.Stmt{
			{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpCall, Sym: "pack25519", Dst: ir.NoVal}},
			{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpReturn}},
		},
	}}}
	stubbed := FillStubs(m)
	if len(stubbed) != 1 || stubbed[0] != "Pack25519" {
		t.Fatalf("FillStubs reported %v, want [Pack25519]", stubbed)
	}
	src, err := Emit(m, ProfileGo127)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(src, "func Pack25519(args ...any)") {
		t.Fatalf("stub signature missing:\n%s", src)
	}
	if !strings.Contains(src, "panic(") {
		t.Errorf("stub body does not panic — a partial harvest must fail loudly:\n%s", src)
	}
	if strings.Contains(src, "func Pack25519(args ...any) {\n\treturn\n}") {
		t.Error("stub still returns silently")
	}
}

func TestIsStubOnlyMatchesStubs(t *testing.T) {
	stub := ir.Func{Name: "X", Params: []ir.Param{{Name: "args", Type: StubParamType}}}
	if !IsStub(&stub) {
		t.Error("stub not recognised")
	}
	real := ir.Func{Name: "Y", Params: []ir.Param{{Name: "x", Type: ir.TypUint64}}}
	if IsStub(&real) {
		t.Error("a real function was taken for a stub")
	}
}
