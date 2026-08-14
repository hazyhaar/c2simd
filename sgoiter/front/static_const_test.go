package front_test

import (
	"strings"
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/front"
	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/ir"
)

func TestStaticConstArrayUsesGlobal(t *testing.T) {
	src := `
#include <stdint.h>
static void mod_like(uint8_t out[4], const uint32_t x[4]) {
	static const uint32_t r[2] = {0x0a2c131b, 0xed9ce5a3};
	uint32_t acc = 0;
	for (int i = 0; i < 2; i++) acc += r[i] + x[i];
	out[0] = (uint8_t)acc;
}
void entry(uint8_t out[4], const uint32_t x[4]) { mod_like(out, x); }
`
	res, err := front.ParsePartial(src, "t")
	if err != nil {
		t.Fatal(err)
	}
	var mod *ir.Func
	for i := range res.Module.Funcs {
		if res.Module.Funcs[i].Name == "mod_like" {
			mod = &res.Module.Funcs[i]
			break
		}
	}
	if mod == nil {
		t.Fatal("no mod_like")
	}
	var nAlloca, nGlobal int
	var loadBases []int
	walk := func(ins ir.Instr) {
		if ins.Op == ir.OpAlloca {
			nAlloca++
			t.Logf("alloca dst=%d imm=%d", ins.Dst, ins.Imm)
		}
		if ins.Op == ir.OpMov && strings.HasPrefix(ins.Sym, "global:") {
			nGlobal++
			t.Logf("global bind dst=%d %s", ins.Dst, ins.Sym)
		}
		if ins.Op == ir.OpLoad && len(ins.Args) >= 1 {
			loadBases = append(loadBases, int(ins.Args[0]))
		}
	}
	var walkStmts func([]ir.Stmt)
	walkStmts = func(ss []ir.Stmt) {
		for _, s := range ss {
			walk(s.Ins)
			walkStmts(s.ForBody)
			walkStmts(s.ThenBody)
			walkStmts(s.ElseBody)
		}
	}
	walkStmts(mod.Stmts)
	for _, ins := range mod.Body {
		walk(ins)
	}
	// Body is walked twice (stmts+body); expect one logical pair.
	if nGlobal < 1 {
		t.Fatalf("want ≥1 global bind got %d", nGlobal)
	}
	// Emit aliases shadowing Alloca to package global; IR may still carry Alloca.
	t.Log("loadBases", loadBases, "alloca", nAlloca, "global", nGlobal)
}
