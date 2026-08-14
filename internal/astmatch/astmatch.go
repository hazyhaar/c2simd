package astmatch

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

// TransformAST applique de manière déterministe les optimisations d'AST sur le C transpilé.
// Consomme uniquement ArchtimeRulesTable[Kind=rewrite] (DeadCode) + passes structurelles :
//  1. rotl32/rotl64 appels → bits.RotateLeft32/64 (+ DeadCode sur déf. rotl32)
//  2. Fold wrappers libc *FromInt32/*FromUint32 (littéraux non négatifs)
//  3. Motifs ROL/ROR (x<<N)|^|(x>>(W-N)) et inverse → bits.RotateLeft*
//  4. Corps load32_le / store32_le → unsafe.Pointer *uint32
//  5. Élision tls *libc.TLS (T0 unexported) + sites d'appel ; drop imports morts
//  6. * ( **T )( __ccgo_up(E) ) → ( *T )( unsafe.Pointer(E) )  (F-20260810-ccgo-up-goulot)
//  6b. polish (*(*[N]T)(p))[i] → (*[N]T)(p)[i] ; 2e tour uintptr(-N) post-capture
// Hors scope (v2) : handwrite_pointer (simd_*.go), declared, boucle→simd générique.
func TransformAST(src []byte) ([]byte, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "src.go", src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parsing ast: %w", err)
	}

	deadCodeSymbols := make(map[string]bool)
	for _, rule := range AppliedRules() {
		if rule.DeadCode {
			deadCodeSymbols[rule.Symbol] = true
		}
	}

	// pureTLSFuncs initial : premier round avant réécritures (rotl peut encore
	// référencer tls dans le corps). Un point fixe post-Apply complète T0/T1
	// (dogfood chacha20_qr : double_round gardait tls mort après strip de QR).
	pureTLSFuncs := findUnexportedT0(node)

	modifiedBits := false
	modifiedUnsafe := false

	astutil.Apply(node, func(cursor *astutil.Cursor) bool {
		n := cursor.Node()

		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			if deadCodeSymbols[funcDecl.Name.Name] {
				cursor.Delete()
				return false
			}

			if funcDecl.Name.Name == "crypto_aead_lock" {
				funcDecl.Name.Name = "Crypto_aead_lock"
			}

			// Passe P2.1 : Suppression du premier paramètre tls si la fonction est T0 (pure)
			if pureTLSFuncs[funcDecl.Name.Name] && len(funcDecl.Type.Params.List) > 0 {
				funcDecl.Type.Params.List = funcDecl.Type.Params.List[1:]
			}

			if funcDecl.Name.Name == "load32_le" {
				funcDecl.Body = &ast.BlockStmt{
					List: []ast.Stmt{
						&ast.ReturnStmt{
							Results: []ast.Expr{
								&ast.StarExpr{
									X: &ast.CallExpr{
										Fun: &ast.ParenExpr{
											X: &ast.StarExpr{
												X: ast.NewIdent("uint32"),
											},
										},
										Args: []ast.Expr{
											&ast.CallExpr{
												Fun: &ast.SelectorExpr{
													X:   ast.NewIdent("unsafe"),
													Sel: ast.NewIdent("Pointer"),
												},
												Args: []ast.Expr{ast.NewIdent("s")},
											},
										},
									},
								},
							},
						},
					},
				}
				modifiedUnsafe = true
				return true
			}

			if funcDecl.Name.Name == "store32_le" {
				funcDecl.Body = &ast.BlockStmt{
					List: []ast.Stmt{
						&ast.AssignStmt{
							Lhs: []ast.Expr{
								&ast.StarExpr{
									X: &ast.CallExpr{
										Fun: &ast.ParenExpr{
											X: &ast.StarExpr{
												X: ast.NewIdent("uint32"),
											},
										},
										Args: []ast.Expr{
											&ast.CallExpr{
												Fun: &ast.SelectorExpr{
													X:   ast.NewIdent("unsafe"),
													Sel: ast.NewIdent("Pointer"),
												},
												Args: []ast.Expr{ast.NewIdent("out")},
											},
										},
									},
								},
							},
							Tok: token.ASSIGN,
							Rhs: []ast.Expr{ast.NewIdent("in")},
						},
					},
				}
				modifiedUnsafe = true
				return true
			}
		}

		if call, ok := n.(*ast.CallExpr); ok {
			// Passe P2.1 : Élision de l'argument tls dans les sites d'appel des fonctions T0
			if ident, ok := call.Fun.(*ast.Ident); ok && pureTLSFuncs[ident.Name] {
				if len(call.Args) > 0 {
					if argIdent, ok := call.Args[0].(*ast.Ident); ok && argIdent.Name == "tls" {
						call.Args = call.Args[1:]
					}
				}
			}

			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				// ccgo extrême (lz4) : iqlibc.__builtin_memmove sans import alias
				// → libc.Xmemmove (modernc.org/libc).
				if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "iqlibc" {
					sel.X = ast.NewIdent("libc")
					if strings.HasPrefix(sel.Sel.Name, "__builtin_") {
						base := strings.TrimPrefix(sel.Sel.Name, "__builtin_")
						sel.Sel.Name = "X" + base
					}
					return true
				}
				if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "libc" {
					funcName := sel.Sel.Name
					if strings.HasSuffix(funcName, "FromInt32") || strings.HasSuffix(funcName, "FromUint32") {
						if len(call.Args) == 1 {
							arg := call.Args[0]
							isNegative := false
							if unary, ok := arg.(*ast.UnaryExpr); ok && unary.Op == token.SUB {
								isNegative = true
							}

							if !isNegative {
								targetType := ""
								if strings.HasPrefix(funcName, "Uint64") {
									targetType = "uint64"
								} else if strings.HasPrefix(funcName, "Uint32") {
									targetType = "uint32"
								} else if strings.HasPrefix(funcName, "Int32") {
									targetType = "int32"
								}

								if targetType != "" {
									cursor.Replace(&ast.CallExpr{
										Fun:  ast.NewIdent(targetType),
										Args: []ast.Expr{arg},
									})
									return true
								}
							}
						}
					}
				}
			}

			// Helpers rotate : rotl32/64 (ccgo mono), L32 (tweetnacl)
			if ident, ok := call.Fun.(*ast.Ident); ok {
				name := ident.Name
				isRol32 := name == "rotl32" || name == "L32" || name == "ROTL32" || name == "rotl"
				isRol64 := name == "rotl64" || name == "L64" || name == "ROTL64"
				if isRol32 || isRol64 {
					var valArg, distArg ast.Expr
					if len(call.Args) >= 3 {
						// (tls, x, c) ou (x, c, _) 
						valArg = call.Args[1]
						distArg = call.Args[2]
					} else if len(call.Args) == 2 {
						valArg = call.Args[0]
						distArg = call.Args[1]
					}

					if valArg != nil && distArg != nil {
						distExpr := distArg
						if innerCall, ok := distArg.(*ast.CallExpr); ok {
							if innerIdent, ok := innerCall.Fun.(*ast.Ident); ok && (innerIdent.Name == "uint32" || innerIdent.Name == "u32" || innerIdent.Name == "int32" || innerIdent.Name == "int64" || innerIdent.Name == "int8" || innerIdent.Name == "int") {
								if len(innerCall.Args) == 1 {
									distExpr = innerCall.Args[0]
								}
							}
						}

						intDistExpr := &ast.CallExpr{
							Fun:  ast.NewIdent("int"),
							Args: []ast.Expr{distExpr},
						}

						selName := "RotateLeft32"
						if isRol64 {
							selName = "RotateLeft64"
						}

						rolArg := valArg
						// tweetnacl : typedef unsigned long u32 (64-bit LP64) mais rotate 32-bit.
						// Cast uint32 → RotateLeft32 → u32 pour rester typable.
						if name == "L32" {
							rolArg = &ast.CallExpr{
								Fun:  ast.NewIdent("uint32"),
								Args: []ast.Expr{valArg},
							}
						}

						newCall := ast.Expr(&ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X:   ast.NewIdent("bits"),
								Sel: ast.NewIdent(selName),
							},
							Args: []ast.Expr{
								rolArg,
								intDistExpr,
							},
						})
						if name == "L32" {
							newCall = &ast.CallExpr{
								Fun:  ast.NewIdent("u32"),
								Args: []ast.Expr{newCall},
							}
						}

						cursor.Replace(newCall)
						modifiedBits = true
						return true
					}
				}
			}
		}

		// 4. Passe P1.5 : motifs rotate → bits.RotateLeft*
		//   ROL : (x << N) |/^ (x >> (W-N))
		//   ROR : (x >> N) |/^ (x << (W-N))  → RotateLeft(x, W-N)
		// OR et XOR admis (bits disjoints sur un rotate entier).
		if binaryExpr, ok := n.(*ast.BinaryExpr); ok && (binaryExpr.Op == token.OR || binaryExpr.Op == token.XOR) {
			if call, ok := matchRotateBinary(binaryExpr); ok {
				cursor.Replace(call)
				modifiedBits = true
				return true
			}
		}

		// 4b. Pointer arith : x + uintptr(-N) → x - uintptr(N)
		// Dogfood tiny-regex / lz4 : `constant -1 overflows uintptr`.
		if binaryExpr, ok := n.(*ast.BinaryExpr); ok && binaryExpr.Op == token.ADD {
			if mag, ok := matchUintptrNegConst(binaryExpr.Y); ok {
				cursor.Replace(&ast.BinaryExpr{
					X:  binaryExpr.X,
					Op: token.SUB,
					Y: &ast.CallExpr{
						Fun: ast.NewIdent("uintptr"),
						Args: []ast.Expr{
							&ast.BasicLit{Kind: token.INT, Value: fmt.Sprintf("%d", mag)},
						},
					},
				})
				return true
			}
		}

		// 6. * ( **T )( __ccgo_up(E) ) → ( *T )( unsafe.Pointer(E) )
		// Sémantique : __ccgo_up(n) = unsafe.Pointer(&n) puis double-deref via **T.
		// Couvre load scalaire **(**T)(__ccgo_up(E)) et base tableau *(**[N]T)(__ccgo_up(E)).
		if star, ok := n.(*ast.StarExpr); ok {
			if repl, ok := matchCcgoUpStar(star); ok {
				cursor.Replace(repl)
				modifiedUnsafe = true
				return true
			}
		}

		return true
	}, nil)

	// 6b. second tour : (*(*[N]T)(p))[i] → (*[N]T)(p)[i]
	// (le parent IndexExpr n'est pas revu après rewrite __ccgo_up du fils)
	astutil.Apply(node, func(cursor *astutil.Cursor) bool {
		idx, ok := cursor.Node().(*ast.IndexExpr)
		if !ok {
			return true
		}
		if repl, ok := matchArrayPtrIndex(idx); ok {
			cursor.Replace(repl)
		}
		return true
	}, nil)

	// 4b-bis. second tour uintptr(-N) : le 1er tour rate les ADD
	// capturés comme args de __ccgo_up puis recollés dans unsafe.Pointer(E)
	// sans re-visite fiable du BinaryExpr (dogfood tiny_regex / lz4).
	astutil.Apply(node, func(cursor *astutil.Cursor) bool {
		bin, ok := cursor.Node().(*ast.BinaryExpr)
		if !ok || bin.Op != token.ADD {
			return true
		}
		if mag, ok := matchUintptrNegConst(bin.Y); ok {
			cursor.Replace(&ast.BinaryExpr{
				X:  bin.X,
				Op: token.SUB,
				Y: &ast.CallExpr{
					Fun: ast.NewIdent("uintptr"),
					Args: []ast.Expr{
						&ast.BasicLit{Kind: token.INT, Value: fmt.Sprintf("%d", mag)},
					},
				},
			})
		}
		return true
	}, nil)

	if modifiedBits {
		astutil.AddImport(fset, node, "math/bits")
	}
	if modifiedUnsafe {
		astutil.AddImport(fset, node, "unsafe")
	}

	// Point fixe T0 : après rotl/call-strip, des callers ne voient plus tls.
	// Max 8 rounds (profondeur d'appel ccgo typique << 8).
	for round := 0; round < 8; round++ {
		pure := findUnexportedT0(node)
		if len(pure) == 0 {
			break
		}
		elideUnexportedT0(node, pure)
	}

	// Purge déf. __ccgo_up si plus aucun site d'appel (rewrite structurel complet).
	dropUnusedCcgoUp(node)

	// Après élision tls T0 (et autres), purger les imports devenus morts.
	// Dogfood 20260810 : siphash/blake2b opt ne compilaient plus (libc unused).
	dropUnusedImports(fset, node)

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, node); err != nil {
		return nil, fmt.Errorf("formatting ast: %w", err)
	}

	return buf.Bytes(), nil
}

// matchCcgoUpStar reconnaît :
//
//	*(**T)(__ccgo_up(E))
//
// et produit :
//
//	(*T)(unsafe.Pointer(E))
//
// T peut être un Ident (uint64_t) ou un ArrayType ([16]u32), etc.
// Avec une étoile externe supplémentaire, le scalaire **(**T)(__ccgo_up(E))
// devient *(*T)(unsafe.Pointer(E)).
func matchCcgoUpStar(star *ast.StarExpr) (ast.Expr, bool) {
	call, ok := star.X.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return nil, false
	}
	elem, ok := typeElemOfDoubleStar(call.Fun)
	if !ok {
		return nil, false
	}
	upCall, ok := call.Args[0].(*ast.CallExpr)
	if !ok || len(upCall.Args) != 1 {
		return nil, false
	}
	upFun, ok := upCall.Fun.(*ast.Ident)
	if !ok || upFun.Name != "__ccgo_up" {
		return nil, false
	}
	addr := upCall.Args[0]
	return &ast.CallExpr{
		Fun: &ast.ParenExpr{
			X: &ast.StarExpr{X: elem},
		},
		Args: []ast.Expr{
			&ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   ast.NewIdent("unsafe"),
					Sel: ast.NewIdent("Pointer"),
				},
				Args: []ast.Expr{addr},
			},
		},
	}, true
}

// typeElemOfDoubleStar extrait T depuis l'expression de type (**T) d'une conversion.
func typeElemOfDoubleStar(fun ast.Expr) (ast.Expr, bool) {
	paren, ok := fun.(*ast.ParenExpr)
	if !ok {
		return nil, false
	}
	outer, ok := paren.X.(*ast.StarExpr)
	if !ok {
		return nil, false
	}
	inner, ok := outer.X.(*ast.StarExpr)
	if !ok {
		return nil, false
	}
	return inner.X, true
}

// matchArrayPtrIndex : (*(*[N]T)(p))[i] → (*[N]T)(p)[i]
// Après rewrite __ccgo_up, l'index sur valeur tableau garde une étoile de trop
// pour le style pointeur-tableau idiomatique Go.
func matchArrayPtrIndex(idx *ast.IndexExpr) (ast.Expr, bool) {
	base := idx.X
	if p, ok := base.(*ast.ParenExpr); ok {
		base = p.X
	}
	star, ok := base.(*ast.StarExpr)
	if !ok {
		return nil, false
	}
	call, ok := star.X.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return nil, false
	}
	// Fun = (*[N]T)  i.e. ParenExpr{ StarExpr{ ArrayType } }
	fun := call.Fun
	if p, ok := fun.(*ast.ParenExpr); ok {
		fun = p.X
	}
	st, ok := fun.(*ast.StarExpr)
	if !ok {
		return nil, false
	}
	if _, ok := st.X.(*ast.ArrayType); !ok {
		return nil, false
	}
	return &ast.IndexExpr{
		X: &ast.CallExpr{
			Fun: &ast.ParenExpr{
				X: &ast.StarExpr{X: st.X},
			},
			Args: call.Args,
		},
		Index: idx.Index,
	}, true
}

// Markers compte des motifs textuels sur une source Go (raw ou opt).
type Markers struct {
	CcgoUp      int
	BitsRotate  int
	RotlCalls   int
	TLSParamFn  int
	Lines       int
}

// CountMarkers scanne le source (pré/post TransformAST) pour métriques gen.
func CountMarkers(src []byte) Markers {
	s := string(src)
	lines := 0
	if len(src) > 0 {
		lines = strings.Count(s, "\n")
		if src[len(src)-1] != '\n' {
			lines++
		}
	}
	return Markers{
		CcgoUp:     strings.Count(s, "__ccgo_up"),
		BitsRotate: strings.Count(s, "bits.RotateLeft"),
		RotlCalls:  countRE(s, `\brotl32\s*\(|\brotl64\s*\(|\bL32\s*\(`),
		TLSParamFn: countRE(s, `func [A-Za-z0-9_]+\(tls \*libc\.TLS`),
		Lines:      lines,
	}
}

func countRE(s, pat string) int {
	// import minimal : éviter regexp global — scan naïf pour rotl/L32
	n := 0
	switch {
	case strings.Contains(pat, "rotl"):
		for _, p := range []string{"rotl32(", "rotl64(", "L32("} {
			n += strings.Count(s, p)
		}
	case strings.Contains(pat, "tls"):
		// approx : "func name(tls *libc.TLS"
		const needle = "(tls *libc.TLS"
		start := 0
		for {
			i := strings.Index(s[start:], needle)
			if i < 0 {
				break
			}
			// vérifier qu'on est sur une ligne func
			abs := start + i
			lineStart := strings.LastIndex(s[:abs], "\n") + 1
			if strings.Contains(s[lineStart:abs], "func ") {
				n++
			}
			start = abs + len(needle)
		}
	}
	return n
}

// dropUnusedCcgoUp supprime la définition locale func __ccgo_up si plus aucun appel.
func dropUnusedCcgoUp(file *ast.File) {
	calls := 0
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		id, ok := call.Fun.(*ast.Ident)
		if ok && id.Name == "__ccgo_up" {
			calls++
		}
		return true
	})
	if calls > 0 {
		return
	}
	out := file.Decls[:0]
	for _, d := range file.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if ok && fd.Name != nil && fd.Name.Name == "__ccgo_up" {
			continue
		}
		out = append(out, d)
	}
	file.Decls = out
}

// findUnexportedT0 : fonctions non exportées dont le 1er param est tls et
// dont le corps ne référence jamais l'ident tls.
func findUnexportedT0(file *ast.File) map[string]bool {
	pure := make(map[string]bool)
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Recv != nil || funcDecl.Type.Params == nil || len(funcDecl.Type.Params.List) == 0 {
			continue
		}
		if ast.IsExported(funcDecl.Name.Name) {
			continue
		}
		first := funcDecl.Type.Params.List[0]
		if len(first.Names) == 0 || first.Names[0].Name != "tls" {
			continue
		}
		usesTLS := false
		if funcDecl.Body != nil {
			ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
				if ident, ok := n.(*ast.Ident); ok && ident.Name == "tls" {
					usesTLS = true
					return false
				}
				return true
			})
		}
		if !usesTLS {
			pure[funcDecl.Name.Name] = true
		}
	}
	return pure
}

// elideUnexportedT0 retire le param tls et les args tls aux call sites.
func elideUnexportedT0(file *ast.File, pure map[string]bool) {
	astutil.Apply(file, func(cursor *astutil.Cursor) bool {
		n := cursor.Node()
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			if pure[funcDecl.Name.Name] && funcDecl.Type.Params != nil && len(funcDecl.Type.Params.List) > 0 {
				first := funcDecl.Type.Params.List[0]
				if len(first.Names) > 0 && first.Names[0].Name == "tls" {
					funcDecl.Type.Params.List = funcDecl.Type.Params.List[1:]
				}
			}
		}
		if call, ok := n.(*ast.CallExpr); ok {
			if ident, ok := call.Fun.(*ast.Ident); ok && pure[ident.Name] && len(call.Args) > 0 {
				if argIdent, ok := call.Args[0].(*ast.Ident); ok && argIdent.Name == "tls" {
					call.Args = call.Args[1:]
				}
			}
		}
		return true
	}, nil)
}

// dropUnusedImports retire les imports dont le nom de base n'apparaît plus
// hors clause d'import (ex. modernc.org/libc après strip total de tls).
func dropUnusedImports(fset *token.FileSet, file *ast.File) {
	used := map[string]bool{}
	ast.Inspect(file, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.SelectorExpr:
			if id, ok := x.X.(*ast.Ident); ok {
				used[id.Name] = true
			}
		case *ast.Ident:
			// types / valeurs référencés par nom de paquet (rare) : ignoré ;
			// le sélecteur couvre libc.TLS, bits.RotateLeft, unsafe.Pointer.
		}
		return true
	})
	var drop []string
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		name := importBaseName(imp, path)
		if name == "." || name == "_" {
			continue
		}
		if !used[name] {
			drop = append(drop, path)
		}
	}
	for _, path := range drop {
		astutil.DeleteImport(fset, file, path)
	}
}

func importBaseName(imp *ast.ImportSpec, path string) string {
	if imp.Name != nil {
		return imp.Name.Name
	}
	// dernier segment du chemin
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[i+1:]
	}
	return path
}

func extractIntVal(expr ast.Expr) int {
	if expr == nil {
		return 0
	}
	if lit, ok := expr.(*ast.BasicLit); ok && lit.Kind == token.INT {
		var v int
		fmt.Sscanf(lit.Value, "%d", &v)
		return v
	}
	if call, ok := expr.(*ast.CallExpr); ok {
		for _, arg := range call.Args {
			v := extractIntVal(arg)
			if v > 0 {
				return v
			}
		}
	}
	if paren, ok := expr.(*ast.ParenExpr); ok {
		return extractIntVal(paren.X)
	}
	// ident numérique rare (après fmt.Sprintf inject) — ignoré
	return 0
}

func extractSubWidth(expr ast.Expr) int {
	if subExpr, ok := expr.(*ast.BinaryExpr); ok && subExpr.Op == token.SUB {
		return extractIntVal(subExpr.X)
	}
	if call, ok := expr.(*ast.CallExpr); ok {
		for _, arg := range call.Args {
			w := extractSubWidth(arg)
			if w > 0 {
				return w
			}
		}
	}
	if paren, ok := expr.(*ast.ParenExpr); ok {
		return extractSubWidth(paren.X)
	}
	return 0
}

// matchUintptrNegConst détecte uintptr(-N) / uintptr(-int32(N)) → magnitude N > 0.
func matchUintptrNegConst(expr ast.Expr) (mag int, ok bool) {
	call, ok := expr.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return 0, false
	}
	fun, ok := call.Fun.(*ast.Ident)
	if !ok || fun.Name != "uintptr" {
		return 0, false
	}
	arg := call.Args[0]
	// -N or -int32(N) or -libc.Int32FromInt32(N)
	unary, ok := arg.(*ast.UnaryExpr)
	if !ok || unary.Op != token.SUB {
		return 0, false
	}
	n := extractIntVal(unary.X)
	if n <= 0 {
		return 0, false
	}
	return n, true
}

// matchRotateBinary détecte ROL/ROR sous | ou ^.
// Retourne bits.RotateLeft{32,64}(x, int(k)) ou (_, false).
func matchRotateBinary(bin *ast.BinaryExpr) (*ast.CallExpr, bool) {
	left, okL := bin.X.(*ast.BinaryExpr)
	right, okR := bin.Y.(*ast.BinaryExpr)
	if !okL || !okR {
		return nil, false
	}

	var val ast.Expr
	var rotAmount, width int

	switch {
	// ROL : (x << N) op (x >> (W-N))
	case left.Op == token.SHL && right.Op == token.SHR:
		n := extractIntVal(left.Y)
		w := extractSubWidth(right.Y)
		if n <= 0 || w < 32 || !exprEqual(left.X, right.X) {
			return nil, false
		}
		val, rotAmount, width = left.X, n, w

	// ROR : (x >> N) op (x << (W-N)) → RotateLeft(x, W-N)
	// ou ROL inversé : (x >> (W-N)) op (x << N)
	case left.Op == token.SHR && right.Op == token.SHL:
		if !exprEqual(left.X, right.X) {
			return nil, false
		}
		wRight := extractSubWidth(right.Y) // (W-N) côté SHL → ROR
		wLeft := extractSubWidth(left.Y)   // (W-N) côté SHR → ROL inv
		switch {
		case wRight >= 32 && wLeft == 0:
			n := extractIntVal(left.Y)
			if n <= 0 {
				return nil, false
			}
			rotAmount = wRight - n
			if rotAmount <= 0 || rotAmount >= wRight {
				return nil, false
			}
			val, width = left.X, wRight
		case wLeft >= 32 && wRight == 0:
			n := extractIntVal(right.Y)
			if n <= 0 {
				return nil, false
			}
			val, rotAmount, width = left.X, n, wLeft
		default:
			return nil, false
		}

	default:
		return nil, false
	}

	selName := "RotateLeft32"
	switch width {
	case 64:
		selName = "RotateLeft64"
	case 32:
		selName = "RotateLeft32"
	default:
		return nil, false
	}

	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent("bits"),
			Sel: ast.NewIdent(selName),
		},
		Args: []ast.Expr{
			val,
			&ast.CallExpr{
				Fun:  ast.NewIdent("int"),
				Args: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: fmt.Sprintf("%d", rotAmount)}},
			},
		},
	}, true
}

// exprEqual compare deux expressions AST pour l'égalité structurelle (sous-ensemble ccgo).
func exprEqual(a, b ast.Expr) bool {
	if a == nil || b == nil {
		return a == b
	}
	// unwrap parens
	for {
		if p, ok := a.(*ast.ParenExpr); ok {
			a = p.X
			continue
		}
		break
	}
	for {
		if p, ok := b.(*ast.ParenExpr); ok {
			b = p.X
			continue
		}
		break
	}
	switch x := a.(type) {
	case *ast.Ident:
		y, ok := b.(*ast.Ident)
		return ok && x.Name == y.Name
	case *ast.BasicLit:
		y, ok := b.(*ast.BasicLit)
		return ok && x.Kind == y.Kind && x.Value == y.Value
	case *ast.SelectorExpr:
		y, ok := b.(*ast.SelectorExpr)
		return ok && exprEqual(x.X, y.X) && x.Sel.Name == y.Sel.Name
	case *ast.IndexExpr:
		y, ok := b.(*ast.IndexExpr)
		return ok && exprEqual(x.X, y.X) && exprEqual(x.Index, y.Index)
	case *ast.BinaryExpr:
		y, ok := b.(*ast.BinaryExpr)
		return ok && x.Op == y.Op && exprEqual(x.X, y.X) && exprEqual(x.Y, y.Y)
	case *ast.CallExpr:
		y, ok := b.(*ast.CallExpr)
		if !ok || !exprEqual(x.Fun, y.Fun) || len(x.Args) != len(y.Args) {
			return false
		}
		for i := range x.Args {
			if !exprEqual(x.Args[i], y.Args[i]) {
				return false
			}
		}
		return true
	case *ast.StarExpr:
		y, ok := b.(*ast.StarExpr)
		return ok && exprEqual(x.X, y.X)
	case *ast.UnaryExpr:
		y, ok := b.(*ast.UnaryExpr)
		return ok && x.Op == y.Op && exprEqual(x.X, y.X)
	case *ast.ParenExpr:
		return exprEqual(x.X, b)
	default:
		return false
	}
}

// TransformRotations est l'alias de compatibilité pour le préprocesseur c2simd-gen
func TransformRotations(src []byte) ([]byte, error) {
	return TransformAST(src)
}
