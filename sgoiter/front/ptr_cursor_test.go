package front

import (
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/ir"
)

func TestPtrCursorStarAndPlusPlus(t *testing.T) {
	src := `
typedef unsigned char uint8_t;
typedef unsigned long size_t;
void poly_align(uint8_t *message, size_t n, uint8_t *out) {
	size_t i;
	for (i = 0; i < n; i++) {
		out[i] = *message;
		message++;
	}
}
`
	m, err := Parse(src, "t")
	if err != nil {
		t.Fatal(err)
	}
	var fn *ir.Func
	for i := range m.Funcs {
		if m.Funcs[i].Name == "poly_align" {
			fn = &m.Funcs[i]
			break
		}
	}
	if fn == nil {
		t.Fatal("missing poly_align")
	}
	var loadsWithIdx, offBumps, badLoad int
	var walk func([]ir.Stmt)
	walk = func(ss []ir.Stmt) {
		for _, s := range ss {
			ins := s.Ins
			if ins.Op == ir.OpLoad && len(ins.Args) == 2 {
				loadsWithIdx++
			}
			if ins.Op == ir.OpLoad && len(ins.Args) == 1 {
				badLoad++
			}
			if ins.Sym == "offslot" && (ins.Op == ir.OpAdd || ins.Op == ir.OpSub) {
				offBumps++
			}
			for _, in := range s.ForPost {
				if in.Sym == "offslot" && (in.Op == ir.OpAdd || in.Op == ir.OpSub) {
					offBumps++
				}
			}
			walk(s.ForBody)
			walk(s.ThenBody)
			walk(s.ElseBody)
		}
	}
	walk(fn.Stmts)
	if badLoad > 0 {
		t.Fatalf("single-arg Load count=%d", badLoad)
	}
	if loadsWithIdx == 0 {
		t.Fatal("expected Load(base, offSlot)")
	}
	if offBumps == 0 {
		t.Fatal("expected offslot bump on message++")
	}
}
