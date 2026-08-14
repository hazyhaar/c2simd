package rules_test

import (
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/ir"
	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/rules"
)

func witnessAndOnes() *ir.Module {
	return &ir.Module{Name: "w", Funcs: []ir.Func{{
		Name: "f", Result: ir.TypInt, NVals: 4,
		Body: []ir.Instr{
			{Op: ir.OpConst, Dst: 0, Imm: 42},
			{Op: ir.OpConst, Dst: 1, Imm: -1},
			{Op: ir.OpAnd, Dst: 2, Args: []ir.Value{0, 1}},
			{Op: ir.OpReturn, Args: []ir.Value{2}},
		},
	}}}
}

func witnessMovElim() *ir.Module {
	return &ir.Module{Name: "w", Funcs: []ir.Func{{
		Name: "f", Result: ir.TypInt, NVals: 2,
		Body: []ir.Instr{
			{Op: ir.OpConst, Dst: 0, Imm: 1},
			{Op: ir.OpMov, Dst: 0, Args: []ir.Value{0}},
			{Op: ir.OpReturn, Args: []ir.Value{0}},
		},
	}}}
}

func TestTableHasFiveRewrites(t *testing.T) {
	n := 0
	for _, d := range rules.Table {
		if d.Kind == rules.KindRewrite {
			n++
		}
	}
	if n < 5 {
		t.Fatalf("rewrites=%d want ≥5", n)
	}
}

// witness builds a two-operand module: v0 op v1 with the given constants, or a
// single register used twice when args are equal.
func witness(op ir.Op, imm0, imm1 int64, sameArg bool) *ir.Module {
	args := []ir.Value{0, 1}
	body := []ir.Instr{
		{Op: ir.OpConst, Dst: 0, Imm: imm0},
		{Op: ir.OpConst, Dst: 1, Imm: imm1},
	}
	if sameArg {
		args = []ir.Value{0, 0}
		body = []ir.Instr{{Op: ir.OpMov, Dst: 0, Args: []ir.Value{2}}}
	}
	body = append(body, ir.Instr{Op: op, Dst: 3, Args: args}, ir.Instr{Op: ir.OpReturn, Args: []ir.Value{3}})
	return &ir.Module{Name: "w", Funcs: []ir.Func{{Name: "f", Result: ir.TypInt, NVals: 4, Body: body}}}
}

// Every rewrite rule must actually rewrite something. Counting table entries let
// two no-op rules satisfy the thesaurus gate while transforming nothing.
func TestEveryRewriteRuleTransformsItsWitness(t *testing.T) {
	witnesses := map[string]*ir.Module{
		"const_fold_add":             witness(ir.OpAdd, 2, 3, false),
		"const_fold_sub":             witness(ir.OpSub, 7, 3, false),
		"const_fold_mul":             witness(ir.OpMul, 2, 3, false),
		"const_fold_mul0":            witness(ir.OpMul, 5, 0, false),
		"strength_mul1":              witness(ir.OpMul, 5, 1, false),
		"add_zero":                   witness(ir.OpAdd, 5, 0, false),
		"sub_zero":                   witness(ir.OpSub, 5, 0, false),
		"xor_self":                   witness(ir.OpXor, 0, 0, true),
		"and_or_self":                witness(ir.OpAnd, 0, 0, true),
		"or_self":                    witness(ir.OpOr, 0, 0, true),
		"xor_zero":                   witness(ir.OpXor, 5, 0, false),
		"or_zero":                    witness(ir.OpOr, 5, 0, false),
		"and_ones_u64":               witnessAndOnes(),
		"shl_zero":                   witness(ir.OpShl, 5, 0, false),
		"shr_zero":                   witness(ir.OpShr, 5, 0, false),
		"mov_elim":                   witnessMovElim(),
		"loop_neg_count_to_forward":  rules.WitnessLoopNeg(),
		"const_prop_binop":           rules.WitnessConstPropBinop(),
		"fold_load_global_const_idx": rules.WitnessFoldGlobalLoad(),
		"dce_if_const_eq":            rules.WitnessDceIfConstEq(),
		"mark_const_call_args":       rules.WitnessMarkConstCall(),
		"murmur_neg_loop_rewrite":    rules.WitnessMurmurNegLoop(),
		"unroll_const_trip_load":     rules.WitnessUnrollConstTrip(),
	}
	// Note: const_prop_binop may equal const_fold_add on same witness — both OK.
	for _, d := range rules.Table {
		if d.Kind != rules.KindRewrite {
			continue
		}
		m, ok := witnesses[d.ID]
		if !ok {
			t.Errorf("rule %q has no witness module: add one, or drop the rule", d.ID)
			continue
		}
		if d.Apply == nil {
			t.Errorf("rule %q has no Apply", d.ID)
			continue
		}
		out, err := d.Apply(m)
		if err != nil {
			t.Errorf("rule %q: %v", d.ID, err)
			continue
		}
		eq, err := ir.EqualJSON(m, out)
		if err != nil {
			t.Fatal(err)
		}
		if eq {
			t.Errorf("rule %q left its witness unchanged — it rewrites nothing", d.ID)
		}
	}
}

func TestNoGenericSIMD(t *testing.T) {
	if rules.HasGenericSIMDRule() {
		t.Fatal("generic simd rule present")
	}
}

func TestConstFoldAdd(t *testing.T) {
	// v0=2, v1=3, v2=v0+v1 → should become const 5
	m := &ir.Module{Name: "t", Funcs: []ir.Func{{
		Name:   "f",
		Result: ir.TypInt,
		NVals:  3,
		Body: []ir.Instr{
			{Op: ir.OpConst, Dst: 0, Imm: 2},
			{Op: ir.OpConst, Dst: 1, Imm: 3},
			{Op: ir.OpAdd, Dst: 2, Args: []ir.Value{0, 1}},
			{Op: ir.OpReturn, Args: []ir.Value{2}},
		},
	}}}
	out, err := rules.ApplyAll(m)
	if err != nil {
		t.Fatal(err)
	}
	// find dst 2
	var found bool
	for _, ins := range out.Funcs[0].Body {
		if ins.Dst == 2 && ins.Op == ir.OpConst && ins.Imm == 5 {
			found = true
		}
	}
	if !found {
		t.Fatalf("fold failed: %+v", out.Funcs[0].Body)
	}
}

func TestXorSelf(t *testing.T) {
	m := &ir.Module{Name: "t", Funcs: []ir.Func{{
		Name: "f", Result: ir.TypInt, NVals: 2,
		Params: []ir.Param{{Name: "x", Type: ir.TypInt, Reg: 0}},
		Body: []ir.Instr{
			{Op: ir.OpXor, Dst: 1, Args: []ir.Value{0, 0}},
			{Op: ir.OpReturn, Args: []ir.Value{1}},
		},
	}}}
	out, err := rules.ApplyAll(m)
	if err != nil {
		t.Fatal(err)
	}
	for _, ins := range out.Funcs[0].Body {
		if ins.Dst == 1 {
			if ins.Op != ir.OpConst || ins.Imm != 0 {
				t.Fatalf("got %+v", ins)
			}
		}
	}
}

func TestMul0(t *testing.T) {
	m := &ir.Module{Name: "t", Funcs: []ir.Func{{
		Name: "f", Result: ir.TypInt, NVals: 3,
		Params: []ir.Param{{Name: "x", Type: ir.TypInt, Reg: 0}},
		Body: []ir.Instr{
			{Op: ir.OpConst, Dst: 1, Imm: 0},
			{Op: ir.OpMul, Dst: 2, Args: []ir.Value{0, 1}},
			{Op: ir.OpReturn, Args: []ir.Value{2}},
		},
	}}}
	out, err := rules.ApplyAll(m)
	if err != nil {
		t.Fatal(err)
	}
	for _, ins := range out.Funcs[0].Body {
		if ins.Dst == 2 && (ins.Op != ir.OpConst || ins.Imm != 0) {
			t.Fatalf("%+v", ins)
		}
	}
}
