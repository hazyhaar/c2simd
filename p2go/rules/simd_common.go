// Socle commun des règles SIMD : reconnaissance de l'en-tête de boucle
// canonique for (i=0; i<count(arr); i++) et garde de sûreté sur le compteur.
package rules

import "code.hazyhaar.fr/devhoros/c2simd/p2go/ir"

// matchHeader reconnaît l'en-tête canonique et retourne (compteur, tableau).
func matchHeader(s *ir.For) (counter, arr int, ok bool) {
	init, isA := s.Init.(*ir.Assign)
	if !isA {
		return 0, 0, false
	}
	zero, isC := init.Expr.(*ir.Const)
	if !isC || zero.Value != 0 {
		return 0, 0, false
	}
	i := init.Slot

	cond, isB := s.Cond.(*ir.Bin)
	if !isB || cond.Op != "<" {
		return 0, 0, false
	}
	cl, isS := cond.L.(*ir.Slot)
	if !isS || cl.Index != i {
		return 0, 0, false
	}
	cnt, isCnt := cond.R.(*ir.Count)
	if !isCnt {
		return 0, 0, false
	}

	post, isP := s.Post.(*ir.Assign)
	if !isP || post.Slot != i {
		return 0, 0, false
	}
	pb, isPB := post.Expr.(*ir.Bin)
	if !isPB || pb.Op != "+" {
		return 0, 0, false
	}
	pl, lok := pb.L.(*ir.Slot)
	pr, rok := pb.R.(*ir.Const)
	if !lok || !rok || pl.Index != i || pr.Value != 1 {
		return 0, 0, false
	}
	return i, cnt.Slot, true
}

// counterIsLocal vérifie que le compteur n'est lu nulle part ailleurs dans la
// fonction que dans la boucle candidate (sa valeur finale n'est pas matérialisée).
func counterIsLocal(f *ir.Func, loop ir.Stmt, counter int) bool {
	return slotReads(f.Body, counter) == slotReads([]ir.Stmt{loop}, counter)
}

// isIndex reconnaît arr[i].
func isIndex(e ir.Expr, arr, i int) bool {
	ix, ok := e.(*ir.Index)
	if !ok || ix.Slot != arr {
		return false
	}
	iv, ok := ix.Idx.(*ir.Slot)
	return ok && iv.Index == i
}
