package emit

// Passes idiomatiques sur AST — remplacent les passes regex dont l'audit
// 2026-08-15 a démontré deux défauts de classe (précédence non portée par le
// texte, casts effacés avant appariement). Principe : le corps émis est parsé
// en AST, les réécritures sont calculées sur l'arbre, puis ÉPISSÉES par
// offsets dans le texte d'origine — le formatage hors sites réécrits est
// préservé, et un corps non parsable ressort inchangé (fail-safe, comme les
// passes texte).

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"sort"
	"strings"
)

const astBodyPrefix = "package p\n\nfunc _sgoiterBody() {\n"

type astEdit struct {
	lo, hi int
	text   string
}

// astRewriteBody parse le corps enveloppé, laisse collect proposer des
// remplacements de nœuds, et épisse les remplacements dans le texte d'origine.
// Les éditions qui se chevauchent sont résolues au profit de la première posée.
func astRewriteBody(body string, collect func(src string, f *ast.File, fset *token.FileSet, add func(n ast.Node, text string))) string {
	src := astBodyPrefix + body + "}\n"
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "sgoiter_body.go", src, 0)
	if err != nil {
		return body
	}
	var edits []astEdit
	add := func(n ast.Node, text string) {
		lo := fset.Position(n.Pos()).Offset - len(astBodyPrefix)
		hi := fset.Position(n.End()).Offset - len(astBodyPrefix)
		if lo < 0 || hi > len(body) || lo >= hi {
			return
		}
		edits = append(edits, astEdit{lo: lo, hi: hi, text: text})
	}
	collect(src, f, fset, add)
	if len(edits) == 0 {
		return body
	}
	sort.SliceStable(edits, func(i, j int) bool { return edits[i].lo < edits[j].lo })
	var b strings.Builder
	prev := 0
	for _, e := range edits {
		if e.lo < prev {
			continue // chevauchement : la première édition gagne
		}
		b.WriteString(body[prev:e.lo])
		b.WriteString(e.text)
		prev = e.hi
	}
	b.WriteString(body[prev:])
	return b.String()
}

// nodeSrc rend le texte source exact d'un nœud dans le fichier enveloppé.
func nodeSrc(src string, fset *token.FileSet, n ast.Node) string {
	lo := fset.Position(n.Pos()).Offset
	hi := fset.Position(n.End()).Offset
	if lo < 0 || hi > len(src) || lo >= hi {
		return ""
	}
	return src[lo:hi]
}

// astWalkWithParent parcourt l'arbre en donnant à visit le parent de chaque nœud.
func astWalkWithParent(root ast.Node, visit func(n, parent ast.Node)) {
	var stack []ast.Node
	ast.Inspect(root, func(n ast.Node) bool {
		if n == nil {
			stack = stack[:len(stack)-1]
			return false
		}
		var parent ast.Node
		if len(stack) > 0 {
			parent = stack[len(stack)-1]
		}
		visit(n, parent)
		stack = append(stack, n)
		return true
	})
}

// isGap16Arg reconnaît le littéral 16, éventuellement sous cast entier.
func isGap16Arg(e ast.Expr) bool {
	if lit, ok := e.(*ast.BasicLit); ok {
		return lit.Kind == token.INT && lit.Value == "16"
	}
	if call, ok := e.(*ast.CallExpr); ok && len(call.Args) == 1 {
		if id, ok := call.Fun.(*ast.Ident); ok {
			switch id.Name {
			case "uint64", "uint32", "uint16", "uint8", "int64", "int32", "int16", "int8", "int", "uintptr":
				return isGap16Arg(call.Args[0])
			}
		}
	}
	return false
}

// astExprIsAtomic : vrai si l'expression n'a pas besoin de parenthèses sous
// une négation unaire (ident, littéral, sélecteur, index, appel, parenthèse).
func astExprIsAtomic(e ast.Expr) bool {
	switch e.(type) {
	case *ast.Ident, *ast.BasicLit, *ast.SelectorExpr, *ast.IndexExpr, *ast.CallExpr, *ast.ParenExpr:
		return true
	}
	return false
}

// astParentShieldsAnd : vrai si le contexte parent n'exige pas de parenthèses
// autour d'une expression dont l'opérateur de tête est & (précédence 5).
func astParentShieldsAnd(parent ast.Node) bool {
	switch p := parent.(type) {
	case *ast.AssignStmt, *ast.ValueSpec, *ast.ReturnStmt, *ast.ExprStmt,
		*ast.KeyValueExpr, *ast.CallExpr, *ast.ParenExpr, *ast.IfStmt, *ast.ForStmt:
		return true
	case *ast.BinaryExpr:
		// Sûr uniquement sous un opérateur de précédence strictement inférieure à
		// celle de & (niveau 5) : comparaisons, +,-,|,^, &&, ||.
		return p.Op.Precedence() < 5
	}
	return false
}

// astFoldGapLiteralConstants remplace Gap(x, 16) par (-x) & 15 en garantissant
// la précédence : l'argument est parenthésé s'il n'est pas atomique, et le
// résultat entier est parenthésé si le contexte parent l'exige. Corrige le
// défaut démontré de la variante regex ((-a + b) & 15 pour Gap(a+b, 16)).
func astFoldGapLiteralConstants(body string) string {
	return astRewriteBody(body, func(src string, f *ast.File, fset *token.FileSet, add func(ast.Node, string)) {
		astWalkWithParent(f, func(n, parent ast.Node) {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) != 2 {
				return
			}
			id, ok := call.Fun.(*ast.Ident)
			if !ok || id.Name != "Gap" || !isGap16Arg(call.Args[1]) {
				return
			}
			argText := nodeSrc(src, fset, call.Args[0])
			if argText == "" {
				return
			}
			if !astExprIsAtomic(call.Args[0]) {
				argText = "(" + argText + ")"
			}
			repl := "(-" + argText + ") & 15"
			if !astParentShieldsAnd(parent) {
				repl = "(" + repl + ")"
			}
			add(call, repl)
		})
	})
}

// minMaxCore décompose une IIFE ternaire en (comparaison, retour-si-vrai,
// retour-si-faux). Rend ok=false si la forme ne correspond pas.
func minMaxCore(call *ast.CallExpr) (cmp *ast.BinaryExpr, retTrue, retFalse ast.Expr, ok bool) {
	fn, isLit := call.Fun.(*ast.FuncLit)
	if !isLit || len(call.Args) != 0 || len(fn.Body.List) != 2 {
		return nil, nil, nil, false
	}
	ifStmt, isIf := fn.Body.List[0].(*ast.IfStmt)
	if !isIf || ifStmt.Init != nil || ifStmt.Else != nil || len(ifStmt.Body.List) != 1 {
		return nil, nil, nil, false
	}
	retT, isRet := ifStmt.Body.List[0].(*ast.ReturnStmt)
	if !isRet || len(retT.Results) != 1 {
		return nil, nil, nil, false
	}
	retF, isRet := fn.Body.List[1].(*ast.ReturnStmt)
	if !isRet || len(retF.Results) != 1 {
		return nil, nil, nil, false
	}
	cond := ast.Unparen(ifStmt.Cond)
	// Forme imbriquée : (func() int { if a OP b { return 1 }; return 0 }()) != 0
	if neq, isBin := cond.(*ast.BinaryExpr); isBin && neq.Op == token.NEQ {
		if lit, isLit := ast.Unparen(neq.Y).(*ast.BasicLit); isLit && lit.Value == "0" {
			if inner, isCall := ast.Unparen(neq.X).(*ast.CallExpr); isCall {
				innerCmp, innerT, innerF, innerOK := minMaxCore(inner)
				if innerOK {
					litT, okT := ast.Unparen(innerT).(*ast.BasicLit)
					litF, okF := ast.Unparen(innerF).(*ast.BasicLit)
					if okT && okF && litT.Value == "1" && litF.Value == "0" {
						cond = innerCmp
					}
				}
			}
		}
	}
	bin, isBin := cond.(*ast.BinaryExpr)
	if !isBin {
		return nil, nil, nil, false
	}
	switch bin.Op {
	case token.LSS, token.LEQ, token.GTR, token.GEQ:
		return bin, retT.Results[0], retF.Results[0], true
	}
	return nil, nil, nil, false
}

// normMinMaxOperand rend la forme canonique d'appariement d'un opérande :
// les conversions entières appliquées (récursivement) à un littéral entier
// sont dépliées — un cast de littéral qui compile est toujours sans perte en
// Go (uint8(300) ne compile pas), donc uint32(64) ≡ 64. Tout autre cast est
// conservé tel quel : c'est lui qui portait le défaut de troncature.
func normMinMaxOperand(e ast.Expr) string {
	e = ast.Unparen(e)
	if isIntLiteralChain(e) {
		for {
			e = ast.Unparen(e)
			call, ok := e.(*ast.CallExpr)
			if !ok {
				break
			}
			e = call.Args[0]
		}
		return types.ExprString(ast.Unparen(e))
	}
	return types.ExprString(e)
}

// isIntLiteralChain : vrai si l'expression est un littéral entier sous zéro ou
// plusieurs conversions entières.
func isIntLiteralChain(e ast.Expr) bool {
	e = ast.Unparen(e)
	if lit, ok := e.(*ast.BasicLit); ok {
		return lit.Kind == token.INT
	}
	if call, ok := e.(*ast.CallExpr); ok && len(call.Args) == 1 {
		if id, ok := call.Fun.(*ast.Ident); ok {
			switch id.Name {
			case "uint64", "uint32", "uint16", "uint8", "int64", "int32", "int16", "int8", "int", "uintptr":
				return isIntLiteralChain(call.Args[0])
			}
		}
	}
	return false
}

// astBuiltinMinMax remplace les IIFE ternaires min/max par les intégrées
// min()/max(), UNIQUEMENT quand les opérandes de la comparaison sont
// identiques aux valeurs retournées après normalisation SÛRE (dépliage des
// seuls casts de littéraux entiers, sans perte par construction). La variante
// regex effaçait TOUS les casts avant d'apparier, ce qui transformait
// « if int32(a) < int32(b) { return a }; return b » (comparaison tronquée)
// en min(a, b) pleine largeur : divergence démontrée par oracle 2026-08-15.
func astBuiltinMinMax(body string) string {
	return astRewriteBody(body, func(src string, f *ast.File, fset *token.FileSet, add func(ast.Node, string)) {
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			cmp, retTrue, retFalse, ok := minMaxCore(call)
			if !ok {
				return true
			}
			cx := normMinMaxOperand(cmp.X)
			cy := normMinMaxOperand(cmp.Y)
			rt := normMinMaxOperand(retTrue)
			rf := normMinMaxOperand(retFalse)
			rtSrc := nodeSrc(src, fset, retTrue)
			rfSrc := nodeSrc(src, fset, retFalse)
			var repl string
			switch cmp.Op {
			case token.LSS, token.LEQ:
				if cx == rt && cy == rf {
					repl = "min(" + rtSrc + ", " + rfSrc + ")"
				} else if cx == rf && cy == rt {
					repl = "max(" + rfSrc + ", " + rtSrc + ")"
				}
			case token.GTR, token.GEQ:
				if cx == rt && cy == rf {
					repl = "max(" + rtSrc + ", " + rfSrc + ")"
				} else if cx == rf && cy == rt {
					repl = "min(" + rfSrc + ", " + rtSrc + ")"
				}
			}
			if repl != "" {
				add(call, repl)
				return false
			}
			return true
		})
	})
}
