// F-p2go-simd-minmax — règle : la boucle de recherche d'extremum canonique
// for ($i=0; $i<count($a); $i++) { if ($a[$i] > $m) { $m = $a[$i]; } }
// devient MinMaxLoop (m = max(m, éléments)), émis via l'helper dual
// (scalaire / archsimd Greater+IfElse — VPMAXSQ est AVX-512, émulation AVX2).
package rules

import "code.hazyhaar.fr/devhoros/c2simd/p2go/ir"

func applySimdMinMax(f *ir.Func) {
	rewriteLoops(f, &f.Body, func(s *ir.For) ir.Stmt {
		i, arr, ok := matchHeader(s)
		if !ok || len(s.Body) != 1 {
			return nil
		}
		iff, ok := s.Body[0].(*ir.If)
		if !ok || len(iff.Else) != 0 || len(iff.Then) != 1 {
			return nil
		}
		cond, ok := iff.Cond.(*ir.Bin)
		if !ok || (cond.Op != ">" && cond.Op != "<") {
			return nil
		}
		ms, ok := cond.R.(*ir.Slot)
		if !ok || !isIndex(cond.L, arr, i) {
			return nil
		}
		upd, ok := iff.Then[0].(*ir.Assign)
		if !ok || upd.Slot != ms.Index || upd.Slot == i {
			return nil
		}
		if !isIndex(upd.Expr, arr, i) {
			return nil
		}
		if !counterIsLocal(f, s, i) {
			return nil
		}
		return &ir.MinMaxLoop{Dst: ms.Index, Arr: arr, IsMax: cond.Op == ">"}
	})
}
