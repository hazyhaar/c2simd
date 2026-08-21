// Package types — passe 2 de p2go : inférence de types stricts (int64 seul en
// v0.1) et assignation de slots indexés fixes par fonction. Aplatit la table de
// symboles PHP avant l'emit : aucun lookup dynamique dans le code généré.
package types

import (
	"fmt"
	"strings"

	"code.hazyhaar.fr/devhoros/c2simd/p2go/front"
)

// FuncInfo décrit une fonction typée : slots fixes (params d'abord) et
// présence d'une valeur de retour.
// SlotKind est le type aplati d'un slot : int64 ou []int64 (v0.2 partiel).
type SlotKind int

const (
	KindInt SlotKind = iota
	KindArr
	KindStr
)

type FuncInfo struct {
	Name         string
	Params       int      // les Params premiers slots (kind fixé par hint, int par défaut)
	Slots        []string // index de slot → nom PHP
	SlotKinds    []SlotKind
	SlotOf       map[string]int
	ReturnsValue bool
	ReturnKind   SlotKind    // KindInt par défaut ; string/array par hint « : type »
	Fn           *front.Func // nil pour le pseudo-main
	ProgInfo     *Info       // référence au programme parent (évite l'état global mutable)
}

// Info est le résultat de la passe sur un programme complet.
type Info struct {
	Funcs map[string]*FuncInfo // clef = nom de fonction (lowercase, PHP insensible)
	Order []string             // ordre de déclaration
	Main  *FuncInfo
}

// Check type l'AST : slots, arité des appels, strings confinées à echo,
// cohérence des return. Fail-loud sur tout écart (codes err_* de front.Err).
func Check(prog *front.Program) (*Info, error) {
	info := &Info{Funcs: map[string]*FuncInfo{}}
	for _, fn := range prog.Funcs {
		key := strings.ToLower(fn.Name)
		if _, dup := info.Funcs[key]; dup {
			return nil, errf("err_parse", fn.Line, "fonction %q redéclarée", fn.Name)
		}
		fi := &FuncInfo{Name: fn.Name, Params: len(fn.Params), SlotOf: map[string]int{}, Fn: fn}
		for i, p := range fn.Params {
			if _, dup := fi.SlotOf[p]; dup {
				return nil, errf("err_parse", fn.Line, "paramètre $%s dupliqué", p)
			}
			fi.SlotOf[p] = len(fi.Slots)
			fi.Slots = append(fi.Slots, p)
			fi.SlotKinds = append(fi.SlotKinds, hintKind(fn.ParamHints[i]))
		}
		fi.ReturnKind = hintKind(fn.RetHint)
		info.Funcs[key] = fi
		info.Order = append(info.Order, key)
	}
	// Deux temps : signatures et présence de retour d'abord (appels avant
	// déclaration légaux en PHP), corps ensuite.
	for _, key := range info.Order {
		fi := info.Funcs[key]
		hasVal, hasBare := scanReturns(fi.Fn.Body)
		if hasVal && hasBare {
			return nil, errf("err_parse", fi.Fn.Line, "fonction %s mélange return valué et return nu", fi.Name)
		}
		if fi.Fn.RetHint != "" && !hasVal {
			return nil, errf("err_parse", fi.Fn.Line, "fonction %s annotée : %s mais sans return valué", fi.Name, fi.Fn.RetHint)
		}
		fi.ReturnsValue = hasVal
	}
	for _, key := range info.Order {
		fi := info.Funcs[key]
		if err := checkBody(fi, fi.Fn.Body, info); err != nil {
			return nil, err
		}
	}
	info.Main = &FuncInfo{Name: "main", SlotOf: map[string]int{}}
	for _, fi := range info.Funcs {
		fi.ProgInfo = info
	}
	info.Main.ProgInfo = info
	if err := checkBody(info.Main, prog.Main, info); err != nil {
		return nil, err
	}
	return info, nil
}

func errf(code string, line int, format string, args ...any) *front.Err {
	return &front.Err{Code: code, Line: line, Msg: fmt.Sprintf(format, args...)}
}

// hintKind traduit un hint PHP en kind de slot ("" = int par défaut).
func hintKind(hint string) SlotKind {
	switch hint {
	case "string":
		return KindStr
	case "array":
		return KindArr
	}
	return KindInt
}

// valKind — kind de slot correspondant à un kind d'expression scalaire.
func valKind(k kind) SlotKind {
	if k == kindStr {
		return KindStr
	}
	return KindInt
}

func (fi *FuncInfo) slot(name string) int {
	if s, ok := fi.SlotOf[name]; ok {
		return s
	}
	fi.SlotOf[name] = len(fi.Slots)
	fi.Slots = append(fi.Slots, name)
	fi.SlotKinds = append(fi.SlotKinds, KindInt)
	return len(fi.Slots) - 1
}

// setKind fixe le type du slot ; un slot déjà typé autrement est refusé
// (aucune variable polymorphe dans le subset).
func (fi *FuncInfo) setKind(name string, k SlotKind, line int, seen map[string]bool) *front.Err {
	s := fi.slot(name)
	if seen[name] && fi.SlotKinds[s] != k {
		return errf("err_parse", line, "$%s change de type (int ↔ array) — hors subset", name)
	}
	fi.SlotKinds[s] = k
	return nil
}

func (fi *FuncInfo) kindOf(name string) SlotKind {
	return fi.SlotKinds[fi.slot(name)]
}

// checkArrWritable — sémantique de COPIE PHP : un paramètre tableau est en
// lecture seule (le muter en Go muterait le tableau de l'appelant, ce que PHP
// ne fait pas). F-p2go-array-signatures.
func (fi *FuncInfo) checkArrWritable(name string, line int) *front.Err {
	if s, ok := fi.SlotOf[name]; ok && s < fi.Params {
		return errf("err_parse", line, "écriture dans le paramètre tableau $%s — sémantique de copie PHP non imitée (param en lecture seule)", name)
	}
	return nil
}

func (k SlotKind) String() string {
	switch k {
	case KindArr:
		return "array"
	case KindStr:
		return "string"
	}
	return "int"
}

func checkBody(fi *FuncInfo, body []front.Stmt, info *Info) error {
	assigned := map[string]bool{}
	for i := 0; i < fi.Params; i++ {
		assigned[fi.Slots[i]] = true
	}
	return checkStmts(fi, body, info, assigned)
}

func scanReturns(body []front.Stmt) (hasVal, hasBare bool) {
	for _, st := range body {
		switch s := st.(type) {
		case *front.Return:
			if s.Expr != nil {
				hasVal = true
			} else {
				hasBare = true
			}
		case *front.If:
			v, b := scanReturns(s.Then)
			hasVal, hasBare = hasVal || v, hasBare || b
			v, b = scanReturns(s.Else)
			hasVal, hasBare = hasVal || v, hasBare || b
		case *front.While:
			v, b := scanReturns(s.Body)
			hasVal, hasBare = hasVal || v, hasBare || b
		case *front.DoWhile:
			v, b := scanReturns(s.Body)
			hasVal, hasBare = hasVal || v, hasBare || b
		case *front.For:
			v, b := scanReturns(s.Body)
			hasVal, hasBare = hasVal || v, hasBare || b
		case *front.Switch:
			for _, c := range s.Cases {
				v, b := scanReturns(c.Body)
				hasVal, hasBare = hasVal || v, hasBare || b
			}
			v, b := scanReturns(s.Default)
			hasVal, hasBare = hasVal || v, hasBare || b
		case *front.Block:
			v, b := scanReturns(s.Stmts)
			hasVal, hasBare = hasVal || v, hasBare || b
		}
	}
	return
}

func checkStmts(fi *FuncInfo, body []front.Stmt, info *Info, assigned map[string]bool) error {
	for _, st := range body {
		if err := checkStmt(fi, st, info, assigned); err != nil {
			return err
		}
	}
	return nil
}

func checkStmt(fi *FuncInfo, st front.Stmt, info *Info, assigned map[string]bool) error {
	switch s := st.(type) {
	case *front.Assign:
		if lit, isArr := s.Expr.(*front.ArrLit); isArr {
			// Littéral de tableau : seul RHS légal d'un = simple (v0.2).
			if s.Op != "=" {
				return errf("err_parse", s.Line, "littéral de tableau en composé — hors subset")
			}
			if err := fi.checkArrWritable(s.Name, s.Line); err != nil {
				return err
			}
			for _, el := range lit.Elems {
				if err := checkExpr(fi, el, info, assigned, false); err != nil {
					return err
				}
			}
			if err := fi.setKind(s.Name, KindArr, s.Line, assigned); err != nil {
				return err
			}
			assigned[s.Name] = true
			return nil
		}
		// $b = $a — copie de tableau var-à-var (sémantique valeur PHP : copie
		// réelle émise, jamais un partage de slice).
		if v, isVar := s.Expr.(*front.Var); isVar && s.Op == "=" && assigned[v.Name] && fi.kindOf(v.Name) == KindArr {
			if err := fi.checkArrWritable(s.Name, s.Line); err != nil {
				return err
			}
			if err := fi.setKind(s.Name, KindArr, s.Line, assigned); err != nil {
				return err
			}
			assigned[s.Name] = true
			return nil
		}
		// $b = f() où f retourne un tableau (hint : array), builtin tableau, ou array_pop.
		if call, isCall := s.Expr.(*front.Call); isCall && s.Op == "=" {
			key := strings.ToLower(call.Name)
			if key == "array_pop" { // $v = array_pop($a) — mutation + valeur
				if len(call.Args) != 1 {
					return errf("err_parse", s.Line, "arité : array_pop attend 1 argument")
				}
				av, ok := call.Args[0].(*front.Var)
				if !ok || !assigned[av.Name] || fi.kindOf(av.Name) != KindArr {
					return errf("err_parse", s.Line, "array_pop attend une $variable tableau affectée")
				}
				if err := fi.checkArrWritable(av.Name, s.Line); err != nil {
					return err
				}
				if err := fi.setKind(s.Name, KindInt, s.Line, assigned); err != nil {
					return err
				}
				assigned[s.Name] = true
				return nil
			}
			retArr := false
			if callee, known := info.Funcs[key]; known && callee.ReturnKind == KindArr {
				if _, err := checkUserCall(fi, call, info, assigned); err != nil {
					return err
				}
				retArr = true
			} else if sig, isB := builtinSigs[key]; isB && sig.ret == KindArr && !sig.retBool {
				if err := checkBuiltinArgs(fi, call, sig, info, assigned); err != nil {
					return err
				}
				retArr = true
			}
			if retArr {
				if err := fi.checkArrWritable(s.Name, s.Line); err != nil {
					return err
				}
				if err := fi.setKind(s.Name, KindArr, s.Line, assigned); err != nil {
					return err
				}
				assigned[s.Name] = true
				return nil
			}
		}
		rk, err := checkValExpr(fi, s.Expr, info, assigned)
		if err != nil {
			return err
		}
		if s.Op != "=" && !assigned[s.Name] {
			return errf("err_parse", s.Line, "$%s utilisée en composé avant affectation", s.Name)
		}
		switch s.Op {
		case "=": // le kind du RHS fixe le slot
			want := KindInt
			if rk == kindStr {
				want = KindStr
			}
			if err := fi.setKind(s.Name, want, s.Line, assigned); err != nil {
				return err
			}
		case ".=": // concaténation en place : slot string, RHS int|string
			if fi.kindOf(s.Name) != KindStr {
				return errf("err_parse", s.Line, "$%s .= sur un slot non-string", s.Name)
			}
		default: // composés arithmétiques : slot et RHS int
			if fi.kindOf(s.Name) != KindInt || rk != kindInt {
				return errf("err_parse", s.Line, "composé %s sur kind non-int", s.Op)
			}
		}
		assigned[s.Name] = true
	case *front.IndexAssign:
		if !assigned[s.Name] {
			return errf("err_parse", s.Line, "$%s indexée avant affectation", s.Name)
		}
		if fi.kindOf(s.Name) != KindArr {
			return errf("err_parse", s.Line, "$%s indexée mais n'est pas un tableau", s.Name)
		}
		if err := fi.checkArrWritable(s.Name, s.Line); err != nil {
			return err
		}
		if err := checkExpr(fi, s.Idx, info, assigned, false); err != nil {
			return err
		}
		if err := checkExpr(fi, s.Expr, info, assigned, false); err != nil {
			return err
		}
	case *front.IncDec:
		if !assigned[s.Name] {
			return errf("err_parse", s.Line, "$%s incrémentée avant affectation", s.Name)
		}
		if fi.kindOf(s.Name) != KindInt {
			return errf("err_parse", s.Line, "$%s incrémentée mais n'est pas int", s.Name)
		}
	case *front.Echo:
		for _, a := range s.Args {
			if _, err := checkValExpr(fi, a, info, assigned); err != nil {
				return err
			}
		}
	case *front.If:
		if err := checkExpr(fi, s.Cond, info, assigned, true); err != nil {
			return err
		}
		// Flow-insensible v0.1 : les affectations de branches comptent pour la suite.
		if err := checkStmts(fi, s.Then, info, assigned); err != nil {
			return err
		}
		if err := checkStmts(fi, s.Else, info, assigned); err != nil {
			return err
		}
	case *front.While:
		if err := checkExpr(fi, s.Cond, info, assigned, true); err != nil {
			return err
		}
		if err := checkStmts(fi, s.Body, info, assigned); err != nil {
			return err
		}
	case *front.DoWhile:
		// Corps AVANT condition : les affectations du corps portent la cond.
		if err := checkStmts(fi, s.Body, info, assigned); err != nil {
			return err
		}
		if err := checkExpr(fi, s.Cond, info, assigned, true); err != nil {
			return err
		}
	case *front.For:
		if s.Init != nil {
			if err := checkStmt(fi, s.Init, info, assigned); err != nil {
				return err
			}
		}
		if s.Cond != nil {
			if err := checkExpr(fi, s.Cond, info, assigned, true); err != nil {
				return err
			}
		}
		if err := checkStmts(fi, s.Body, info, assigned); err != nil {
			return err
		}
		if s.Post != nil {
			if err := checkStmt(fi, s.Post, info, assigned); err != nil {
				return err
			}
		}
	case *front.Return:
		if fi.Fn == nil && s.Expr != nil {
			return errf("err_parse", s.Line, "return valué au top-level hors subset")
		}
		if s.Expr != nil {
			switch fi.ReturnKind {
			case KindArr:
				return checkArrArg(fi, s.Expr, info, assigned)
			case KindStr:
				k, err := checkValExpr(fi, s.Expr, info, assigned)
				if err != nil {
					return err
				}
				if k != kindStr {
					return errf("err_parse", s.Line, "return %s dans une fonction : string", k)
				}
				return nil
			default:
				return checkIntExpr(fi, s.Expr, info, assigned)
			}
		}
	case *front.ExprStmt:
		// Appel en statement : le résultat éventuel est jeté, void légal ici.
		call := s.Expr.(*front.Call)
		if strings.ToLower(call.Name) == "array_push" { // array_push($a, $v) : mutation
			if len(call.Args) != 2 {
				return errf("err_parse", s.Line, "arité : array_push attend 2 arguments (variadique : v0.5)")
			}
			av, ok := call.Args[0].(*front.Var)
			if !ok || !assigned[av.Name] || fi.kindOf(av.Name) != KindArr {
				return errf("err_parse", s.Line, "array_push attend une $variable tableau affectée")
			}
			if err := fi.checkArrWritable(av.Name, s.Line); err != nil {
				return err
			}
			return checkIntExpr(fi, call.Args[1], info, assigned)
		}
		if _, err := checkUserCall(fi, call, info, assigned); err != nil {
			return err
		}
	case *front.Block:
		return checkStmts(fi, s.Stmts, info, assigned)
	case *front.Switch:
		sk, err := checkValExpr(fi, s.Subject, info, assigned)
		if err != nil {
			return err
		}
		for _, c := range s.Cases {
			for _, v := range c.Vals {
				vk, err := checkValExpr(fi, v, info, assigned)
				if err != nil {
					return err
				}
				if vk != sk {
					return errf("err_parse", s.Line, "case de kind %s sur sujet %s — kinds mêlés hors subset", vk, sk)
				}
			}
			if err := checkStmts(fi, c.Body, info, assigned); err != nil {
				return err
			}
		}
		if err := checkStmts(fi, s.Default, info, assigned); err != nil {
			return err
		}
	}
	return nil
}

// checkExpr — compat des sites historiques : contexte int strict quand
// inCond=false, contexte condition (bool|int) quand inCond=true.
func checkExpr(fi *FuncInfo, e front.Expr, info *Info, assigned map[string]bool, inCond bool) error {
	if inCond {
		return checkCondExpr(fi, e, info, assigned)
	}
	return checkIntExpr(fi, e, info, assigned)
}

// checkIntExpr exige une expression de kind int.
func checkIntExpr(fi *FuncInfo, e front.Expr, info *Info, assigned map[string]bool) error {
	k, err := exprKind(fi, e, info, assigned)
	if err != nil {
		return err
	}
	if k != kindInt {
		return errf("err_parse", exprLine(e), "expression %s en contexte int hors subset", k)
	}
	return nil
}

// checkValExpr accepte int ou string (RHS d'affectation, argument d'echo) et
// retourne le kind constaté ; bool refusé.
func checkValExpr(fi *FuncInfo, e front.Expr, info *Info, assigned map[string]bool) (kind, error) {
	k, err := exprKind(fi, e, info, assigned)
	if err != nil {
		return k, err
	}
	if k == kindBool {
		return k, errf("err_parse", exprLine(e), "expression booléenne en contexte valeur hors subset")
	}
	return k, nil
}

// checkCondExpr accepte bool (tel quel) ou int (coercé != 0 à l'IR) ; string
// refusée — la truthiness PHP des strings ("" ET "0" falsy) est un piège que
// le subset n'imite pas silencieusement.
func checkCondExpr(fi *FuncInfo, e front.Expr, info *Info, assigned map[string]bool) error {
	k, err := exprKind(fi, e, info, assigned)
	if err != nil {
		return err
	}
	if k == kindStr {
		return errf("err_parse", exprLine(e), "string en condition — truthiness PHP ('0' falsy) hors subset, comparer explicitement")
	}
	return nil
}

// builtinSig — signature typée d'un builtin PHP supporté : kinds des
// arguments (SlotKind ; KindArr = argument tableau) et kind de retour.
type builtinSig struct {
	args    []SlotKind
	ret     SlotKind
	retBool bool // in_array : bool, contexte condition seul
}

// Builtins en expression. array_push (statement) et array_pop (RHS direct)
// sont traités à part — mutation de tableau hors expression.
var builtinSigs = map[string]builtinSig{
	"intdiv":        {args: []SlotKind{KindInt, KindInt}, ret: KindInt},
	"strlen":        {args: []SlotKind{KindStr}, ret: KindInt},
	"abs":           {args: []SlotKind{KindInt}, ret: KindInt},
	"min":           {args: []SlotKind{KindInt, KindInt}, ret: KindInt},
	"max":           {args: []SlotKind{KindInt, KindInt}, ret: KindInt},
	"pow":           {args: []SlotKind{KindInt, KindInt}, ret: KindInt},
	"floor":         {args: []SlotKind{KindInt}, ret: KindInt}, // identité sur int
	"ceil":          {args: []SlotKind{KindInt}, ret: KindInt},
	"round":         {args: []SlotKind{KindInt}, ret: KindInt},
	"ord":           {args: []SlotKind{KindStr}, ret: KindInt},
	"chr":           {args: []SlotKind{KindInt}, ret: KindStr},
	"substr":        {args: []SlotKind{KindStr, KindInt, KindInt}, ret: KindStr},
	"str_replace":   {args: []SlotKind{KindStr, KindStr, KindStr}, ret: KindStr},
	"trim":          {args: []SlotKind{KindStr}, ret: KindStr},
	"strtoupper":    {args: []SlotKind{KindStr}, ret: KindStr},
	"strtolower":    {args: []SlotKind{KindStr}, ret: KindStr},
	"strpos":        {args: []SlotKind{KindStr, KindStr}, ret: KindInt}, // SENTINELLE -1, jamais false (F-p2go-strpos-sentinel)
	"array_reverse": {args: []SlotKind{KindArr}, ret: KindArr},
	"array_slice":   {args: []SlotKind{KindArr, KindInt, KindInt}, ret: KindArr},
	"array_fill":    {args: []SlotKind{KindInt, KindInt, KindInt}, ret: KindArr},
	"in_array":      {args: []SlotKind{KindInt, KindArr}, ret: KindInt, retBool: true},
}

// IsBuiltin expose la table des builtins en expression (consommée par ir/).
func IsBuiltin(name string) bool {
	_, ok := builtinSigs[name]
	return ok
}

// BuiltinReturnsString — vrai si le builtin retourne un kind string
// (décision ItoS de la concaténation à l'abaissement).
func BuiltinReturnsString(name string) bool {
	sig, ok := builtinSigs[name]
	return ok && !sig.retBool && sig.ret == KindStr
}

// checkBuiltinArgs vérifie les arguments d'un builtin contre sa signature.
func checkBuiltinArgs(fi *FuncInfo, x *front.Call, sig builtinSig, info *Info, assigned map[string]bool) error {
	if len(x.Args) != len(sig.args) {
		return errf("err_parse", x.Line, "arité : builtin %s attend %d arguments", strings.ToLower(x.Name), len(sig.args))
	}
	for i, a := range x.Args {
		want := sig.args[i]
		if want == KindArr {
			if err := checkArrArg(fi, a, info, assigned); err != nil {
				return err
			}
			continue
		}
		k, err := exprKind(fi, a, info, assigned)
		if err != nil {
			return err
		}
		if k == kindBool || valKind(k) != want {
			return errf("err_parse", x.Line, "builtin %s : argument %d de kind %s, %s attendu", strings.ToLower(x.Name), i+1, k, want)
		}
	}
	// array_fill : les clefs PHP démarrent au premier argument — seul 0 donne
	// un tableau indexé dense compatible []int64.
	if strings.ToLower(x.Name) == "array_fill" {
		if lit, ok := x.Args[0].(*front.IntLit); !ok || lit.Value != 0 {
			return errf("err_parse", x.Line, "array_fill : start littéral 0 obligatoire (clefs denses)")
		}
	}
	return nil
}

type kind int

const (
	kindInt kind = iota
	kindBool
	kindStr
)

func (k kind) String() string {
	switch k {
	case kindBool:
		return "booléenne"
	case kindStr:
		return "string"
	}
	return "int"
}

func exprLine(e front.Expr) int {
	switch x := e.(type) {
	case *front.IntLit:
		return x.Line
	case *front.StrLit:
		return x.Line
	case *front.Var:
		return x.Line
	case *front.Unary:
		return x.Line
	case *front.Binary:
		return x.Line
	case *front.Call:
		return x.Line
	}
	return 0
}

func exprKind(fi *FuncInfo, e front.Expr, info *Info, assigned map[string]bool) (kind, error) {
	switch x := e.(type) {
	case *front.IntLit:
		return kindInt, nil
	case *front.StrLit:
		return kindStr, nil
	case *front.Var:
		if !assigned[x.Name] {
			return kindInt, errf("err_parse", x.Line, "$%s lue avant toute affectation", x.Name)
		}
		switch fi.kindOf(x.Name) {
		case KindArr:
			return kindInt, errf("err_parse", x.Line, "tableau $%s en contexte scalaire (passage/copie de tableaux : v0.3)", x.Name)
		case KindStr:
			return kindStr, nil
		}
		return kindInt, nil
	case *front.Index:
		if !assigned[x.Name] {
			return kindInt, errf("err_parse", x.Line, "$%s indexée avant toute affectation", x.Name)
		}
		if fi.kindOf(x.Name) != KindArr {
			return kindInt, errf("err_parse", x.Line, "$%s indexée mais n'est pas un tableau", x.Name)
		}
		if err := checkExpr(fi, x.Idx, info, assigned, false); err != nil {
			return kindInt, err
		}
		return kindInt, nil
	case *front.ArrLit:
		return kindInt, errf("err_parse", x.Line, "littéral de tableau hors affectation directe — hors subset")
	case *front.Unary:
		k, err := exprKind(fi, x.X, info, assigned)
		if err != nil {
			return kindInt, err
		}
		if x.Op == "!" {
			if k == kindStr {
				return kindBool, errf("err_parse", x.Line, "! sur string — truthiness PHP hors subset")
			}
			return kindBool, nil // opérande int ou bool, coercé à l'IR
		}
		if k != kindInt {
			return kindInt, errf("err_parse", x.Line, "unaire %s sur %s hors subset", x.Op, k)
		}
		return kindInt, nil
	case *front.Binary:
		lk, err := exprKind(fi, x.L, info, assigned)
		if err != nil {
			return kindInt, err
		}
		rk, err := exprKind(fi, x.R, info, assigned)
		if err != nil {
			return kindInt, err
		}
		switch x.Op {
		case "&&", "||":
			if lk == kindStr || rk == kindStr {
				return kindBool, errf("err_parse", x.Line, "string en opérande logique — truthiness PHP hors subset")
			}
			return kindBool, nil // opérandes int coercés (!= 0) à l'IR
		case ".": // concaténation : int et string se mêlent, résultat string
			if lk == kindBool || rk == kindBool {
				return kindStr, errf("err_parse", x.Line, "booléen en concaténation hors subset")
			}
			return kindStr, nil
		case "==", "!=":
			if lk == kindStr && rk == kindStr {
				return kindBool, nil // égalité de strings, native Go
			}
			if lk != kindInt || rk != kindInt {
				return kindBool, errf("err_parse", x.Line, "égalité entre kinds %s et %s hors subset", lk, rk)
			}
			return kindBool, nil
		case "<", "<=", ">", ">=":
			if lk == kindStr && rk == kindStr {
				return kindBool, nil // ordre lexicographique octet à octet (strcmp PHP = Go)
			}
			if lk != kindInt || rk != kindInt {
				return kindBool, errf("err_parse", x.Line, "comparaison ordonnée entre kinds %s et %s hors subset", lk, rk)
			}
			return kindBool, nil
		default: // arithmétique
			if lk != kindInt || rk != kindInt {
				return kindInt, errf("err_parse", x.Line, "arithmétique sur %s hors subset", lk)
			}
			return kindInt, nil
		}
	case *front.Call:
		key := strings.ToLower(x.Name)
		if key == "count" { // count($a) : argument tableau obligatoire
			if len(x.Args) != 1 {
				return kindInt, errf("err_parse", x.Line, "arité : count attend 1 argument")
			}
			v, isVar := x.Args[0].(*front.Var)
			if !isVar {
				return kindInt, errf("err_parse", x.Line, "count attend une $variable tableau")
			}
			if !assigned[v.Name] || fi.kindOf(v.Name) != KindArr {
				return kindInt, errf("err_parse", x.Line, "count sur $%s qui n'est pas un tableau affecté", v.Name)
			}
			return kindInt, nil
		}
		if key == "array_push" || key == "array_pop" {
			return kindInt, errf("err_parse", x.Line, "%s hors expression (array_push : statement seul ; array_pop : RHS direct d'affectation)", key)
		}
		if key == "array_map" {
			return kindInt, errf("err_parse", x.Line, "array_map hors subset — callables non supportés, écrire la boucle for/foreach explicite")
		}
		// min/max variadiques (≥ 2 arguments int) — PHP accepte n args.
		if (key == "min" || key == "max") && len(x.Args) > 2 {
			for _, a := range x.Args {
				if err := checkIntExpr(fi, a, info, assigned); err != nil {
					return kindInt, err
				}
			}
			return kindInt, nil
		}
		// Builtins v0.1+ : signatures typées (builtinSigs).
		if sig, isBuiltin := builtinSigs[key]; isBuiltin {
			if err := checkBuiltinArgs(fi, x, sig, info, assigned); err != nil {
				return kindInt, err
			}
			if sig.retBool {
				return kindBool, nil
			}
			switch sig.ret {
			case KindArr:
				return kindInt, errf("err_parse", x.Line, "builtin %s retournant un tableau en contexte scalaire", key)
			case KindStr:
				return kindStr, nil
			}
			return kindInt, nil
		}
		callee, err := checkUserCall(fi, x, info, assigned)
		if err != nil {
			return kindInt, err
		}
		if !callee.ReturnsValue {
			return kindInt, errf("err_parse", x.Line, "fonction void %s en contexte valeur", x.Name)
		}
		switch callee.ReturnKind {
		case KindArr:
			return kindInt, errf("err_parse", x.Line, "appel %s retournant un tableau en contexte scalaire", x.Name)
		case KindStr:
			return kindStr, nil
		}
		return kindInt, nil
	}
	return kindInt, errf("err_parse", 0, "expression inconnue")
}

// checkUserCall valide lookup, arité et kind de chaque argument contre la
// signature (hints) de la fonction utilisateur appelée.
func checkUserCall(fi *FuncInfo, x *front.Call, info *Info, assigned map[string]bool) (*FuncInfo, error) {
	callee, ok := info.Funcs[strings.ToLower(x.Name)]
	if !ok {
		return nil, errf("err_parse", x.Line, "fonction inconnue %s", x.Name)
	}
	if len(x.Args) != callee.Params {
		return nil, errf("err_parse", x.Line, "arité : %s attend %d arguments", x.Name, callee.Params)
	}
	for i, a := range x.Args {
		want := callee.SlotKinds[i]
		if want == KindArr {
			if err := checkArrArg(fi, a, info, assigned); err != nil {
				return nil, err
			}
			continue
		}
		k, err := exprKind(fi, a, info, assigned)
		if err != nil {
			return nil, err
		}
		if valKind(k) != want || k == kindBool {
			return nil, errf("err_parse", x.Line, "argument %d de %s : kind %s, %s attendu (hint)", i+1, x.Name, k, want)
		}
	}
	return callee, nil
}

// checkArrArg valide une expression attendue de kind tableau : $variable
// tableau ou appel retournant un tableau.
func checkArrArg(fi *FuncInfo, a front.Expr, info *Info, assigned map[string]bool) error {
	switch v := a.(type) {
	case *front.Var:
		if !assigned[v.Name] || fi.kindOf(v.Name) != KindArr {
			return errf("err_parse", v.Line, "$%s attendue tableau affecté", v.Name)
		}
		return nil
	case *front.ArrLit: // littéral passé directement : f([1,2,3])
		for _, el := range v.Elems {
			if err := checkIntExpr(fi, el, info, assigned); err != nil {
				return err
			}
		}
		return nil
	case *front.Call:
		if sig, isB := builtinSigs[strings.ToLower(v.Name)]; isB {
			if sig.ret != KindArr || sig.retBool {
				return errf("err_parse", v.Line, "builtin %s ne retourne pas un tableau", v.Name)
			}
			return checkBuiltinArgs(fi, v, sig, info, assigned)
		}
		callee, err := checkUserCall(fi, v, info, assigned)
		if err != nil {
			return err
		}
		if callee.ReturnKind != KindArr {
			return errf("err_parse", v.Line, "appel %s ne retourne pas un tableau", v.Name)
		}
		return nil
	}
	return errf("err_parse", exprLine(a), "expression tableau attendue ($var ou appel : array)")
}
