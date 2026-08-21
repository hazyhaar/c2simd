// Désucreur post-parse — F-p2go-ternary-subexpr, F-p2go-match : hoiste les
// Ternary et Match hors des expressions vers des temporaires (p2go_tN) posés
// AVANT le statement porteur, en préservant la paresse (les branches vivent
// dans les corps du If/Switch hoisté). Après cette passe, aucun Ternary ni
// Match n'atteint types/. Interdits gardés : sucre dans une condition de
// boucle (réévaluée par itération, le hoisting fausserait la sémantique) et
// dans les valeurs de case.
package front

import "fmt"

type desugarer struct{ n int }

func (d *desugarer) gensym() string {
	s := fmt.Sprintf("p2go_t%d", d.n)
	d.n++
	return s
}

func (d *desugarer) stmts(body []Stmt) ([]Stmt, *Err) {
	var out []Stmt
	for _, st := range body {
		pre, st2, err := d.stmt(st)
		if err != nil {
			return nil, err
		}
		out = append(out, pre...)
		out = append(out, st2)
	}
	return out, nil
}

func (d *desugarer) stmt(st Stmt) ([]Stmt, Stmt, *Err) {
	switch s := st.(type) {
	case *Assign:
		pre, e, err := d.extract(s.Expr)
		s.Expr = e
		return pre, s, err
	case *IndexAssign:
		p1, idx, err := d.extract(s.Idx)
		if err != nil {
			return nil, nil, err
		}
		p2, ex, err := d.extract(s.Expr)
		if err != nil {
			return nil, nil, err
		}
		s.Idx, s.Expr = idx, ex
		return append(p1, p2...), s, nil
	case *Echo:
		var pre []Stmt
		for i, a := range s.Args {
			pi, a2, err := d.extract(a)
			if err != nil {
				return nil, nil, err
			}
			pre = append(pre, pi...)
			s.Args[i] = a2
		}
		return pre, s, nil
	case *Return:
		if s.Expr == nil {
			return nil, s, nil
		}
		pre, e, err := d.extract(s.Expr)
		s.Expr = e
		return pre, s, err
	case *ExprStmt:
		pre, e, err := d.extract(s.Expr)
		s.Expr = e
		return pre, s, err
	case *If:
		pre, c, err := d.extract(s.Cond)
		if err != nil {
			return nil, nil, err
		}
		s.Cond = c
		if s.Then, err = d.stmts(s.Then); err != nil {
			return nil, nil, err
		}
		if s.Else, err = d.stmts(s.Else); err != nil {
			return nil, nil, err
		}
		return pre, s, nil
	case *While:
		if line, has := findSugar(s.Cond); has {
			return nil, nil, errf("err_parse", line, "ternaire/match dans une condition de boucle hors subset (réévaluation par itération)")
		}
		var err *Err
		if s.Body, err = d.stmts(s.Body); err != nil {
			return nil, nil, err
		}
		return nil, s, nil
	case *DoWhile:
		if line, has := findSugar(s.Cond); has {
			return nil, nil, errf("err_parse", line, "ternaire/match dans une condition de boucle hors subset (réévaluation par itération)")
		}
		var err *Err
		if s.Body, err = d.stmts(s.Body); err != nil {
			return nil, nil, err
		}
		return nil, s, nil
	case *For:
		if s.Cond != nil {
			if line, has := findSugar(s.Cond); has {
				return nil, nil, errf("err_parse", line, "ternaire/match dans une condition de boucle hors subset (réévaluation par itération)")
			}
		}
		for _, clause := range []Stmt{s.Init, s.Post} {
			if a, ok := clause.(*Assign); ok {
				if line, has := findSugar(a.Expr); has {
					return nil, nil, errf("err_parse", line, "ternaire/match en clause de for hors subset")
				}
			}
		}
		var err *Err
		if s.Body, err = d.stmts(s.Body); err != nil {
			return nil, nil, err
		}
		return nil, s, nil
	case *Block:
		body, err := d.stmts(s.Stmts)
		if err != nil {
			return nil, nil, err
		}
		s.Stmts = body
		return nil, s, nil
	case *Switch:
		pre, subj, err := d.extract(s.Subject)
		if err != nil {
			return nil, nil, err
		}
		s.Subject = subj
		for i := range s.Cases {
			for _, v := range s.Cases[i].Vals {
				if line, has := findSugar(v); has {
					return nil, nil, errf("err_parse", line, "ternaire/match en valeur de case hors subset")
				}
			}
			if s.Cases[i].Body, err = d.stmts(s.Cases[i].Body); err != nil {
				return nil, nil, err
			}
		}
		if s.Default != nil {
			if s.Default, err = d.stmts(s.Default); err != nil {
				return nil, nil, err
			}
		}
		return pre, s, nil
	}
	return nil, st, nil // IncDec
}

// extract hoiste les Ternary/Match d'une expression (gauche→droite) et
// retourne les statements préalables plus l'expression réécrite.
func (d *desugarer) extract(e Expr) ([]Stmt, Expr, *Err) {
	switch x := e.(type) {
	case *Unary:
		pre, sub, err := d.extract(x.X)
		x.X = sub
		return pre, x, err
	case *Binary:
		pl, l, err := d.extract(x.L)
		if err != nil {
			return nil, nil, err
		}
		pr, r, err := d.extract(x.R)
		if err != nil {
			return nil, nil, err
		}
		x.L, x.R = l, r
		return append(pl, pr...), x, nil
	case *Call:
		var pre []Stmt
		for i, a := range x.Args {
			pi, a2, err := d.extract(a)
			if err != nil {
				return nil, nil, err
			}
			pre = append(pre, pi...)
			x.Args[i] = a2
		}
		return pre, x, nil
	case *ArrLit:
		var pre []Stmt
		for i, el := range x.Elems {
			pi, el2, err := d.extract(el)
			if err != nil {
				return nil, nil, err
			}
			pre = append(pre, pi...)
			x.Elems[i] = el2
		}
		return pre, x, nil
	case *Index:
		pre, idx, err := d.extract(x.Idx)
		x.Idx = idx
		return pre, x, err
	case *Ternary:
		condPre, cond, err := d.extract(x.Cond)
		if err != nil {
			return nil, nil, err
		}
		tmp := d.gensym()
		line := x.Line
		if x.A == nil { // forme courte c ?: b — c évalué UNE fois via temporaire
			tc := d.gensym()
			bPre, b, err := d.extract(x.B)
			if err != nil {
				return nil, nil, err
			}
			pre := append(condPre,
				&Assign{Name: tc, Op: "=", Expr: cond, Line: line},
				&If{Cond: &Var{Name: tc, Line: line},
					Then: []Stmt{&Assign{Name: tmp, Op: "=", Expr: &Var{Name: tc, Line: line}, Line: line}},
					Else: append(bPre, &Assign{Name: tmp, Op: "=", Expr: b, Line: line}),
					Line: line})
			return pre, &Var{Name: tmp, Line: line}, nil
		}
		aPre, a, err := d.extract(x.A)
		if err != nil {
			return nil, nil, err
		}
		bPre, b, err := d.extract(x.B)
		if err != nil {
			return nil, nil, err
		}
		pre := append(condPre, &If{Cond: cond,
			Then: append(aPre, &Assign{Name: tmp, Op: "=", Expr: a, Line: line}),
			Else: append(bPre, &Assign{Name: tmp, Op: "=", Expr: b, Line: line}),
			Line: line})
		return pre, &Var{Name: tmp, Line: line}, nil
	case *Match:
		subjPre, subj, err := d.extract(x.Subject)
		if err != nil {
			return nil, nil, err
		}
		tmp := d.gensym()
		line := x.Line
		sw := &Switch{Subject: subj, Line: line}
		for _, arm := range x.Arms {
			for _, v := range arm.Vals {
				if vline, has := findSugar(v); has {
					return nil, nil, errf("err_parse", vline, "ternaire/match en condition d'arm de match hors subset")
				}
			}
			rPre, res, err := d.extract(arm.Result)
			if err != nil {
				return nil, nil, err
			}
			sw.Cases = append(sw.Cases, SwitchCase{
				Vals: arm.Vals,
				Body: append(rPre, &Assign{Name: tmp, Op: "=", Expr: res, Line: line}),
			})
		}
		dPre, dRes, err := d.extract(x.Default)
		if err != nil {
			return nil, nil, err
		}
		sw.Default = append(dPre, &Assign{Name: tmp, Op: "=", Expr: dRes, Line: line})
		return append(subjPre, sw), &Var{Name: tmp, Line: line}, nil
	}
	return nil, e, nil // IntLit, StrLit, Var
}

// findSugar détecte un Ternary/Match résiduel dans une expression.
func findSugar(e Expr) (int, bool) {
	switch x := e.(type) {
	case *Ternary:
		return x.Line, true
	case *Match:
		return x.Line, true
	case *Unary:
		return findSugar(x.X)
	case *Binary:
		if l, has := findSugar(x.L); has {
			return l, true
		}
		return findSugar(x.R)
	case *Call:
		for _, a := range x.Args {
			if l, has := findSugar(a); has {
				return l, true
			}
		}
	case *ArrLit:
		for _, el := range x.Elems {
			if l, has := findSugar(el); has {
				return l, true
			}
		}
	case *Index:
		return findSugar(x.Idx)
	}
	return 0, false
}
