package rules

import (
	"strconv"
	"strings"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/ir"
)

func init() {
	Table = append(Table,
		Def{ID: "const_prop_binop", Kind: KindRewrite,
			Summary: "fold binop when both args resolve to single-assign consts",
			Apply:   constPropBinop},
		Def{ID: "fold_load_global_const_idx", Kind: KindRewrite,
			Summary: "load global[const] → const when InitCSV present",
			Apply:   foldLoadGlobalConstIdx},
	)
}

func constPropBinop(m *ir.Module) (*ir.Module, error) {
	return mapInstr(m, func(ins ir.Instr, f *ir.Func) (ir.Instr, bool) {
		switch ins.Op {
		case ir.OpAdd, ir.OpSub, ir.OpMul, ir.OpAnd, ir.OpOr, ir.OpXor, ir.OpShl, ir.OpShr:
		default:
			return ins, false
		}
		if len(ins.Args) != 2 {
			return ins, false
		}
		a, oka := constOf(f, ins.Args[0])
		b, okb := constOf(f, ins.Args[1])
		if !oka || !okb {
			return ins, false
		}
		var r int64
		switch ins.Op {
		case ir.OpAdd:
			r = a + b
		case ir.OpSub:
			r = a - b
		case ir.OpMul:
			r = a * b
		case ir.OpAnd:
			r = a & b
		case ir.OpOr:
			r = a | b
		case ir.OpXor:
			r = a ^ b
		case ir.OpShl:
			if b < 0 || b > 63 {
				return ins, false
			}
			r = a << uint64(b)
		case ir.OpShr:
			if b < 0 || b > 63 {
				return ins, false
			}
			r = int64(uint64(a) >> uint64(b))
		}
		return ir.Instr{Op: ir.OpConst, Dst: ins.Dst, Imm: r}, true
	})
}

func foldLoadGlobalConstIdx(m *ir.Module) (*ir.Module, error) {
	if m == nil {
		return nil, nil
	}
	tables := map[string][]int64{}
	for _, g := range m.Globals {
		if g.InitCSV == "" {
			continue
		}
		vals, ok := parseCSVImm(g.InitCSV)
		if ok && len(vals) > 0 {
			tables[g.Name] = vals
		}
	}
	if len(tables) == 0 {
		return m, nil
	}
	return mapInstr(m, func(ins ir.Instr, f *ir.Func) (ir.Instr, bool) {
		if ins.Op != ir.OpLoad || len(ins.Args) != 2 {
			return ins, false
		}
		gname := ""
		for _, x := range f.Body {
			if x.Dst == ins.Args[0] && x.Op == ir.OpMov && strings.HasPrefix(x.Sym, "global:") {
				gname = x.Sym[len("global:"):]
				break
			}
		}
		if gname == "" {
			return ins, false
		}
		tab, ok := tables[gname]
		if !ok {
			return ins, false
		}
		idx, ok := constOf(f, ins.Args[1])
		if !ok || idx < 0 || int(idx) >= len(tab) {
			return ins, false
		}
		return ir.Instr{Op: ir.OpConst, Dst: ins.Dst, Imm: tab[idx]}, true
	})
}

func parseCSVImm(s string) ([]int64, bool) {
	parts := strings.Split(s, ",")
	var out []int64
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// strip ULL/UL suffixes
		for strings.HasSuffix(p, "U") || strings.HasSuffix(p, "L") || strings.HasSuffix(p, "u") || strings.HasSuffix(p, "l") {
			p = p[:len(p)-1]
		}
		var v int64
		var err error
		if strings.HasPrefix(p, "0x") || strings.HasPrefix(p, "0X") {
			var u uint64
			u, err = strconv.ParseUint(p[2:], 16, 64)
			v = int64(u)
		} else {
			v, err = strconv.ParseInt(p, 10, 64)
		}
		if err != nil {
			return nil, false
		}
		out = append(out, v)
	}
	return out, true
}

// WitnessConstPropBinop for rules_test.
func WitnessConstPropBinop() *ir.Module {
	return &ir.Module{Name: "w", Funcs: []ir.Func{{
		Name: "f", Result: ir.TypInt, NVals: 4,
		Body: []ir.Instr{
			{Op: ir.OpConst, Dst: 0, Imm: 10},
			{Op: ir.OpConst, Dst: 1, Imm: 3},
			{Op: ir.OpAdd, Dst: 2, Args: []ir.Value{0, 1}},
			{Op: ir.OpReturn, Args: []ir.Value{2}},
		},
	}}}
}

// WitnessFoldGlobalLoad for rules_test.
func WitnessFoldGlobalLoad() *ir.Module {
	return &ir.Module{
		Name: "w",
		Globals: []ir.Global{{
			Name: "T", Type: ir.TypUint8, InitCSV: "10,20,30,40",
		}},
		Funcs: []ir.Func{{
			Name: "f", Result: ir.TypInt, NVals: 4,
			Body: []ir.Instr{
				{Op: ir.OpMov, Dst: 0, Sym: "global:T"},
				{Op: ir.OpConst, Dst: 1, Imm: 2},
				{Op: ir.OpLoad, Dst: 2, Args: []ir.Value{0, 1}, Elem: ir.TypUint8},
				{Op: ir.OpReturn, Args: []ir.Value{2}},
			},
		}},
	}
}
