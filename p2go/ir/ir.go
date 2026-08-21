// Package ir — passe 3 de p2go : abaissement de l'AST front en IR structurée
// désucrée. Les noms de variables deviennent des slots indexés fixes ; les
// affectations composées et incréments deviennent des Assign explicites ; les
// conditions sont typées bool (coercition int != 0 explicite).
package ir

import (
	"strings"

	"code.hazyhaar.fr/devhoros/c2simd/p2go/front"
	"code.hazyhaar.fr/devhoros/c2simd/p2go/types"
)

type Program struct {
	Funcs []*Func
	Main  *Func // corps top-level, Name "main"
}

type Func struct {
	Name         string
	Params       int // les Params premiers slots
	SlotNames    []string
	SlotKinds    []types.SlotKind
	ReturnsValue bool
	ReturnKind   types.SlotKind
	Body         []Stmt
}

type Stmt interface{ stmt() }

type Assign struct {
	Slot int
	Expr Expr
}
type Echo struct {
	Args []Expr // Str ou expression int
}
type If struct {
	Cond Expr // bool
	Then []Stmt
	Else []Stmt
}
type While struct {
	Cond Expr // bool, nil = true
	Body []Stmt
}
type For struct {
	Init Stmt // nil ou *Assign
	Cond Expr // bool, nil = true
	Post Stmt // nil ou *Assign
	Body []Stmt
}
type Return struct {
	Expr Expr // nil pour void
}
type CallStmt struct {
	Call *Call
}
type ArrAssign struct { // a = []int64{…}
	Slot  int
	Elems []Expr
}
type IndexAssign struct { // a[idx] = e
	Slot int
	Idx  Expr
	Expr Expr
}

// SumLoop est la forme vectorisable produite par rules (F-p2go-simd-sum-reduction) :
// acc += somme(arr). Émise via l'helper SIMD/scalaire dual.
type SumLoop struct {
	Acc int // slot int accumulé (+=)
	Arr int // slot tableau réduit
}

// DotLoop — F-p2go-simd-dot : acc += Σ a[i]*b[i] (mod 2⁶⁴, wraparound int64).
type DotLoop struct {
	Acc  int
	A, B int
}

// MinMaxLoop — F-p2go-simd-minmax : dst = min/max(dst, éléments de arr).
type MinMaxLoop struct {
	Dst   int
	Arr   int
	IsMax bool
}

type Switch struct { // switch Go natif (cases sans fallthrough)
	Subject Expr
	Cases   []SwitchCase
	Default []Stmt // nil = absent
}
type SwitchCase struct {
	Vals []Expr
	Body []Stmt
}
type ArrPush struct { // array_push($a, v) → a = append(a, v)
	Slot int
	Val  Expr
}
type ArrPop struct { // $v = array_pop($a) → v = a[len-1] ; a = a[:len-1]
	Dst int
	Arr int
}
type Break struct{}    // break Go (boucle ou switch le plus proche)
type Continue struct{} // continue Go (boucle la plus proche)

func (*Assign) stmt()      {}
func (*Echo) stmt()        {}
func (*If) stmt()          {}
func (*While) stmt()       {}
func (*For) stmt()         {}
func (*Return) stmt()      {}
func (*CallStmt) stmt()    {}
func (*ArrAssign) stmt()   {}
func (*IndexAssign) stmt() {}
func (*SumLoop) stmt()     {}
func (*DotLoop) stmt()     {}
func (*MinMaxLoop) stmt()  {}
func (*Switch) stmt()      {}
func (*ArrPush) stmt()     {}
func (*ArrPop) stmt()      {}
func (*Break) stmt()       {}
func (*Continue) stmt()    {}

type Expr interface{ expr() }

type Const struct{ Value int64 }
type Str struct{ Value string } // uniquement en Echo.Args
type Slot struct{ Index int }
type Bin struct { // int×int→int (arith) ou int×int→bool (comparaisons)
	Op   string
	L, R Expr
}
type Logic struct { // bool×bool→bool : && ||
	Op   string
	L, R Expr
}
type Not struct{ X Expr }    // bool→bool
type Neg struct{ X Expr }    // int→int
type BitNot struct{ X Expr } // int→int : ~x PHP → ^x Go
type Call struct {
	Name string
	Args []Expr
}
type Index struct { // a[idx] en lecture
	Slot int
	Idx  Expr
}
type Count struct{ Slot int }      // int64(len(a))
type ItoS struct{ X Expr }         // conversion int → string en concaténation
type StrLen struct{ X Expr }       // int64(len(s)) — builtin strlen
type ArrCopy struct{ X Expr }      // copie défensive au return d'un tableau (sémantique PHP)
type ArrLit struct{ Elems []Expr } // littéral []int64{…} en position argument
type Builtin struct {              // builtin PHP émis inline ou via helper p2go*
	Name string // lowercase
	Args []Expr
}

func (*Const) expr()   {}
func (*Str) expr()     {}
func (*Slot) expr()    {}
func (*Bin) expr()     {}
func (*Logic) expr()   {}
func (*Not) expr()     {}
func (*Neg) expr()     {}
func (*BitNot) expr()  {}
func (*Call) expr()    {}
func (*Index) expr()   {}
func (*Count) expr()   {}
func (*ItoS) expr()    {}
func (*StrLen) expr()  {}
func (*ArrCopy) expr() {}
func (*ArrLit) expr()  {}
func (*Builtin) expr() {}

// Lower abaisse un programme typé en IR. Ne peut plus échouer : tout écart a
// été refusé par front et types. Entièrement réentrant et thread-safe.
func Lower(prog *front.Program, info *types.Info) *Program {
	out := &Program{}
	for _, fn := range prog.Funcs {
		fi := info.Funcs[strings.ToLower(fn.Name)]
		out.Funcs = append(out.Funcs, lowerFunc(fi, fn.Body))
	}
	out.Main = lowerFunc(info.Main, prog.Main)
	return out
}

func lowerFunc(fi *types.FuncInfo, body []front.Stmt) *Func {
	f := &Func{
		Name:         fi.Name,
		Params:       fi.Params,
		SlotNames:    fi.Slots,
		SlotKinds:    fi.SlotKinds,
		ReturnsValue: fi.ReturnsValue,
		ReturnKind:   fi.ReturnKind,
	}
	f.Body = lowerStmts(fi, body)
	return f
}

func lowerStmts(fi *types.FuncInfo, body []front.Stmt) []Stmt {
	var out []Stmt
	for _, st := range body {
		// F-p2go-do-while : do{B}while(c) désucré en B ; while(c){B}
		// (Go n'a pas de do-while ; chaque abaissement produit des nœuds neufs,
		// la duplication du corps est sûre).
		if dw, ok := st.(*front.DoWhile); ok {
			out = append(out, lowerStmts(fi, dw.Body)...)
			out = append(out, &While{Cond: lowerCond(fi, dw.Cond), Body: lowerStmts(fi, dw.Body)})
			continue
		}
		if bl, ok := st.(*front.Block); ok { // produit de désucrage : aplati
			out = append(out, lowerStmts(fi, bl.Stmts)...)
			continue
		}
		out = append(out, lowerStmt(fi, st))
	}
	return out
}

func lowerStmt(fi *types.FuncInfo, st front.Stmt) Stmt {
	switch s := st.(type) {
	case *front.Assign:
		slot := fi.SlotOf[s.Name]
		if call, isCall := s.Expr.(*front.Call); isCall && strings.ToLower(call.Name) == "array_pop" {
			return &ArrPop{Dst: slot, Arr: fi.SlotOf[call.Args[0].(*front.Var).Name]}
		}
		// $b = $a tableau : copie réelle (sémantique valeur PHP).
		if v, isVar := s.Expr.(*front.Var); isVar && s.Op == "=" &&
			fi.SlotKinds[fi.SlotOf[v.Name]] == types.KindArr {
			return &Assign{Slot: slot, Expr: &ArrCopy{X: &Slot{Index: fi.SlotOf[v.Name]}}}
		}
		if lit, isArr := s.Expr.(*front.ArrLit); isArr {
			a := &ArrAssign{Slot: slot}
			for _, el := range lit.Elems {
				a.Elems = append(a.Elems, lowerExpr(fi, el))
			}
			return a
		}
		if s.Op == ".=" { // $s .= e → s = s . e (RHS int converti ItoS)
			return &Assign{Slot: slot, Expr: &Bin{Op: ".", L: &Slot{Index: slot}, R: lowerConcatOperand(fi, s.Expr)}}
		}
		rhs := lowerExpr(fi, s.Expr)
		if s.Op != "=" { // $x += e → x = x <op> e
			rhs = &Bin{Op: strings.TrimSuffix(s.Op, "="), L: &Slot{Index: slot}, R: rhs}
		}
		return &Assign{Slot: slot, Expr: rhs}
	case *front.IncDec:
		slot := fi.SlotOf[s.Name]
		op := "+"
		if s.Op == "--" {
			op = "-"
		}
		return &Assign{Slot: slot, Expr: &Bin{Op: op, L: &Slot{Index: slot}, R: &Const{Value: 1}}}
	case *front.Echo:
		e := &Echo{}
		for _, a := range s.Args {
			if str, ok := a.(*front.StrLit); ok {
				e.Args = append(e.Args, &Str{Value: str.Value})
				continue
			}
			e.Args = append(e.Args, lowerExpr(fi, a))
		}
		return e
	case *front.If:
		return &If{
			Cond: lowerCond(fi, s.Cond),
			Then: lowerStmts(fi, s.Then),
			Else: lowerStmts(fi, s.Else),
		}
	case *front.While:
		return &While{Cond: lowerCond(fi, s.Cond), Body: lowerStmts(fi, s.Body)}
	case *front.For:
		f := &For{Body: lowerStmts(fi, s.Body)}
		if s.Init != nil {
			f.Init = lowerStmt(fi, s.Init)
		}
		if s.Cond != nil {
			f.Cond = lowerCond(fi, s.Cond)
		}
		if s.Post != nil {
			f.Post = lowerStmt(fi, s.Post)
		}
		return f
	case *front.Return:
		if s.Expr == nil {
			return &Return{}
		}
		if fi.ReturnKind == types.KindArr {
			// Copie défensive : PHP retourne par valeur, un slice Go partagerait.
			return &Return{Expr: &ArrCopy{X: lowerExpr(fi, s.Expr)}}
		}
		return &Return{Expr: lowerExpr(fi, s.Expr)}
	case *front.IndexAssign:
		return &IndexAssign{
			Slot: fi.SlotOf[s.Name],
			Idx:  lowerExpr(fi, s.Idx),
			Expr: lowerExpr(fi, s.Expr),
		}
	case *front.Break:
		return &Break{}
	case *front.Continue:
		return &Continue{}
	case *front.ExprStmt:
		c := s.Expr.(*front.Call)
		if strings.ToLower(c.Name) == "array_push" {
			return &ArrPush{
				Slot: fi.SlotOf[c.Args[0].(*front.Var).Name],
				Val:  lowerExpr(fi, c.Args[1]),
			}
		}
		return &CallStmt{Call: lowerCall(fi, c)}
	case *front.Switch:
		sw := &Switch{Subject: lowerExpr(fi, s.Subject)}
		for _, c := range s.Cases {
			var vals []Expr
			for _, v := range c.Vals {
				vals = append(vals, lowerExpr(fi, v))
			}
			sw.Cases = append(sw.Cases, SwitchCase{Vals: vals, Body: lowerStmts(fi, c.Body)})
		}
		if s.Default != nil {
			sw.Default = lowerStmts(fi, s.Default)
			if sw.Default == nil {
				sw.Default = []Stmt{}
			}
		}
		return sw
	}
	panic("ir: statement front inconnu")
}

func lowerCall(fi *types.FuncInfo, c *front.Call) *Call {
	out := &Call{Name: strings.ToLower(c.Name)}
	for _, a := range c.Args {
		out.Args = append(out.Args, lowerExpr(fi, a))
	}
	return out
}

// lowerCallExpr abaisse un appel en contexte valeur ; les builtins se plient
// en opérations natives (F-p2go-intdiv-builtin : intdiv(a,b) → a / b, les
// deux tronquent vers zéro).
func lowerCallExpr(fi *types.FuncInfo, c *front.Call) Expr {
	key := strings.ToLower(c.Name)
	switch key {
	case "intdiv":
		return &Bin{Op: "/", L: lowerExpr(fi, c.Args[0]), R: lowerExpr(fi, c.Args[1])}
	case "count":
		return &Count{Slot: fi.SlotOf[c.Args[0].(*front.Var).Name]}
	case "strlen":
		return &StrLen{X: lowerExpr(fi, c.Args[0])}
	case "floor", "ceil", "round": // identité sur int (le subset n'a pas de float)
		return lowerExpr(fi, c.Args[0])
	}
	if types.IsBuiltin(key) {
		b := &Builtin{Name: key}
		for _, a := range c.Args {
			b.Args = append(b.Args, lowerExpr(fi, a))
		}
		return b
	}
	return lowerCall(fi, c)
}

// isStrExpr — kind string d'une expression déjà validée par types :
// littéral, slot string, concaténation, builtin ou fonction retournant string.
func isStrExpr(fi *types.FuncInfo, e front.Expr) bool {
	switch x := e.(type) {
	case *front.StrLit:
		return true
	case *front.Var:
		return fi.SlotKinds[fi.SlotOf[x.Name]] == types.KindStr
	case *front.Binary:
		return x.Op == "."
	case *front.Call:
		key := strings.ToLower(x.Name)
		if types.BuiltinReturnsString(key) {
			return true
		}
		if fi.ProgInfo != nil {
			if callee, ok := fi.ProgInfo.Funcs[key]; ok {
				return callee.ReturnKind == types.KindStr
			}
		}
	}
	return false
}

// lowerConcatOperand abaisse un opérande de « . » : un int se convertit
// explicitement (ItoS), sémantique PHP de la concaténation mixte.
func lowerConcatOperand(fi *types.FuncInfo, e front.Expr) Expr {
	low := lowerExpr(fi, e)
	if isStrExpr(fi, e) {
		return low
	}
	return &ItoS{X: low}
}

// lowerExpr abaisse une expression en contexte int.
func lowerExpr(fi *types.FuncInfo, e front.Expr) Expr {
	switch x := e.(type) {
	case *front.IntLit:
		return &Const{Value: x.Value}
	case *front.Var:
		return &Slot{Index: fi.SlotOf[x.Name]}
	case *front.Unary:
		switch x.Op {
		case "-":
			return &Neg{X: lowerExpr(fi, x.X)}
		case "~":
			return &BitNot{X: lowerExpr(fi, x.X)}
		}
		panic("ir: ! en contexte int (refusé par types)")
	case *front.StrLit:
		return &Str{Value: x.Value}
	case *front.Binary:
		if x.Op == "." {
			return &Bin{Op: ".", L: lowerConcatOperand(fi, x.L), R: lowerConcatOperand(fi, x.R)}
		}
		return &Bin{Op: x.Op, L: lowerExpr(fi, x.L), R: lowerExpr(fi, x.R)}
	case *front.Call:
		return lowerCallExpr(fi, x)
	case *front.Index:
		return &Index{Slot: fi.SlotOf[x.Name], Idx: lowerExpr(fi, x.Idx)}
	case *front.ArrLit:
		out := &ArrLit{}
		for _, el := range x.Elems {
			out.Elems = append(out.Elems, lowerExpr(fi, el))
		}
		return out
	}
	panic("ir: expression front inconnue en contexte int")
}

// lowerCond abaisse une expression en contexte bool : les comparaisons et la
// logique passent telles quelles, un int e devient e != 0 (truthiness PHP).
func lowerCond(fi *types.FuncInfo, e front.Expr) Expr {
	switch x := e.(type) {
	case *front.Call:
		if strings.ToLower(x.Name) == "in_array" { // builtin déjà bool en Go
			return lowerCallExpr(fi, x)
		}
	case *front.Unary:
		if x.Op == "!" {
			return &Not{X: lowerCond(fi, x.X)}
		}
	case *front.Binary:
		switch x.Op {
		case "&&", "||":
			return &Logic{Op: x.Op, L: lowerCond(fi, x.L), R: lowerCond(fi, x.R)}
		case "==", "!=", "<", "<=", ">", ">=":
			return &Bin{Op: x.Op, L: lowerExpr(fi, x.L), R: lowerExpr(fi, x.R)}
		}
	}
	return &Bin{Op: "!=", L: lowerExpr(fi, e), R: &Const{Value: 0}}
}
