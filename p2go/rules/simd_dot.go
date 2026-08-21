// F-p2go-simd-dot — règle : la boucle de produit scalaire canonique
// for ($i=0; $i<count($a); $i++) { $s += $a[$i] * $b[$i]; } devient le nœud
// DotLoop, émis via l'helper dual (scalaire / archsimd AVX2 avec multiply
// 64-bit émulé par VPMULUDQ — pas de VPMULLQ hors AVX-512).
package rules

import "code.hazyhaar.fr/devhoros/c2simd/p2go/ir"

func applySimdDot(f *ir.Func) {
	rewriteLoops(f, &f.Body, func(s *ir.For) ir.Stmt {
		i, arrA, ok := matchHeader(s)
		if !ok || len(s.Body) != 1 {
			return nil
		}
		acc, ok := s.Body[0].(*ir.Assign)
		if !ok || acc.Slot == i {
			return nil
		}
		add, ok := acc.Expr.(*ir.Bin)
		if !ok || add.Op != "+" {
			return nil
		}
		al, ok := add.L.(*ir.Slot)
		if !ok || al.Index != acc.Slot {
			return nil
		}
		mul, ok := add.R.(*ir.Bin)
		if !ok || mul.Op != "*" {
			return nil
		}
		// a[i] * b[i], a étant le tableau de l'en-tête (count)
		var arrB int
		switch {
		case isIndex(mul.L, arrA, i) && isIndexAny(mul.R, i):
			arrB = mul.R.(*ir.Index).Slot
		case isIndex(mul.R, arrA, i) && isIndexAny(mul.L, i):
			arrB = mul.L.(*ir.Index).Slot
		default:
			return nil
		}
		if !counterIsLocal(f, s, i) {
			return nil
		}
		return &ir.DotLoop{Acc: acc.Slot, A: arrA, B: arrB}
	})
}

// isIndexAny reconnaît x[i] quel que soit le tableau x.
func isIndexAny(e ir.Expr, i int) bool {
	ix, ok := e.(*ir.Index)
	if !ok {
		return false
	}
	iv, ok := ix.Idx.(*ir.Slot)
	return ok && iv.Index == i
}

// rewriteLoops applique try à chaque For du corps (récursif) ; un remplacement
// non-nil substitue le nœud.
func rewriteLoops(f *ir.Func, body *[]ir.Stmt, try func(*ir.For) ir.Stmt) {
	for idx, st := range *body {
		switch s := st.(type) {
		case *ir.For:
			if repl := try(s); repl != nil {
				(*body)[idx] = repl
				continue
			}
			rewriteLoops(f, &s.Body, try)
		case *ir.If:
			rewriteLoops(f, &s.Then, try)
			rewriteLoops(f, &s.Else, try)
		case *ir.While:
			rewriteLoops(f, &s.Body, try)
		case *ir.Switch:
			for i := range s.Cases {
				rewriteLoops(f, &s.Cases[i].Body, try)
			}
			rewriteLoops(f, &s.Default, try)
		}
	}
}
