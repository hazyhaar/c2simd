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

// TransformAST applique de manière déterministe les optimisations d'AST sur le C transpilé :
// 1. rotl32 -> bits.RotateLeft32 / rotl64 -> bits.RotateLeft64
// 2. Simplification des wrappers libc identité pour littéraux (Constant Folding P1.2)
// 3. Conversion des motifs (x << N | x >> (W-N)) -> bits.RotateLeft32 / RotateLeft64 (Passe P1.5)
// 4. Réécriture de corps load32_le/store32_le (Passe P1.4 en unsafe.Pointer direct)
// 5. Passe P2.1 : Élision du paramètre tls *libc.TLS pour les fonctions pures (Analyse d'effets T0/T1)
func TransformAST(src []byte) ([]byte, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "src.go", src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parsing ast: %w", err)
	}

	deadCodeSymbols := make(map[string]bool)
	for _, rule := range ArchtimeRulesTable {
		if rule.DeadCode {
			deadCodeSymbols[rule.Symbol] = true
		}
	}

	pureTLSFuncs := make(map[string]bool)
	// Identification T0 des fonctions qui n'utilisent jamais tls
	for _, decl := range node.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			if funcDecl.Recv == nil && len(funcDecl.Type.Params.List) > 0 {
				firstParam := funcDecl.Type.Params.List[0]
				if len(firstParam.Names) > 0 && firstParam.Names[0].Name == "tls" {
					usesTLS := false
					ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
						if ident, ok := n.(*ast.Ident); ok && ident.Name == "tls" {
							usesTLS = true
							return false
						}
						return true
					})
					if !usesTLS {
						pureTLSFuncs[funcDecl.Name.Name] = true
					}
				}
			}
		}
	}

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

			if ident, ok := call.Fun.(*ast.Ident); ok && (ident.Name == "rotl32" || ident.Name == "rotl64") {
				var valArg, distArg ast.Expr
				if len(call.Args) >= 3 {
					valArg = call.Args[1]
					distArg = call.Args[2]
				} else if len(call.Args) == 2 {
					valArg = call.Args[0]
					distArg = call.Args[1]
				}

				if valArg != nil && distArg != nil {
					distExpr := distArg
					if innerCall, ok := distArg.(*ast.CallExpr); ok {
						if innerIdent, ok := innerCall.Fun.(*ast.Ident); ok && (innerIdent.Name == "uint32" || innerIdent.Name == "u32" || innerIdent.Name == "int32" || innerIdent.Name == "int64") {
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
					if ident.Name == "rotl64" {
						selName = "RotateLeft64"
					}

					newCall := &ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X:   ast.NewIdent("bits"),
							Sel: ast.NewIdent(selName),
						},
						Args: []ast.Expr{
							valArg,
							intDistExpr,
						},
					}

					cursor.Replace(newCall)
					modifiedBits = true
					return true
				}
			}
		}

		// 4. Passe P1.5 : Conversion des motifs (x << int32(N) | x >> (int32(W) - int32(N))) -> bits.RotateLeft32 / RotateLeft64
		if binaryExpr, ok := n.(*ast.BinaryExpr); ok && binaryExpr.Op == token.OR {
			if shlExpr, ok := binaryExpr.X.(*ast.BinaryExpr); ok && shlExpr.Op == token.SHL {
				if shrExpr, ok := binaryExpr.Y.(*ast.BinaryExpr); ok && shrExpr.Op == token.SHR {
					shiftVal := extractIntVal(shlExpr.Y)
					widthVal := extractSubWidth(shrExpr.Y)

					if shiftVal > 0 {
						valX := shlExpr.X
						selName := "RotateLeft32"
						if widthVal == 64 {
							selName = "RotateLeft64"
						}

						newCall := &ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X:   ast.NewIdent("bits"),
								Sel: ast.NewIdent(selName),
							},
							Args: []ast.Expr{
								valX,
								&ast.CallExpr{
									Fun:  ast.NewIdent("int"),
									Args: []ast.Expr{ast.NewIdent(fmt.Sprintf("%d", shiftVal))},
								},
							},
						}
						cursor.Replace(newCall)
						modifiedBits = true
						return true
					}
				}
			}
		}

		return true
	}, nil)

	if modifiedBits {
		astutil.AddImport(fset, node, "math/bits")
	}
	if modifiedUnsafe {
		astutil.AddImport(fset, node, "unsafe")
	}

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, node); err != nil {
		return nil, fmt.Errorf("formatting ast: %w", err)
	}

	return buf.Bytes(), nil
}

func extractIntVal(expr ast.Expr) int {
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
	return 32
}

// TransformRotations est l'alias de compatibilité pour le préprocesseur c2simd-gen
func TransformRotations(src []byte) ([]byte, error) {
	return TransformAST(src)
}
