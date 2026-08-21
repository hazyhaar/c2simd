// F-p2go-simd-sum-reduction — règle pattern-matching : une boucle de réduction
// canonique for ($i=0; $i<count($a); $i++) { $s += $a[$i]; } est remplacée par
// le nœud IR SumLoop (acc += somme(arr)), émis via l'helper dual
// scalaire / simd/archsimd. Garde de sûreté : le compteur ne doit être lu
// nulle part ailleurs dans la fonction (sa valeur finale n'est pas matérialisée).
package rules

import "code.hazyhaar.fr/devhoros/c2simd/p2go/ir"

func applySimdSum(f *ir.Func) {
	rewriteSumLoops(f, &f.Body)
}

func rewriteSumLoops(f *ir.Func, body *[]ir.Stmt) {
	for idx, st := range *body {
		switch s := st.(type) {
		case *ir.For:
			if sl, counter := matchSumLoop(s); sl != nil && slotReads(f.Body, counter) == slotReads([]ir.Stmt{s}, counter) {
				(*body)[idx] = sl
				continue
			}
			rewriteSumLoops(f, &s.Body)
		case *ir.If:
			rewriteSumLoops(f, &s.Then)
			rewriteSumLoops(f, &s.Else)
		case *ir.While:
			rewriteSumLoops(f, &s.Body)
		case *ir.Switch:
			for i := range s.Cases {
				rewriteSumLoops(f, &s.Cases[i].Body)
			}
			rewriteSumLoops(f, &s.Default)
		}
	}
}

// matchSumLoop reconnaît la forme canonique et retourne (SumLoop, slot compteur).
func matchSumLoop(s *ir.For) (*ir.SumLoop, int) {
	init, ok := s.Init.(*ir.Assign)
	if !ok {
		return nil, 0
	}
	zero, ok := init.Expr.(*ir.Const)
	if !ok || zero.Value != 0 {
		return nil, 0
	}
	i := init.Slot

	cond, ok := s.Cond.(*ir.Bin)
	if !ok || cond.Op != "<" {
		return nil, 0
	}
	cl, ok := cond.L.(*ir.Slot)
	if !ok || cl.Index != i {
		return nil, 0
	}
	cnt, ok := cond.R.(*ir.Count)
	if !ok {
		return nil, 0
	}

	post, ok := s.Post.(*ir.Assign)
	if !ok || post.Slot != i {
		return nil, 0
	}
	pb, ok := post.Expr.(*ir.Bin)
	if !ok || pb.Op != "+" {
		return nil, 0
	}
	pl, lok := pb.L.(*ir.Slot)
	pr, rok := pb.R.(*ir.Const)
	if !lok || !rok || pl.Index != i || pr.Value != 1 {
		return nil, 0
	}

	if len(s.Body) != 1 {
		return nil, 0
	}
	acc, ok := s.Body[0].(*ir.Assign)
	if !ok || acc.Slot == i {
		return nil, 0
	}
	ab, ok := acc.Expr.(*ir.Bin)
	if !ok || ab.Op != "+" {
		return nil, 0
	}
	al, ok := ab.L.(*ir.Slot)
	if !ok || al.Index != acc.Slot {
		return nil, 0
	}
	ix, ok := ab.R.(*ir.Index)
	if !ok || ix.Slot != cnt.Slot {
		return nil, 0
	}
	iv, ok := ix.Idx.(*ir.Slot)
	if !ok || iv.Index != i {
		return nil, 0
	}
	return &ir.SumLoop{Acc: acc.Slot, Arr: cnt.Slot}, i
}

// slotReads compte les lectures du slot dans un sous-arbre IR.
func slotReads(body []ir.Stmt, slot int) int {
	n := 0
	var walkE func(ir.Expr)
	walkE = func(x ir.Expr) {
		switch v := x.(type) {
		case *ir.Slot:
			if v.Index == slot {
				n++
			}
		case *ir.Bin:
			walkE(v.L)
			walkE(v.R)
		case *ir.Logic:
			walkE(v.L)
			walkE(v.R)
		case *ir.Not:
			walkE(v.X)
		case *ir.Neg:
			walkE(v.X)
		case *ir.BitNot:
			walkE(v.X)
		case *ir.ItoS:
			walkE(v.X)
		case *ir.StrLen:
			walkE(v.X)
		case *ir.ArrCopy:
			walkE(v.X)
		case *ir.ArrLit:
			for _, el := range v.Elems {
				walkE(el)
			}
		case *ir.Builtin:
			for _, a := range v.Args {
				walkE(a)
			}
		case *ir.Index:
			walkE(v.Idx)
		case *ir.Call:
			for _, a := range v.Args {
				walkE(a)
			}
		}
	}
	var walkS func([]ir.Stmt)
	walkS = func(stmts []ir.Stmt) {
		for _, st := range stmts {
			switch s := st.(type) {
			case *ir.Assign:
				walkE(s.Expr)
			case *ir.ArrAssign:
				for _, el := range s.Elems {
					walkE(el)
				}
			case *ir.IndexAssign:
				walkE(s.Idx)
				walkE(s.Expr)
			case *ir.Echo:
				for _, a := range s.Args {
					walkE(a)
				}
			case *ir.If:
				walkE(s.Cond)
				walkS(s.Then)
				walkS(s.Else)
			case *ir.While:
				walkE(s.Cond)
				walkS(s.Body)
			case *ir.For:
				if s.Init != nil {
					walkS([]ir.Stmt{s.Init})
				}
				if s.Cond != nil {
					walkE(s.Cond)
				}
				if s.Post != nil {
					walkS([]ir.Stmt{s.Post})
				}
				walkS(s.Body)
			case *ir.Return:
				if s.Expr != nil {
					walkE(s.Expr)
				}
			case *ir.CallStmt:
				for _, a := range s.Call.Args {
					walkE(a)
				}
			case *ir.SumLoop:
				if s.Acc == slot || s.Arr == slot {
					n++
				}
			case *ir.DotLoop:
				if s.Acc == slot || s.A == slot || s.B == slot {
					n++
				}
			case *ir.MinMaxLoop:
				if s.Dst == slot || s.Arr == slot {
					n++
				}
			case *ir.ArrPush:
				if s.Slot == slot {
					n++
				}
				walkE(s.Val)
			case *ir.ArrPop:
				if s.Arr == slot || s.Dst == slot {
					n++
				}
			case *ir.Switch:
				walkE(s.Subject)
				for _, c := range s.Cases {
					for _, v := range c.Vals {
						walkE(v)
					}
					walkS(c.Body)
				}
				walkS(s.Default)
			}
		}
	}
	walkS(body)
	return n
}
