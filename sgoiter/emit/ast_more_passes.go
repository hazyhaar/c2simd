package emit

// Second lot de passes AST (2026-08-15) : forwarding store→load adjacent,
// rotations droites, masques identité, casts d'indice, aplatissement des
// boucles à un tour, hints BCE, et élimination des globales mortes à l'échelle
// du fichier. Même socle que ast_idiomatic_passes.go : épissage par offsets,
// fail-safe sur corps non parsable.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"regexp"
	"strconv"
	"strings"
)

// exprKey rend une clé syntaxique canonique pour comparer deux expressions.
func exprKey(e ast.Expr) string { return types.ExprString(e) }

// astForwardAdjacentStoreLoad remplace, dans chaque bloc, le motif
//
//	B[I] = V        (V identifiant ou littéral)
//	x := B[I]       (immédiatement adjacent, même base et même indice)
//
// par x := V — le rechargement du slot tout juste écrit coûte des µops et
// casse la tenue en registres (motif mesuré dans la boucle BlaMka d'Argon2).
// L'adjacence stricte rend la passe triviale à prouver : aucune écriture ni
// aucun appel ne peut s'intercaler.
func astForwardAdjacentStoreLoad(body string) string {
	return astRewriteBody(body, func(src string, f *ast.File, fset *token.FileSet, add func(ast.Node, string)) {
		ast.Inspect(f, func(n ast.Node) bool {
			blk, ok := n.(*ast.BlockStmt)
			if !ok {
				return true
			}
			for i := 0; i+1 < len(blk.List); i++ {
				st1, ok1 := blk.List[i].(*ast.AssignStmt)
				st2, ok2 := blk.List[i+1].(*ast.AssignStmt)
				if !ok1 || !ok2 || len(st1.Lhs) != 1 || len(st1.Rhs) != 1 || len(st2.Lhs) != 1 || len(st2.Rhs) != 1 {
					continue
				}
				if st1.Tok != token.ASSIGN {
					continue
				}
				storeIdx, ok := st1.Lhs[0].(*ast.IndexExpr)
				if !ok {
					continue
				}
				switch st1.Rhs[0].(type) {
				case *ast.Ident, *ast.BasicLit:
				default:
					continue
				}
				loadIdx, ok := st2.Rhs[0].(*ast.IndexExpr)
				if !ok {
					continue
				}
				if exprKey(storeIdx) != exprKey(loadIdx) {
					continue
				}
				add(st2.Rhs[0], nodeSrc(src, fset, st1.Rhs[0]))
			}
			return true
		})
	})
}

// Note (2026-08-15) : la réécriture RotateLeft(x, -N) → RotateRight(x, N),
// suggérée en revue, est INFONDÉE en Go — math/bits n'expose pas RotateRight ;
// la rotation droite s'écrit RotateLeft à compte négatif. Passe retirée après
// échec de compilation du dogfood.

// astDropIdentityMasks supprime les masques identité `x & uint32(0xffffffff)`
// et `x & uint64(0xffffffffffffffff)` (et leurs formes inversées) : en Go, les
// deux opérandes de & partagent déjà le type, le masque pleine largeur est un
// no-op.
func astDropIdentityMasks(body string) string {
	full := map[string]string{
		"uint32": "0xffffffff",
		"uint64": "0xffffffffffffffff",
	}
	isIdentityMask := func(e ast.Expr) bool {
		call, ok := ast.Unparen(e).(*ast.CallExpr)
		if !ok || len(call.Args) != 1 {
			return false
		}
		id, ok := call.Fun.(*ast.Ident)
		if !ok {
			return false
		}
		want, ok := full[id.Name]
		if !ok {
			return false
		}
		lit, ok := ast.Unparen(call.Args[0]).(*ast.BasicLit)
		if !ok || lit.Kind != token.INT {
			return false
		}
		v1, err1 := strconv.ParseUint(strings.TrimPrefix(strings.ToLower(lit.Value), "0x"), 16, 64)
		v2, _ := strconv.ParseUint(strings.TrimPrefix(want, "0x"), 16, 64)
		if err1 != nil {
			return false
		}
		return v1 == v2 && strings.HasPrefix(strings.ToLower(lit.Value), "0x")
	}
	return astRewriteBody(body, func(src string, f *ast.File, fset *token.FileSet, add func(ast.Node, string)) {
		ast.Inspect(f, func(n ast.Node) bool {
			bin, ok := n.(*ast.BinaryExpr)
			if !ok || bin.Op != token.AND {
				return true
			}
			if isIdentityMask(bin.Y) {
				add(bin, nodeSrc(src, fset, bin.X))
				return false
			}
			if isIdentityMask(bin.X) {
				add(bin, nodeSrc(src, fset, bin.Y))
				return false
			}
			return true
		})
	})
}

// astSimplifyIndexCasts nettoie les indices de la forme int(E) — un indice Go
// accepte tout type entier — quand E est un identifiant, un sélecteur, ou une
// somme d'identifiants/sélecteurs/littéraux (casts de littéraux dépliés) :
// h[int(uint64(1)+v3)] devient h[v3+1], ctx.C[int(ctx.C_idx)] devient
// ctx.C[ctx.C_idx].
func astSimplifyIndexCasts(body string) string {
	var renderInner func(e ast.Expr) (string, bool)
	renderInner = func(e ast.Expr) (string, bool) {
		e = ast.Unparen(e)
		switch v := e.(type) {
		case *ast.Ident:
			return v.Name, true
		case *ast.SelectorExpr:
			return types.ExprString(v), true
		case *ast.BasicLit:
			if v.Kind == token.INT {
				return v.Value, true
			}
		case *ast.CallExpr:
			if isIntLiteralChain(v) {
				return normMinMaxOperand(v), true
			}
		case *ast.BinaryExpr:
			if v.Op != token.ADD {
				return "", false
			}
			l, okL := renderInner(v.X)
			r, okR := renderInner(v.Y)
			if !okL || !okR {
				return "", false
			}
			// littéral en second pour la lisibilité (v3+1, pas 1+v3)
			if _, err := strconv.Atoi(l); err == nil {
				l, r = r, l
			}
			return l + "+" + r, true
		}
		return "", false
	}
	return astRewriteBody(body, func(src string, f *ast.File, fset *token.FileSet, add func(ast.Node, string)) {
		ast.Inspect(f, func(n ast.Node) bool {
			idx, ok := n.(*ast.IndexExpr)
			if !ok {
				return true
			}
			call, ok := ast.Unparen(idx.Index).(*ast.CallExpr)
			if !ok || len(call.Args) != 1 {
				return true
			}
			fn, ok := call.Fun.(*ast.Ident)
			if !ok || fn.Name != "int" {
				return true
			}
			if inner, okR := renderInner(call.Args[0]); okR {
				add(idx.Index, inner)
			}
			return true
		})
	})
}

// constTripLoop décompose le motif émis d'une boucle à compteur :
//
//	v = 0            (statement PRÉCÉDENT, garantit la valeur d'entrée)
//	for v < N {      (N littéral entier)
//	    …corps…
//	    v++
//	}
//
// et rend (variable, N, corps sans l'incrément). ok=false si la forme,
// la garantie d'entrée ou l'hygiène du compteur manquent.
func constTripLoop(prev ast.Stmt, fs *ast.ForStmt) (loopVar *ast.Ident, trip int, body []ast.Stmt, ok bool) {
	if fs.Init != nil || fs.Post != nil || fs.Cond == nil {
		return nil, 0, nil, false
	}
	cond, isBin := fs.Cond.(*ast.BinaryExpr)
	if !isBin || cond.Op != token.LSS {
		return nil, 0, nil, false
	}
	v, isIdent := cond.X.(*ast.Ident)
	if !isIdent {
		return nil, 0, nil, false
	}
	lit, isLit := cond.Y.(*ast.BasicLit)
	if !isLit || lit.Kind != token.INT {
		return nil, 0, nil, false
	}
	n, err := strconv.Atoi(lit.Value)
	if err != nil || n < 1 {
		return nil, 0, nil, false
	}
	// Garantie d'entrée : le statement précédent est exactement `v = 0`.
	// Sans elle, aplatir/dérouler avec v substitué par 0..N-1 serait FAUX
	// (une boucle entrée avec v >= N ne s'exécute pas du tout).
	as, isAssign := prev.(*ast.AssignStmt)
	if !isAssign || as.Tok != token.ASSIGN || len(as.Lhs) != 1 || len(as.Rhs) != 1 {
		return nil, 0, nil, false
	}
	lhs, isIdent := as.Lhs[0].(*ast.Ident)
	if !isIdent || lhs.Name != v.Name {
		return nil, 0, nil, false
	}
	rhs, isLit := as.Rhs[0].(*ast.BasicLit)
	if !isLit || rhs.Kind != token.INT || rhs.Value != "0" {
		return nil, 0, nil, false
	}
	if len(fs.Body.List) < 2 {
		return nil, 0, nil, false
	}
	last, isInc := fs.Body.List[len(fs.Body.List)-1].(*ast.IncDecStmt)
	if !isInc || last.Tok != token.INC {
		return nil, 0, nil, false
	}
	incVar, isIdent := last.X.(*ast.Ident)
	if !isIdent || incVar.Name != v.Name {
		return nil, 0, nil, false
	}
	inner := fs.Body.List[:len(fs.Body.List)-1]
	// Ni écriture du compteur, ni rupture de flot dans le corps.
	bad := false
	for _, st := range inner {
		ast.Inspect(st, func(m ast.Node) bool {
			switch w := m.(type) {
			case *ast.AssignStmt:
				for _, l := range w.Lhs {
					if id, isID := l.(*ast.Ident); isID && id.Name == v.Name {
						bad = true
					}
				}
			case *ast.IncDecStmt:
				if id, isID := w.X.(*ast.Ident); isID && id.Name == v.Name {
					bad = true
				}
			case *ast.BranchStmt, *ast.ReturnStmt, *ast.LabeledStmt:
				bad = true
			}
			return !bad
		})
	}
	if bad {
		return nil, 0, nil, false
	}
	return v, n, inner, true
}

// lineIndentAt rend l'indentation (tabs/espaces) de la ligne contenant l'offset.
func lineIndentAt(src string, off int) string {
	start := off
	for start > 0 && src[start-1] != '\n' {
		start--
	}
	end := start
	for end < len(src) && (src[end] == '\t' || src[end] == ' ') {
		end++
	}
	return src[start:end]
}

// counterUsedAfter : vrai si le compteur est référencé après la fin de la boucle
// (après déroulage, v ne vaudrait plus N — on refuse alors la transformation).
func counterUsedAfter(f *ast.File, name string, after token.Pos) bool {
	used := false
	ast.Inspect(f, func(m ast.Node) bool {
		if id, ok := m.(*ast.Ident); ok && id.Name == name && id.Pos() >= after {
			used = true
		}
		return !used
	})
	return used
}

// astTransformShiftedClearLoops convertit la boucle de mise à zéro DÉCALÉE
//
//	v = 0
//	for v < N { arr[v+K] = 0; v++ }
//
// en clear(arr[K:K+N]) — la variante regex ne couvrait que K=0 (motif Fe_1 :
// h[0]=1 puis zéros sur h[1..9]).
func astTransformShiftedClearLoops(bodySrc string) string {
	return astRewriteBody(bodySrc, func(src string, f *ast.File, fset *token.FileSet, add func(ast.Node, string)) {
		ast.Inspect(f, func(n ast.Node) bool {
			blk, ok := n.(*ast.BlockStmt)
			if !ok {
				return true
			}
			for i := 1; i < len(blk.List); i++ {
				fs, isFor := blk.List[i].(*ast.ForStmt)
				if !isFor {
					continue
				}
				v, trip, inner, ok := constTripLoop(blk.List[i-1], fs)
				if !ok || len(inner) != 1 || counterUsedAfter(f, v.Name, fs.End()) {
					continue
				}
				as, isAssign := inner[0].(*ast.AssignStmt)
				if !isAssign || as.Tok != token.ASSIGN || len(as.Lhs) != 1 || len(as.Rhs) != 1 {
					continue
				}
				// membre droit : zéro (littéral ou cast de littéral)
				if !isIntLiteralChain(as.Rhs[0]) || normMinMaxOperand(as.Rhs[0]) != "0" {
					continue
				}
				idx, isIdx := as.Lhs[0].(*ast.IndexExpr)
				if !isIdx {
					continue
				}
				base, isID := idx.X.(*ast.Ident)
				if !isID {
					continue
				}
				offset := -1
				switch ie := ast.Unparen(idx.Index).(type) {
				case *ast.Ident:
					if ie.Name == v.Name {
						offset = 0
					}
				case *ast.BinaryExpr:
					if ie.Op == token.ADD {
						l, lID := ast.Unparen(ie.X).(*ast.Ident)
						r, rLit := ast.Unparen(ie.Y).(*ast.BasicLit)
						if lID && rLit && l.Name == v.Name && r.Kind == token.INT {
							if k, err := strconv.Atoi(r.Value); err == nil {
								offset = k
							}
						}
					}
				}
				if offset < 0 {
					continue
				}
				var repl string
				if offset == 0 {
					repl = "clear(" + base.Name + "[:" + strconv.Itoa(trip) + "])"
				} else {
					repl = "clear(" + base.Name + "[" + strconv.Itoa(offset) + ":" + strconv.Itoa(offset+trip) + "])"
				}
				add(fs, repl)
			}
			return true
		})
	})
}

// astUnrollConstTripLoops déroule les boucles à trip constant 1..maxUnrollTrip
// (corps court, compteur hygiénique, valeur d'entrée garantie par `v = 0`
// adjacent, compteur mort après la boucle) en substituant le compteur par les
// littéraux 0..N-1 — Fe_neg/add/sub/cswap et les boucles Poly1305 init/final
// deviennent du straight-line où le compilateur voit les indices constants.
const maxUnrollTrip = 10
const maxUnrollStmts = 4

func astUnrollConstTripLoops(bodySrc string) string {
	return astRewriteBody(bodySrc, func(src string, f *ast.File, fset *token.FileSet, add func(ast.Node, string)) {
		ast.Inspect(f, func(n ast.Node) bool {
			blk, ok := n.(*ast.BlockStmt)
			if !ok {
				return true
			}
			for i := 1; i < len(blk.List); i++ {
				fs, isFor := blk.List[i].(*ast.ForStmt)
				if !isFor {
					continue
				}
				v, trip, inner, ok := constTripLoop(blk.List[i-1], fs)
				if !ok || trip > maxUnrollTrip || len(inner) > maxUnrollStmts {
					continue
				}
				if counterUsedAfter(f, v.Name, fs.End()) {
					continue
				}
				// Le corps doit RÉFÉRENCER le compteur et ne déclarer aucune
				// variable. Sans référence, les copies seraient identiques :
				// zéro gain, et archDeduplicateStores (module-level, aval) les
				// recollerait — miscompile CRC32 mesuré le 2026-08-15 (8 tours
				// réduits à 2). Un `:=` dupliqué ne compilerait pas.
				usesCounter, hasDefine := false, false
				for _, st := range inner {
					ast.Inspect(st, func(m ast.Node) bool {
						if id, isID := m.(*ast.Ident); isID && id.Name == v.Name {
							usesCounter = true
						}
						if as, isAs := m.(*ast.AssignStmt); isAs && as.Tok == token.DEFINE {
							hasDefine = true
						}
						return true
					})
				}
				if !usesCounter || hasDefine {
					continue
				}
				indent := lineIndentAt(src, fset.Position(fs.Pos()).Offset)
				re := regexp.MustCompile(`\b` + regexp.QuoteMeta(v.Name) + `\b`)
				var parts []string
				for it := 0; it < trip; it++ {
					lit := strconv.Itoa(it)
					for _, st := range inner {
						parts = append(parts, re.ReplaceAllString(nodeSrc(src, fset, st), lit))
					}
				}
				add(fs, strings.Join(parts, "\n"+indent))
			}
			return true
		})
	})
}

// astInsertBoundsHints insère `_ = s[M]` en tête de fonction quand un slice y
// est accédé par au moins minConstIdx indices constants distincts (max M) : un
// seul contrôle de borne domine alors tous les accès (BCE), au lieu d'un par
// accès en ordre croissant. Ne s'applique qu'aux paramètres de type slice
// (jamais aux tableaux, dont les bornes sont déjà constantes).
func astInsertBoundsHints(funcSrc string) string {
	const minConstIdx = 8
	return astRewriteDecls(funcSrc, func(src string, f *ast.File, fset *token.FileSet, add func(ast.Node, string)) {
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Body == nil || len(fn.Body.List) == 0 {
				continue
			}
			sliceParams := map[string]bool{}
			for _, p := range fn.Type.Params.List {
				if _, isSlice := p.Type.(*ast.ArrayType); isSlice {
					if at := p.Type.(*ast.ArrayType); at.Len == nil {
						for _, nm := range p.Names {
							sliceParams[nm.Name] = true
						}
					}
				}
			}
			if len(sliceParams) == 0 {
				continue
			}
			maxIdx := map[string]int{}
			count := map[string]map[string]bool{}
			assigned := map[string]bool{}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if as, ok := n.(*ast.AssignStmt); ok {
					for _, l := range as.Lhs {
						if id, ok := l.(*ast.Ident); ok {
							assigned[id.Name] = true
						}
					}
				}
				idx, ok := n.(*ast.IndexExpr)
				if !ok {
					return true
				}
				base, ok := idx.X.(*ast.Ident)
				if !ok || !sliceParams[base.Name] {
					return true
				}
				lit, ok := idx.Index.(*ast.BasicLit)
				if !ok || lit.Kind != token.INT {
					return true
				}
				v, err := strconv.Atoi(lit.Value)
				if err != nil {
					return true
				}
				if count[base.Name] == nil {
					count[base.Name] = map[string]bool{}
				}
				count[base.Name][lit.Value] = true
				if v > maxIdx[base.Name] {
					maxIdx[base.Name] = v
				}
				return true
			})
			var names []string
			for name := range count {
				names = append(names, name)
			}
			// ordre déterministe : l'itération de map varie d'un run à l'autre
			// et casserait l'égalité octet pour octet du corpus versionné.
			for i := 1; i < len(names); i++ {
				for j := i; j > 0 && names[j] < names[j-1]; j-- {
					names[j], names[j-1] = names[j-1], names[j]
				}
			}
			var hints []string
			for _, name := range names {
				if len(count[name]) >= minConstIdx && !assigned[name] {
					hints = append(hints, "_ = "+name+"["+strconv.Itoa(maxIdx[name])+"]")
				}
			}
			if len(hints) == 0 {
				continue
			}
			first := fn.Body.List[0]
			add(first, strings.Join(hints, "\n\t")+"\n\t"+nodeSrc(src, fset, first))
		}
	})
}

// astSimplifyNegatedComparisons plie !(E != 0) en E == 0 et !(E == 0) en
// E != 0 pour TOUTE expression E, appels compris — la variante regex ne
// couvrait que les identifiants (if !(Invsqrt(...) != 0) survivait, audit
// 2026-08-15).
func astSimplifyNegatedComparisons(body string) string {
	return astRewriteBody(body, func(src string, f *ast.File, fset *token.FileSet, add func(ast.Node, string)) {
		ast.Inspect(f, func(n ast.Node) bool {
			un, ok := n.(*ast.UnaryExpr)
			if !ok || un.Op != token.NOT {
				return true
			}
			bin, ok := ast.Unparen(un.X).(*ast.BinaryExpr)
			if !ok {
				return true
			}
			var newOp string
			switch bin.Op {
			case token.NEQ:
				newOp = "=="
			case token.EQL:
				newOp = "!="
			default:
				return true
			}
			lit, ok := ast.Unparen(bin.Y).(*ast.BasicLit)
			if !ok || lit.Kind != token.INT || lit.Value != "0" {
				return true
			}
			add(un, nodeSrc(src, fset, bin.X)+" "+newOp+" 0")
			return false
		})
	})
}

// astStripAssignLiteralCasts nettoie `lhs = T(littéral)` en `lhs = littéral`
// (affectation NUE uniquement, jamais := qui typerait la variable) — le
// littéral non typé adopte le type du membre gauche. Motif apparu avec le
// typage int32 du lot B : h[0] = int32(1).
func astStripAssignLiteralCasts(body string) string {
	return astRewriteBody(body, func(src string, f *ast.File, fset *token.FileSet, add func(ast.Node, string)) {
		ast.Inspect(f, func(n ast.Node) bool {
			as, ok := n.(*ast.AssignStmt)
			if !ok || as.Tok != token.ASSIGN || len(as.Rhs) != 1 {
				return true
			}
			rhs := as.Rhs[0]
			if _, isLit := ast.Unparen(rhs).(*ast.BasicLit); isLit {
				return true
			}
			if isIntLiteralChain(rhs) {
				add(rhs, normMinMaxOperand(rhs))
			}
			return true
		})
	})
}

const astDeclPrefix = "package p\n\n"

// astRewriteDecls : même mécanique d'épissage que astRewriteBody, mais pour un
// fragment de niveau DÉCLARATION (une ou plusieurs fonctions complètes) — la
// signature reste visible du parseur.
func astRewriteDecls(frag string, collect func(src string, f *ast.File, fset *token.FileSet, add func(n ast.Node, text string))) string {
	src := astDeclPrefix + frag
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "sgoiter_decls.go", src, 0)
	if err != nil {
		return frag
	}
	var edits []astEdit
	add := func(n ast.Node, text string) {
		lo := fset.Position(n.Pos()).Offset - len(astDeclPrefix)
		hi := fset.Position(n.End()).Offset - len(astDeclPrefix)
		if lo < 0 || hi > len(frag) || lo >= hi {
			return
		}
		edits = append(edits, astEdit{lo: lo, hi: hi, text: text})
	}
	collect(src, f, fset, add)
	if len(edits) == 0 {
		return frag
	}
	sortEditsAsc(edits)
	var b strings.Builder
	prev := 0
	for _, e := range edits {
		if e.lo < prev {
			continue
		}
		b.WriteString(frag[prev:e.lo])
		b.WriteString(e.text)
		prev = e.hi
	}
	b.WriteString(frag[prev:])
	return b.String()
}

func sortEditsAsc(edits []astEdit) {
	for i := 1; i < len(edits); i++ {
		for j := i; j > 0 && edits[j].lo < edits[j-1].lo; j-- {
			edits[j], edits[j-1] = edits[j-1], edits[j]
		}
	}
}

// astStripDeadGlobals supprime, à l'échelle du FICHIER émis complet, les
// variables package-level jamais référencées (fixpoint : une globale lue
// uniquement par une autre globale morte tombe aussi). Ne touche jamais les
// noms exportés ni les noms de keepNames (consommés par des strates main hors
// du fichier émis, comme `zero`).
func astStripDeadGlobals(src string, keepNames ...string) string {
	keep := map[string]bool{}
	for _, k := range keepNames {
		keep[k] = true
	}
	for {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "emitted.go", src, 0)
		if err != nil {
			return src
		}
		decl := map[string]*ast.GenDecl{}
		declIdent := map[string]*ast.Ident{}
		for _, d := range f.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok || len(gd.Specs) != 1 {
				continue
			}
			switch gd.Tok {
			case token.VAR:
				vs, ok := gd.Specs[0].(*ast.ValueSpec)
				if !ok || len(vs.Names) != 1 {
					continue
				}
				name := vs.Names[0].Name
				if ast.IsExported(name) || keep[name] || name == "_" {
					continue
				}
				decl[name] = gd
				declIdent[name] = vs.Names[0]
			case token.TYPE:
				// Types morts (S25/S26, audit 2026-08-15). Contrairement aux
				// vars, les types émis sont exportés par convention C→Go : la
				// garde d'export ne s'applique pas — la passe étant opt-in et
				// bornée par keepNames, un type consommé hors du fichier émis
				// se protège nommément dans la recette.
				ts, ok := gd.Specs[0].(*ast.TypeSpec)
				if !ok || keep[ts.Name.Name] {
					continue
				}
				decl[ts.Name.Name] = gd
				declIdent[ts.Name.Name] = ts.Name
			}
		}
		if len(decl) == 0 {
			return src
		}
		used := map[string]bool{}
		ast.Inspect(f, func(n ast.Node) bool {
			id, ok := n.(*ast.Ident)
			if !ok {
				return true
			}
			if declID, isDecl := declIdent[id.Name]; isDecl && declID == id {
				return true // position de déclaration, pas un usage
			}
			used[id.Name] = true
			return true
		})
		var edits []astEdit
		for name, gd := range decl {
			if used[name] {
				continue
			}
			lo := fset.Position(gd.Pos()).Offset
			hi := fset.Position(gd.End()).Offset
			// absorber la fin de ligne
			for hi < len(src) && (src[hi] == '\n' || src[hi] == '\r') {
				hi++
				break
			}
			edits = append(edits, astEdit{lo: lo, hi: hi, text: ""})
		}
		if len(edits) == 0 {
			return src
		}
		// épissage décroissant
		for i := range edits {
			for j := i + 1; j < len(edits); j++ {
				if edits[j].lo > edits[i].lo {
					edits[i], edits[j] = edits[j], edits[i]
				}
			}
		}
		for _, e := range edits {
			src = src[:e.lo] + src[e.hi:]
		}
	}
}
