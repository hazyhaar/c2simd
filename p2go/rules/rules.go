// Package rules — passe 4 de p2go : réécritures pattern-matching sur l'IR.
// v0.1 : pliage de constantes arithmétiques (Bin/Neg sur Const), division et
// modulo par zéro laissés en place (l'erreur runtime Go est la sémantique).
package rules

import "code.hazyhaar.fr/devhoros/c2simd/p2go/ir"

// Apply réécrit le programme en place et le retourne : const-fold puis
// reconnaissance des boucles de réduction vectorisables (simd_sum.go).
func Apply(prog *ir.Program) *ir.Program {
	for _, f := range prog.Funcs {
		rewriteStmts(f.Body)
		applySimdDot(f) // avant sum : corps plus spécifique
		applySimdSum(f)
		applySimdMinMax(f)
	}
	rewriteStmts(prog.Main.Body)
	applySimdDot(prog.Main)
	applySimdSum(prog.Main)
	applySimdMinMax(prog.Main)
	return prog
}

func rewriteStmts(body []ir.Stmt) {
	for _, st := range body {
		switch s := st.(type) {
		case *ir.Assign:
			s.Expr = foldExpr(s.Expr)
		case *ir.Echo:
			for i, a := range s.Args {
				s.Args[i] = foldExpr(a)
			}
		case *ir.If:
			s.Cond = foldExpr(s.Cond)
			rewriteStmts(s.Then)
			rewriteStmts(s.Else)
		case *ir.While:
			if s.Cond != nil {
				s.Cond = foldExpr(s.Cond)
			}
			rewriteStmts(s.Body)
		case *ir.For:
			if s.Init != nil {
				rewriteStmts([]ir.Stmt{s.Init})
			}
			if s.Cond != nil {
				s.Cond = foldExpr(s.Cond)
			}
			if s.Post != nil {
				rewriteStmts([]ir.Stmt{s.Post})
			}
			rewriteStmts(s.Body)
		case *ir.Return:
			if s.Expr != nil {
				s.Expr = foldExpr(s.Expr)
			}
		case *ir.CallStmt:
			foldCall(s.Call)
		case *ir.ArrAssign:
			for i, el := range s.Elems {
				s.Elems[i] = foldExpr(el)
			}
		case *ir.IndexAssign:
			s.Idx = foldExpr(s.Idx)
			s.Expr = foldExpr(s.Expr)
		case *ir.ArrPush:
			s.Val = foldExpr(s.Val)
		case *ir.Switch:
			s.Subject = foldExpr(s.Subject)
			for i := range s.Cases {
				for j, v := range s.Cases[i].Vals {
					s.Cases[i].Vals[j] = foldExpr(v)
				}
				rewriteStmts(s.Cases[i].Body)
			}
			rewriteStmts(s.Default)
		}
	}
}

func foldCall(c *ir.Call) {
	for i, a := range c.Args {
		c.Args[i] = foldExpr(a)
	}
}

func foldExpr(e ir.Expr) ir.Expr {
	switch x := e.(type) {
	case *ir.Neg:
		x.X = foldExpr(x.X)
		if c, ok := x.X.(*ir.Const); ok {
			return &ir.Const{Value: -c.Value}
		}
	case *ir.Not:
		x.X = foldExpr(x.X)
	case *ir.BitNot:
		x.X = foldExpr(x.X)
		if c, ok := x.X.(*ir.Const); ok {
			return &ir.Const{Value: ^c.Value}
		}
	case *ir.Logic:
		x.L = foldExpr(x.L)
		x.R = foldExpr(x.R)
	case *ir.Bin:
		x.L = foldExpr(x.L)
		x.R = foldExpr(x.R)
		l, lok := x.L.(*ir.Const)
		r, rok := x.R.(*ir.Const)
		if lok && rok {
			if v, ok := evalBin(x.Op, l.Value, r.Value); ok {
				return &ir.Const{Value: v}
			}
		}
	case *ir.Call:
		foldCall(x)
	case *ir.Index:
		x.Idx = foldExpr(x.Idx)
	case *ir.ItoS:
		x.X = foldExpr(x.X)
	case *ir.StrLen:
		x.X = foldExpr(x.X)
	case *ir.ArrCopy:
		x.X = foldExpr(x.X)
	case *ir.ArrLit:
		for i, el := range x.Elems {
			x.Elems[i] = foldExpr(el)
		}
	case *ir.Builtin:
		for i, a := range x.Args {
			x.Args[i] = foldExpr(a)
		}
	}
	return e
}

// evalBin plie les opérateurs arithmétiques ; les comparaisons produisent des
// bool en Go et ne se plient pas en Const int (elles restent en place).
func evalBin(op string, l, r int64) (int64, bool) {
	switch op {
	case "+":
		return l + r, true
	case "-":
		return l - r, true
	case "*":
		return l * r, true
	case "/":
		if r == 0 {
			return 0, false
		}
		return l / r, true
	case "%":
		if r == 0 {
			return 0, false
		}
		return l % r, true
	case "&":
		return l & r, true
	case "|":
		return l | r, true
	case "^":
		return l ^ r, true
	case "<<":
		if r < 0 || r > 63 {
			return 0, false
		}
		return l << uint(r), true
	case ">>":
		if r < 0 || r > 63 {
			return 0, false
		}
		return l >> uint(r), true
	}
	return 0, false
}
