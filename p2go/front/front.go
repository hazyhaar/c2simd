// Package front — passe 1 de p2go : lexer + parser du subset PHP v0.1,
// whitelist fail-loud. Tout construct hors subset est rejeté avec un code
// err_* explicite, la ligne et le lexème fautif (SPEC.md §2).
package front

import (
	"fmt"
	"strconv"
	"strings"
)

// Err est l'erreur fail-loud typée du front.
type Err struct {
	Code string // err_eval, err_parse, ...
	Line int
	Msg  string
}

func (e *Err) Error() string {
	return fmt.Sprintf("%s: ligne %d: %s", e.Code, e.Line, e.Msg)
}

func errf(code string, line int, format string, args ...any) *Err {
	return &Err{Code: code, Line: line, Msg: fmt.Sprintf(format, args...)}
}

// --- AST ---

type Program struct {
	Funcs []*Func
	Main  []Stmt // statements top-level (corps du main généré)
}

type Func struct {
	Name       string
	Params     []string
	ParamHints []string // "" (int par défaut), "int", "string", "array"
	RetHint    string   // "" (int par défaut si retour valué), "int", "string", "array"
	Body       []Stmt
	Line       int
}

type Stmt interface{ stmt() }

type Assign struct {
	Name string // sans le $
	Op   string // "=", "+=", "-=", "*=", "/=", "%="
	Expr Expr
	Line int
}
type IncDec struct {
	Name string
	Op   string // "++" ou "--"
	Line int
}
type IndexAssign struct { // $a[idx] = e;
	Name string
	Idx  Expr
	Expr Expr
	Line int
}
type Echo struct {
	Args []Expr
	Line int
}
type If struct {
	Cond Expr
	Then []Stmt
	Else []Stmt // nil, ou 1 If pour elseif, ou bloc else
	Line int
}
type While struct {
	Cond Expr
	Body []Stmt
	Line int
}
type DoWhile struct { // do { body } while (cond); — corps exécuté au moins une fois
	Cond Expr
	Body []Stmt
	Line int
}
type For struct {
	Init Stmt // nil, Assign ou IncDec
	Cond Expr // nil = boucle infinie
	Post Stmt // nil, Assign ou IncDec
	Body []Stmt
	Line int
}
type Return struct {
	Expr Expr // nil pour return;
	Line int
}
type ExprStmt struct {
	Expr Expr // appel de fonction en statement
	Line int
}
type Break struct{ Line int }     // break de boucle (ou de switch, plus proche englobant)
type Continue struct{ Line int }  // continue de boucle
type Block struct{ Stmts []Stmt } // séquence plate (produit de désucrage), aplatie à l'IR

type Switch struct { // switch strict : chaque case non vide finit par break/return
	Subject Expr
	Cases   []SwitchCase
	Default []Stmt // nil si absent
	Line    int
}
type SwitchCase struct {
	Vals []Expr // plusieurs valeurs = cases vides empilés (case A: case B: corps)
	Body []Stmt
}

func (*Assign) stmt()      {}
func (*IncDec) stmt()      {}
func (*IndexAssign) stmt() {}
func (*Echo) stmt()        {}
func (*If) stmt()          {}
func (*While) stmt()       {}
func (*DoWhile) stmt()     {}
func (*For) stmt()         {}
func (*Return) stmt()      {}
func (*ExprStmt) stmt()    {}
func (*Switch) stmt()      {}
func (*Break) stmt()       {}
func (*Continue) stmt()    {}
func (*Block) stmt()       {}

type Expr interface{ expr() }

type IntLit struct {
	Value int64
	Line  int
}
type StrLit struct { // uniquement en argument d'echo (types/ le vérifie)
	Value string
	Line  int
}
type Var struct {
	Name string
	Line int
}
type Unary struct {
	Op   string // "-", "!"
	X    Expr
	Line int
}
type Binary struct {
	Op   string
	L, R Expr
	Line int
}
type Call struct {
	Name string
	Args []Expr
	Line int
}
type ArrLit struct { // [e1, e2, …] — tableau indexé homogène int (v0.2 partiel)
	Elems []Expr
	Line  int
}
type Index struct { // $a[idx] en lecture
	Name string
	Idx  Expr
	Line int
}
type Ternary struct { // c ? a : b — désucré par hoisting avant types (A nil = forme ?:)
	Cond, A, B Expr
	Line       int
}
type Match struct { // match (subj) { v1, v2 => e, default => e } — désucré en Switch
	Subject Expr
	Arms    []MatchArm
	Default Expr
	Line    int
}
type MatchArm struct {
	Vals   []Expr
	Result Expr
}

func (*IntLit) expr()  {}
func (*StrLit) expr()  {}
func (*Var) expr()     {}
func (*Unary) expr()   {}
func (*Binary) expr()  {}
func (*Call) expr()    {}
func (*ArrLit) expr()  {}
func (*Index) expr()   {}
func (*Ternary) expr() {}
func (*Match) expr()   {}

// --- Lexer ---

type tokKind int

const (
	tkEOF tokKind = iota
	tkIdent
	tkVar // $name
	tkInt
	tkStr
	tkOp // ponctuation et opérateurs
)

type token struct {
	kind tokKind
	text string
	ival int64
	line int
}

type lexer struct {
	src  string
	pos  int
	line int
	toks []token
}

func lex(src string) ([]token, *Err) {
	l := &lexer{src: src, line: 1}
	for l.pos < len(l.src) {
		c := l.src[l.pos]
		switch {
		case c == '\n':
			l.line++
			l.pos++
		case c == ' ' || c == '\t' || c == '\r':
			l.pos++
		case c == '/' && l.peek(1) == '/':
			l.skipLineComment()
		case c == '#':
			l.skipLineComment()
		case c == '/' && l.peek(1) == '*':
			if err := l.skipBlockComment(); err != nil {
				return nil, err
			}
		case c == '$':
			if l.peek(1) == '$' {
				return nil, errf("err_varvar", l.line, "variable variable $$ hors subset")
			}
			if err := l.lexVar(); err != nil {
				return nil, err
			}
		case c >= '0' && c <= '9':
			if err := l.lexNumber(); err != nil {
				return nil, err
			}
		case c == '"' || c == '\'':
			if err := l.lexString(c); err != nil {
				return nil, err
			}
		case isIdentStart(c):
			l.lexIdent()
		default:
			if err := l.lexOp(); err != nil {
				return nil, err
			}
		}
	}
	l.toks = append(l.toks, token{kind: tkEOF, line: l.line})
	return l.toks, nil
}

func (l *lexer) peek(n int) byte {
	if l.pos+n < len(l.src) {
		return l.src[l.pos+n]
	}
	return 0
}

func (l *lexer) skipLineComment() {
	for l.pos < len(l.src) && l.src[l.pos] != '\n' {
		l.pos++
	}
}

func (l *lexer) skipBlockComment() *Err {
	start := l.line
	l.pos += 2
	for l.pos < len(l.src) {
		if l.src[l.pos] == '\n' {
			l.line++
		}
		if l.src[l.pos] == '*' && l.peek(1) == '/' {
			l.pos += 2
			return nil
		}
		l.pos++
	}
	return errf("err_parse", start, "commentaire /* non terminé")
}

func isIdentStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
func isIdentPart(c byte) bool { return isIdentStart(c) || (c >= '0' && c <= '9') }

func (l *lexer) lexVar() *Err {
	l.pos++ // $
	start := l.pos
	for l.pos < len(l.src) && isIdentPart(l.src[l.pos]) {
		l.pos++
	}
	if l.pos == start {
		return errf("err_parse", l.line, "$ sans nom de variable")
	}
	l.toks = append(l.toks, token{kind: tkVar, text: l.src[start:l.pos], line: l.line})
	return nil
}

func (l *lexer) lexIdent() {
	start := l.pos
	for l.pos < len(l.src) && isIdentPart(l.src[l.pos]) {
		l.pos++
	}
	l.toks = append(l.toks, token{kind: tkIdent, text: l.src[start:l.pos], line: l.line})
}

func (l *lexer) lexNumber() *Err {
	// F-p2go-hex-literals : 0x… lu en base 16 (masques CRC32/Murmur3 etc.),
	// parsé en non-signé puis réinterprété int64 (0xFFFFFFFFFFFFFFFF légal).
	if l.src[l.pos] == '0' && (l.peek(1) == 'x' || l.peek(1) == 'X') {
		l.pos += 2
		start := l.pos
		for l.pos < len(l.src) && isHexDigit(l.src[l.pos]) {
			l.pos++
		}
		if l.pos == start {
			return errf("err_parse", l.line, "littéral hexadécimal vide")
		}
		u, err := strconv.ParseUint(l.src[start:l.pos], 16, 64)
		if err != nil {
			return errf("err_parse", l.line, "littéral hexadécimal invalide %q", l.src[start:l.pos])
		}
		l.toks = append(l.toks, token{kind: tkInt, ival: int64(u), line: l.line})
		return nil
	}
	start := l.pos
	for l.pos < len(l.src) && l.src[l.pos] >= '0' && l.src[l.pos] <= '9' {
		l.pos++
	}
	if l.pos < len(l.src) && l.src[l.pos] == '.' {
		return errf("err_float", l.line, "littéral flottant hors subset v0.1")
	}
	v, err := strconv.ParseInt(l.src[start:l.pos], 10, 64)
	if err != nil {
		return errf("err_parse", l.line, "littéral entier invalide %q", l.src[start:l.pos])
	}
	l.toks = append(l.toks, token{kind: tkInt, ival: v, line: l.line})
	return nil
}

func isHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// lexString lit une string. En double quote, l'interpolation $ident est
// DÉSUCRÉE au lexer (F-p2go-string-interp) : "a $x b" émet les tokens
// ( "a " . $x . " b" ) et réutilise l'opérateur de concaténation. Les formes
// ${…} et {$…} restent refusées err_interp.
func (l *lexer) lexString(quote byte) *Err {
	startLine := l.line
	l.pos++
	var b strings.Builder
	var vars []string  // variables interpolées, dans l'ordre
	var parts []string // segments littéraux : len(vars)+1 segments
	for l.pos < len(l.src) {
		c := l.src[l.pos]
		switch {
		case c == quote:
			l.pos++
			parts = append(parts, b.String())
			l.emitStringTokens(parts, vars, startLine)
			return nil
		case c == '\\':
			esc := l.peek(1)
			l.pos += 2
			if quote == '\'' {
				switch esc {
				case '\'', '\\':
					b.WriteByte(esc)
				default: // PHP : \x littéral en simple quote
					b.WriteByte('\\')
					b.WriteByte(esc)
				}
				continue
			}
			switch esc {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case '\\':
				b.WriteByte('\\')
			case '"':
				b.WriteByte('"')
			case '$':
				b.WriteByte('$')
			default:
				return errf("err_parse", startLine, "escape \\%c non supporté", esc)
			}
		case c == '$' && quote == '"':
			if !isIdentStart(l.peek(1)) { // ${…} ou $ nu
				return errf("err_interp", startLine, "interpolation ${…} hors subset ($ident seul)")
			}
			l.pos++
			start := l.pos
			for l.pos < len(l.src) && isIdentPart(l.src[l.pos]) {
				l.pos++
			}
			parts = append(parts, b.String())
			b.Reset()
			vars = append(vars, l.src[start:l.pos])
		case c == '{' && l.peek(1) == '$' && quote == '"':
			return errf("err_interp", startLine, "interpolation {$…} hors subset ($ident seul)")
		case c == '\n':
			l.line++
			b.WriteByte(c)
			l.pos++
		default:
			b.WriteByte(c)
			l.pos++
		}
	}
	return errf("err_parse", startLine, "string non terminée")
}

// emitStringTokens émet soit un tkStr simple, soit la forme désucrée
// parenthésée ( s0 . $v1 . s1 … ) — les segments vides sont omis sauf s0
// quand la string commence par une variable (force le kind string).
func (l *lexer) emitStringTokens(parts []string, vars []string, line int) {
	if len(vars) == 0 {
		l.toks = append(l.toks, token{kind: tkStr, text: parts[0], line: line})
		return
	}
	op := func(text string) { l.toks = append(l.toks, token{kind: tkOp, text: text, line: line}) }
	str := func(s string) { l.toks = append(l.toks, token{kind: tkStr, text: s, line: line}) }
	op("(")
	str(parts[0]) // conservé même vide : ancre le kind string
	for i, v := range vars {
		op(".")
		l.toks = append(l.toks, token{kind: tkVar, text: v, line: line})
		if parts[i+1] != "" {
			op(".")
			str(parts[i+1])
		}
	}
	op(")")
}

var multiOps = []string{
	// les plus longs d'abord (matching par préfixe) : <<= avant <<, ** avant *=
	"<?php", "?>", "===", "!==", "<<=", ">>=", "==", "!=", "<<", ">>", "<=", ">=",
	"&&", "||", "**", "&=", "|=", "^=",
	"+=", "-=", "*=", "/=", "%=", ".=", "++", "--", "->", "::", "=>",
}

func (l *lexer) lexOp() *Err {
	for _, op := range multiOps {
		if strings.HasPrefix(l.src[l.pos:], op) {
			l.toks = append(l.toks, token{kind: tkOp, text: op, line: l.line})
			l.pos += len(op)
			return nil
		}
	}
	c := l.src[l.pos]
	if strings.IndexByte("+-*/%<>=!(){},;[]&|^~.?:", c) >= 0 {
		l.toks = append(l.toks, token{kind: tkOp, text: string(c), line: l.line})
		l.pos++
		return nil
	}
	return errf("err_parse", l.line, "caractère inattendu %q", string(c))
}

// --- Parser ---

type parser struct {
	toks        []token
	pos         int
	feN         int // compteur gensym des foreach désucrés
	loopDepth   int // boucles englobantes (break/continue légaux)
	switchDepth int // corps de case englobants (break y est légal)
}

// parseLoopBody parse un corps de boucle avec la profondeur incrémentée.
func (p *parser) parseLoopBody() ([]Stmt, error) {
	p.loopDepth++
	body, err := p.parseBlockOrStmt()
	p.loopDepth--
	return body, err
}

// Parse lexe et parse une source PHP complète (avec <?php en tête).
func Parse(src string) (*Program, error) {
	toks, lerr := lex(src)
	if lerr != nil {
		return nil, lerr
	}
	p := &parser{toks: toks}
	if !p.eatOp("<?php") {
		return nil, errf("err_parse", 1, "balise <?php attendue en tête de fichier")
	}
	prog := &Program{}
	for {
		t := p.cur()
		if t.kind == tkEOF {
			break
		}
		if t.kind == tkOp && t.text == "?>" {
			p.pos++
			if p.cur().kind != tkEOF {
				return nil, errf("err_parse", p.cur().line, "contenu après ?> hors subset")
			}
			break
		}
		if t.kind == tkIdent && t.text == "function" {
			fn, err := p.parseFunc()
			if err != nil {
				return nil, err
			}
			prog.Funcs = append(prog.Funcs, fn)
			continue
		}
		st, err := p.parseStmt()
		if err != nil {
			return nil, err
		}
		prog.Main = append(prog.Main, st)
	}
	if len(prog.Main) == 0 && len(prog.Funcs) == 0 {
		return nil, errf("err_empty", 1, "aucun statement ni fonction harvestable")
	}
	// Désucrage ternaire/match par hoisting (desugar.go) — avant types/.
	d := &desugarer{}
	for _, fn := range prog.Funcs {
		body, derr := d.stmts(fn.Body)
		if derr != nil {
			return nil, derr
		}
		fn.Body = body
	}
	mainBody, derr := d.stmts(prog.Main)
	if derr != nil {
		return nil, derr
	}
	prog.Main = mainBody
	return prog, nil
}

func (p *parser) cur() token  { return p.toks[p.pos] }
func (p *parser) next() token { t := p.toks[p.pos]; p.pos++; return t }

func (p *parser) eatOp(text string) bool {
	if t := p.cur(); t.kind == tkOp && t.text == text {
		p.pos++
		return true
	}
	return false
}

func (p *parser) expectOp(text string) *Err {
	if !p.eatOp(text) {
		t := p.cur()
		return errf("err_parse", t.line, "%q attendu, trouvé %q", text, t.text)
	}
	return nil
}

// forbiddenKeywords : mot-clé hors subset → code de refus (SPEC.md §2).
var forbiddenKeywords = map[string]string{
	"eval": "err_eval", "global": "err_global", "static": "err_global",
	"include": "err_include", "require": "err_include",
	"include_once": "err_include", "require_once": "err_include",
	"class": "err_oop", "new": "err_oop", "interface": "err_oop",
	"trait": "err_oop", "extends": "err_oop",
	"array": "err_array", "list": "err_array",
}

func (p *parser) checkForbidden(t token) *Err {
	if t.kind == tkIdent {
		if code, bad := forbiddenKeywords[strings.ToLower(t.text)]; bad {
			return errf(code, t.line, "%q hors subset v0.1", t.text)
		}
	}
	if t.kind == tkOp {
		switch t.text {
		case "->", "::":
			return errf("err_oop", t.line, "%q hors subset v0.1", t.text)
		}
	}
	return nil
}

func (p *parser) parseFunc() (*Func, error) {
	line := p.next().line // function
	name := p.cur()
	if name.kind != tkIdent {
		return nil, errf("err_parse", name.line, "nom de fonction attendu")
	}
	p.pos++
	if err := p.expectOp("("); err != nil {
		return nil, err
	}
	fn := &Func{Name: name.text, Line: line}
	for !p.eatOp(")") {
		if len(fn.Params) > 0 {
			if err := p.expectOp(","); err != nil {
				return nil, err
			}
		}
		pt := p.cur()
		hint := "" // sans hint : int par défaut (types/)
		if pt.kind == tkIdent {
			switch pt.text {
			case "int", "string", "array":
				hint = pt.text
			case "float", "bool", "callable", "object", "mixed", "iterable":
				return nil, errf("err_parse", pt.line, "type hint %q hors subset (int, string, array seuls)", pt.text)
			default:
				return nil, errf("err_parse", pt.line, "paramètre $nom (ou hint) attendu, trouvé %q", pt.text)
			}
			p.pos++
			pt = p.cur()
		}
		if pt.kind != tkVar {
			return nil, errf("err_parse", pt.line, "paramètre $nom attendu, trouvé %q", pt.text)
		}
		fn.Params = append(fn.Params, pt.text)
		fn.ParamHints = append(fn.ParamHints, hint)
		p.pos++
	}
	// Type de retour optionnel « : int|string|array » — requis pour retourner
	// un kind non-int (F-p2go-scalar-signatures : contrat explicite, pas
	// d'inférence interprocédurale).
	if p.eatOp(":") {
		rt := p.cur()
		if rt.kind != tkIdent || (rt.text != "int" && rt.text != "string" && rt.text != "array") {
			return nil, errf("err_parse", rt.line, "type de retour %q hors subset (int, string, array)", rt.text)
		}
		fn.RetHint = rt.text
		p.pos++
	}
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	fn.Body = body
	return fn, nil
}

func (p *parser) parseBlock() ([]Stmt, error) {
	if err := p.expectOp("{"); err != nil {
		return nil, err
	}
	var out []Stmt
	for !p.eatOp("}") {
		if p.cur().kind == tkEOF {
			return nil, errf("err_parse", p.cur().line, "} attendu avant fin de fichier")
		}
		st, err := p.parseStmt()
		if err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	return out, nil
}

// parseBlockOrStmt : bloc {…} ou statement unique (corps de if/while/for sans accolades).
func (p *parser) parseBlockOrStmt() ([]Stmt, error) {
	if p.cur().kind == tkOp && p.cur().text == "{" {
		return p.parseBlock()
	}
	st, err := p.parseStmt()
	if err != nil {
		return nil, err
	}
	return []Stmt{st}, nil
}

func (p *parser) parseStmt() (Stmt, error) {
	t := p.cur()
	if err := p.checkForbidden(t); err != nil {
		return nil, err
	}
	switch {
	case t.kind == tkIdent && t.text == "function":
		return nil, errf("err_parse", t.line, "fonction imbriquée hors subset v0.1")
	case t.kind == tkIdent && t.text == "echo":
		return p.parseEcho()
	case t.kind == tkIdent && t.text == "if":
		return p.parseIf()
	case t.kind == tkIdent && t.text == "while":
		return p.parseWhile()
	case t.kind == tkIdent && t.text == "do":
		return p.parseDoWhile()
	case t.kind == tkIdent && t.text == "for":
		return p.parseFor()
	case t.kind == tkIdent && t.text == "foreach":
		return p.parseForeach()
	case t.kind == tkIdent && t.text == "switch":
		return p.parseSwitch()
	case t.kind == tkIdent && t.text == "break":
		p.pos++
		if p.cur().kind == tkInt {
			return nil, errf("err_parse", t.line, "break à niveaux (break n) hors subset")
		}
		if p.loopDepth == 0 && p.switchDepth == 0 {
			return nil, errf("err_parse", t.line, "break hors boucle et hors switch")
		}
		if err := p.expectOp(";"); err != nil {
			return nil, err
		}
		return &Break{Line: t.line}, nil
	case t.kind == tkIdent && t.text == "continue":
		p.pos++
		if p.cur().kind == tkInt {
			return nil, errf("err_parse", t.line, "continue à niveaux (continue n) hors subset")
		}
		if p.loopDepth == 0 {
			return nil, errf("err_parse", t.line, "continue hors boucle")
		}
		if err := p.expectOp(";"); err != nil {
			return nil, err
		}
		return &Continue{Line: t.line}, nil
	case t.kind == tkIdent && t.text == "return":
		p.pos++
		r := &Return{Line: t.line}
		if !(p.cur().kind == tkOp && p.cur().text == ";") {
			e, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			r.Expr = e
		}
		if err := p.expectOp(";"); err != nil {
			return nil, err
		}
		return r, nil
	case t.kind == tkVar || (t.kind == tkOp && (t.text == "++" || t.text == "--")):
		st, err := p.parseSimpleStmt(true)
		if err != nil {
			return nil, err
		}
		if err := p.expectOp(";"); err != nil {
			return nil, err
		}
		return st, nil
	case t.kind == tkIdent:
		// appel de fonction en statement
		e, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if _, ok := e.(*Call); !ok {
			return nil, errf("err_parse", t.line, "statement expression non-appel hors subset")
		}
		if err := p.expectOp(";"); err != nil {
			return nil, err
		}
		return &ExprStmt{Expr: e, Line: t.line}, nil
	}
	return nil, errf("err_parse", t.line, "statement inattendu %q", t.text)
}

// parseTernaryTail détecte « ? a : b » ou « ?: b » après une expression déjà
// parsée et construit le nœud Ternary (F-p2go-ternary-subexpr : désucré par
// hoisting de temporaire AVANT la passe types, voir desugarer).
func (p *parser) parseTernaryTail(cond Expr) (Expr, error) {
	t := p.cur()
	if t.kind != tkOp || t.text != "?" {
		return cond, nil
	}
	p.pos++
	if p.eatOp(":") { // forme courte c ?: b — c évalué une seule fois (temp)
		b, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		return &Ternary{Cond: cond, A: nil, B: b, Line: t.line}, nil
	}
	a, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if err := p.expectOp(":"); err != nil {
		return nil, err
	}
	b, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	return &Ternary{Cond: cond, A: a, B: b, Line: t.line}, nil
}

// parseSimpleStmt : affectation, composé ou incrément — sans le ; final.
// allowTernary n'est vrai qu'en position statement : les clauses init/post de
// for refusent le ternaire (l'emit d'un for Go n'y loge pas un if).
func (p *parser) parseSimpleStmt(allowTernary bool) (Stmt, error) {
	t := p.cur()
	if t.kind == tkOp && (t.text == "++" || t.text == "--") { // ++$i
		p.pos++
		v := p.cur()
		if v.kind != tkVar {
			return nil, errf("err_parse", v.line, "$variable attendue après %s", t.text)
		}
		p.pos++
		return &IncDec{Name: v.text, Op: t.text, Line: t.line}, nil
	}
	if t.kind != tkVar {
		return nil, errf("err_parse", t.line, "$variable attendue")
	}
	p.pos++
	if p.eatOp("[") { // $a[idx] = e; (écriture indexée, v0.2 partiel)
		idx, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if err := p.expectOp("]"); err != nil {
			return nil, err
		}
		if err := p.expectOp("="); err != nil {
			return nil, errf("err_parse", t.line, "= attendu après $%s[…] (composés indexés : v0.3)", t.text)
		}
		rhs, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		return &IndexAssign{Name: t.text, Idx: idx, Expr: rhs, Line: t.line}, nil
	}
	op := p.cur()
	if err := p.checkForbidden(op); err != nil {
		return nil, err
	}
	if op.kind != tkOp {
		return nil, errf("err_parse", op.line, "opérateur d'affectation attendu après $%s", t.text)
	}
	switch op.text {
	case "++", "--": // $i++
		p.pos++
		return &IncDec{Name: t.text, Op: op.text, Line: t.line}, nil
	case "=", "+=", "-=", "*=", "/=", "%=", ".=", "&=", "|=", "^=", "<<=", ">>=":
		p.pos++
		e, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		_ = allowTernary // le ternaire vit dans parseExpr ; l'interdiction en clause de for est portée par le désucreur
		return &Assign{Name: t.text, Op: op.text, Expr: e, Line: t.line}, nil
	}
	return nil, errf("err_parse", op.line, "opérateur %q inattendu après $%s", op.text, t.text)
}

func (p *parser) parseEcho() (Stmt, error) {
	line := p.next().line
	e := &Echo{Line: line}
	for {
		a, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		e.Args = append(e.Args, a)
		if !p.eatOp(",") {
			break
		}
	}
	if err := p.expectOp(";"); err != nil {
		return nil, err
	}
	return e, nil
}

func (p *parser) parseIf() (Stmt, error) {
	line := p.next().line
	if err := p.expectOp("("); err != nil {
		return nil, err
	}
	cond, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if err := p.expectOp(")"); err != nil {
		return nil, err
	}
	then, err := p.parseBlockOrStmt()
	if err != nil {
		return nil, err
	}
	st := &If{Cond: cond, Then: then, Line: line}
	if p.cur().kind == tkIdent && p.cur().text == "elseif" {
		p.toks[p.pos].text = "if" // désucre elseif → else { if … }
		sub, err := p.parseIf()
		if err != nil {
			return nil, err
		}
		st.Else = []Stmt{sub}
		return st, nil
	}
	if p.cur().kind == tkIdent && p.cur().text == "else" {
		p.pos++
		if p.cur().kind == tkIdent && p.cur().text == "if" { // else if
			sub, err := p.parseIf()
			if err != nil {
				return nil, err
			}
			st.Else = []Stmt{sub}
			return st, nil
		}
		els, err := p.parseBlockOrStmt()
		if err != nil {
			return nil, err
		}
		st.Else = els
	}
	return st, nil
}

func (p *parser) parseWhile() (Stmt, error) {
	line := p.next().line
	if err := p.expectOp("("); err != nil {
		return nil, err
	}
	cond, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if err := p.expectOp(")"); err != nil {
		return nil, err
	}
	body, err := p.parseLoopBody()
	if err != nil {
		return nil, err
	}
	return &While{Cond: cond, Body: body, Line: line}, nil
}

// parseForeach — F-p2go-foreach : foreach ($a as [$i =>] $v) B est désucré en
// for indexé — for ($i = 0; $i < count($a); $i++) { $v = $a[$i]; B } — avec un
// compteur gensym p2go_feN quand la clé n'est pas nommée. La forme by-ref
// (&$v) est refusée : muter $v n'écrit pas dans le tableau (sémantique valeur).
func (p *parser) parseForeach() (Stmt, error) {
	line := p.next().line // foreach
	if err := p.expectOp("("); err != nil {
		return nil, err
	}
	// Sujet : $variable tableau, ou toute expression tableau (v0.5) —
	// matérialisée dans un temporaire p2go_faN avant la boucle.
	var arrName string
	var subjExpr Expr
	if t := p.cur(); t.kind == tkVar && p.toks[p.pos+1].kind == tkIdent && p.toks[p.pos+1].text == "as" {
		arrName = t.text
		p.pos++
	} else {
		e, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		subjExpr = e
		arrName = fmt.Sprintf("p2go_fa%d", p.feN)
		p.feN++
	}
	as := p.cur()
	if as.kind != tkIdent || as.text != "as" {
		return nil, errf("err_parse", as.line, "as attendu dans foreach")
	}
	p.pos++
	if p.cur().kind == tkOp && p.cur().text == "&" {
		return nil, errf("err_parse", p.cur().line, "foreach by-ref (&$v) hors subset — sémantique valeur seule")
	}
	first := p.cur()
	if first.kind != tkVar {
		return nil, errf("err_parse", first.line, "$variable attendue après as")
	}
	p.pos++
	keyName := ""
	valName := first.text
	if p.eatOp("=>") {
		if p.cur().kind == tkOp && p.cur().text == "&" {
			return nil, errf("err_parse", p.cur().line, "foreach by-ref (&$v) hors subset — sémantique valeur seule")
		}
		v := p.cur()
		if v.kind != tkVar {
			return nil, errf("err_parse", v.line, "$valeur attendue après =>")
		}
		p.pos++
		keyName = first.text
		valName = v.text
	}
	if err := p.expectOp(")"); err != nil {
		return nil, err
	}
	body, err := p.parseLoopBody()
	if err != nil {
		return nil, err
	}
	counter := keyName
	if counter == "" {
		counter = fmt.Sprintf("p2go_fe%d", p.feN)
		p.feN++
	}
	inner := append([]Stmt{
		&Assign{Name: valName, Op: "=",
			Expr: &Index{Name: arrName, Idx: &Var{Name: counter, Line: line}, Line: line},
			Line: line},
	}, body...)
	loop := &For{
		Init: &Assign{Name: counter, Op: "=", Expr: &IntLit{Value: 0, Line: line}, Line: line},
		Cond: &Binary{Op: "<",
			L:    &Var{Name: counter, Line: line},
			R:    &Call{Name: "count", Args: []Expr{&Var{Name: arrName, Line: line}}, Line: line},
			Line: line},
		Post: &IncDec{Name: counter, Op: "++", Line: line},
		Body: inner,
		Line: line,
	}
	if subjExpr != nil { // matérialisation du sujet-expression avant la boucle
		return &Block{Stmts: []Stmt{
			&Assign{Name: arrName, Op: "=", Expr: subjExpr, Line: line},
			loop,
		}}, nil
	}
	return loop, nil
}

// parseSwitch — F-p2go-switch : switch strict sans fallthrough implicite.
// Chaque case non vide se termine par break; (consommé) ou return ; un case
// vide s'empile sur le suivant (case A: case B: corps → case A, B en Go).
func (p *parser) parseSwitch() (Stmt, error) {
	line := p.next().line // switch
	if err := p.expectOp("("); err != nil {
		return nil, err
	}
	subj, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if err := p.expectOp(")"); err != nil {
		return nil, err
	}
	if err := p.expectOp("{"); err != nil {
		return nil, err
	}
	st := &Switch{Subject: subj, Line: line}
	var pending []Expr
	for !p.eatOp("}") {
		t := p.cur()
		switch {
		case t.kind == tkIdent && t.text == "case":
			p.pos++
			v, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			if err := p.expectOp(":"); err != nil {
				return nil, err
			}
			pending = append(pending, v)
			body, terminated, err := p.parseCaseBody()
			if err != nil {
				return nil, err
			}
			if len(body) == 0 && !terminated {
				continue // case vide empilé sur le suivant
			}
			st.Cases = append(st.Cases, SwitchCase{Vals: pending, Body: body})
			pending = nil
		case t.kind == tkIdent && t.text == "default":
			p.pos++
			if len(pending) > 0 {
				return nil, errf("err_parse", t.line, "case empilé sur default hors subset")
			}
			if st.Default != nil {
				return nil, errf("err_parse", t.line, "default dupliqué")
			}
			if err := p.expectOp(":"); err != nil {
				return nil, err
			}
			body, _, err := p.parseCaseBody()
			if err != nil {
				return nil, err
			}
			if body == nil {
				body = []Stmt{}
			}
			st.Default = body
		default:
			return nil, errf("err_parse", t.line, "case ou default attendu dans switch, trouvé %q", t.text)
		}
	}
	if len(pending) > 0 {
		return nil, errf("err_parse", line, "case sans corps ni break en fin de switch")
	}
	return st, nil
}

// parseCaseBody lit les statements d'un case jusqu'à break; (consommé),
// case/default/} (borne). Fail-loud : un corps non vide sans break doit se
// terminer par return — le fallthrough implicite PHP n'est pas imité.
func (p *parser) parseCaseBody() ([]Stmt, bool, error) {
	p.switchDepth++
	defer func() { p.switchDepth-- }()
	var body []Stmt
	for {
		t := p.cur()
		if t.kind == tkOp && t.text == "}" {
			break
		}
		if t.kind == tkIdent && (t.text == "case" || t.text == "default") {
			break
		}
		if t.kind == tkIdent && t.text == "break" {
			p.pos++
			if err := p.expectOp(";"); err != nil {
				return nil, false, err
			}
			return body, true, nil
		}
		if t.kind == tkEOF {
			return nil, false, errf("err_parse", t.line, "} attendu avant fin de fichier (switch)")
		}
		st, err := p.parseStmt()
		if err != nil {
			return nil, false, err
		}
		body = append(body, st)
	}
	if len(body) == 0 {
		return nil, false, nil
	}
	if _, isRet := body[len(body)-1].(*Return); !isRet {
		return nil, false, errf("err_parse", p.cur().line, "fallthrough implicite hors subset — terminer le case par break; ou return")
	}
	return body, true, nil
}

// parseMatch — F-p2go-match : match (subj) { v1, v2 => e, default => e }
// (expression PHP 8, sans fallthrough) ; désucré en Switch par le désucreur.
// default obligatoire : UnhandledMatchError n'est pas émulé.
func (p *parser) parseMatch(line int) (Expr, error) {
	if err := p.expectOp("("); err != nil {
		return nil, err
	}
	subj, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if err := p.expectOp(")"); err != nil {
		return nil, err
	}
	if err := p.expectOp("{"); err != nil {
		return nil, err
	}
	m := &Match{Subject: subj, Line: line}
	for !p.eatOp("}") {
		if t := p.cur(); t.kind == tkIdent && t.text == "default" {
			p.pos++
			if err := p.expectOp("=>"); err != nil {
				return nil, err
			}
			res, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			if m.Default != nil {
				return nil, errf("err_parse", t.line, "default dupliqué dans match")
			}
			m.Default = res
			p.eatOp(",")
			continue
		}
		var vals []Expr
		for {
			v, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			vals = append(vals, v)
			if p.eatOp("=>") {
				break
			}
			if err := p.expectOp(","); err != nil {
				return nil, err
			}
		}
		res, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		m.Arms = append(m.Arms, MatchArm{Vals: vals, Result: res})
		p.eatOp(",")
	}
	if m.Default == nil {
		return nil, errf("err_parse", line, "match sans default hors subset (UnhandledMatchError non émulé)")
	}
	return m, nil
}

// parseDoWhile — F-p2go-do-while : do { … } while (cond); (capture dogfood 2026-08-19).
func (p *parser) parseDoWhile() (Stmt, error) {
	line := p.next().line // do
	body, err := p.parseLoopBody()
	if err != nil {
		return nil, err
	}
	kw := p.cur()
	if kw.kind != tkIdent || kw.text != "while" {
		return nil, errf("err_parse", kw.line, "while attendu après le corps du do")
	}
	p.pos++
	if err := p.expectOp("("); err != nil {
		return nil, err
	}
	cond, perr := p.parseExpr()
	if perr != nil {
		return nil, perr
	}
	if err := p.expectOp(")"); err != nil {
		return nil, err
	}
	if err := p.expectOp(";"); err != nil {
		return nil, err
	}
	// Le désucrage do…while DUPLIQUE le corps hors boucle : un break/continue
	// qui s'échapperait du corps (non capturé par une boucle interne) serait
	// orphelin dans la copie déroulée — refus explicite.
	if jl, esc := jumpEscapes(body, false); esc {
		return nil, errf("err_parse", jl, "break/continue s'échappant d'un do…while hors subset (désucrage par duplication du corps)")
	}
	return &DoWhile{Cond: cond, Body: body, Line: line}, nil
}

// jumpEscapes détecte un Break/Continue qui s'échapperait du corps : un break
// est capturé par un switch interne (sémantique Go et PHP alignées), un
// continue ne l'est jamais ; les boucles internes capturent les deux.
func jumpEscapes(body []Stmt, inSwitch bool) (int, bool) {
	for _, st := range body {
		switch s := st.(type) {
		case *Break:
			if !inSwitch {
				return s.Line, true
			}
		case *Continue:
			return s.Line, true
		case *If:
			if l, e := jumpEscapes(s.Then, inSwitch); e {
				return l, true
			}
			if l, e := jumpEscapes(s.Else, inSwitch); e {
				return l, true
			}
		case *Switch:
			for _, c := range s.Cases {
				if l, e := jumpEscapes(c.Body, true); e {
					return l, true
				}
			}
			if l, e := jumpEscapes(s.Default, true); e {
				return l, true
			}
		case *Block:
			if l, e := jumpEscapes(s.Stmts, inSwitch); e {
				return l, true
			}
		}
	}
	return 0, false
}

func (p *parser) parseFor() (Stmt, error) {
	line := p.next().line
	if err := p.expectOp("("); err != nil {
		return nil, err
	}
	st := &For{Line: line}
	if !(p.cur().kind == tkOp && p.cur().text == ";") {
		init, err := p.parseSimpleStmt(false)
		if err != nil {
			return nil, err
		}
		st.Init = init
	}
	if err := p.expectOp(";"); err != nil {
		return nil, err
	}
	if !(p.cur().kind == tkOp && p.cur().text == ";") {
		cond, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		st.Cond = cond
	}
	if err := p.expectOp(";"); err != nil {
		return nil, err
	}
	if !(p.cur().kind == tkOp && p.cur().text == ")") {
		post, err := p.parseSimpleStmt(false)
		if err != nil {
			return nil, err
		}
		st.Post = post
	}
	if err := p.expectOp(")"); err != nil {
		return nil, err
	}
	body, err := p.parseLoopBody()
	if err != nil {
		return nil, err
	}
	st.Body = body
	return st, nil
}

// --- Expressions — précédence PHP 8 croissante :
// ternaire < || < && < | < ^ < & < ==/!= < rel < . < <</>> < +/- < */ /% < unaire ---

func (p *parser) parseExpr() (Expr, error) {
	e, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	return p.parseTernaryTail(e)
}

func (p *parser) parseBinaryLevel(ops []string, sub func() (Expr, error)) (Expr, error) {
	l, err := sub()
	if err != nil {
		return nil, err
	}
	for {
		t := p.cur()
		matched := false
		if t.kind == tkOp {
			for _, op := range ops {
				if t.text == op {
					p.pos++
					r, err := sub()
					if err != nil {
						return nil, err
					}
					l = &Binary{Op: op, L: l, R: r, Line: t.line}
					matched = true
					break
				}
			}
		}
		if !matched {
			return l, nil
		}
	}
}

func (p *parser) parseOr() (Expr, error) {
	return p.parseBinaryLevel([]string{"||"}, p.parseAnd)
}
func (p *parser) parseAnd() (Expr, error) {
	return p.parseBinaryLevel([]string{"&&"}, p.parseBitOr)
}
func (p *parser) parseBitOr() (Expr, error) {
	return p.parseBinaryLevel([]string{"|"}, p.parseBitXor)
}
func (p *parser) parseBitXor() (Expr, error) {
	return p.parseBinaryLevel([]string{"^"}, p.parseBitAnd)
}
func (p *parser) parseBitAnd() (Expr, error) {
	return p.parseBinaryLevel([]string{"&"}, p.parseEq)
}
func (p *parser) parseEq() (Expr, error) {
	// === / !== sur ints : mêmes sémantiques que == / != une fois typé int strict
	l, err := p.parseBinaryLevel([]string{"==", "!=", "===", "!=="}, p.parseRel)
	if err != nil {
		return nil, err
	}
	normalizeStrictEq(l)
	return l, nil
}
func (p *parser) parseRel() (Expr, error) {
	return p.parseBinaryLevel([]string{"<", "<=", ">", ">="}, p.parseConcat)
}

// parseConcat — précédence PHP 8 : « . » lie moins fort que les shifts.
func (p *parser) parseConcat() (Expr, error) {
	return p.parseBinaryLevel([]string{"."}, p.parseShift)
}
func (p *parser) parseShift() (Expr, error) {
	return p.parseBinaryLevel([]string{"<<", ">>"}, p.parseAdd)
}
func (p *parser) parseAdd() (Expr, error) {
	return p.parseBinaryLevel([]string{"+", "-"}, p.parseMul)
}
func (p *parser) parseMul() (Expr, error) {
	return p.parseBinaryLevel([]string{"*", "/", "%"}, p.parseUnary)
}

func normalizeStrictEq(e Expr) {
	if b, ok := e.(*Binary); ok {
		switch b.Op {
		case "===":
			b.Op = "=="
		case "!==":
			b.Op = "!="
		}
		normalizeStrictEq(b.L)
		normalizeStrictEq(b.R)
	}
}

func (p *parser) parseUnary() (Expr, error) {
	t := p.cur()
	if t.kind == tkOp && (t.text == "-" || t.text == "!" || t.text == "~") {
		p.pos++
		x, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &Unary{Op: t.text, X: x, Line: t.line}, nil
	}
	return p.parsePow()
}

// parsePow — ** PHP : lie plus fort que l'unaire (-2**2 = -(2**2)),
// associatif à DROITE, exposant unaire autorisé (2**-1 refusé plus loin par
// p2goPow, domaine int). Plié en appel pow(a, b).
func (p *parser) parsePow() (Expr, error) {
	l, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	t := p.cur()
	if t.kind == tkOp && t.text == "**" {
		p.pos++
		r, err := p.parseUnary() // droite-assoc, unaire admis dans l'exposant
		if err != nil {
			return nil, err
		}
		return &Call{Name: "pow", Args: []Expr{l, r}, Line: t.line}, nil
	}
	return l, nil
}

func (p *parser) parsePrimary() (Expr, error) {
	t := p.cur()
	if err := p.checkForbidden(t); err != nil {
		return nil, err
	}
	switch t.kind {
	case tkInt:
		p.pos++
		return &IntLit{Value: t.ival, Line: t.line}, nil
	case tkStr:
		p.pos++
		return &StrLit{Value: t.text, Line: t.line}, nil
	case tkVar:
		p.pos++
		if p.eatOp("[") { // lecture indexée $a[idx]
			idx, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			if err := p.expectOp("]"); err != nil {
				return nil, err
			}
			return &Index{Name: t.text, Idx: idx, Line: t.line}, nil
		}
		return &Var{Name: t.text, Line: t.line}, nil
	case tkIdent:
		name := t.text
		if strings.ToLower(name) == "true" {
			p.pos++
			return &IntLit{Value: 1, Line: t.line}, nil
		}
		if strings.ToLower(name) == "false" {
			p.pos++
			return &IntLit{Value: 0, Line: t.line}, nil
		}
		if strings.ToLower(name) == "match" {
			p.pos++
			return p.parseMatch(t.line)
		}
		p.pos++
		if err := p.expectOp("("); err != nil {
			return nil, errf("err_parse", t.line, "identifiant %q hors contexte d'appel", name)
		}
		call := &Call{Name: name, Line: t.line}
		for !p.eatOp(")") {
			if len(call.Args) > 0 {
				if err := p.expectOp(","); err != nil {
					return nil, err
				}
			}
			a, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			call.Args = append(call.Args, a)
		}
		return call, nil
	case tkOp:
		if t.text == "[" { // littéral de tableau indexé homogène
			p.pos++
			lit := &ArrLit{Line: t.line}
			for !p.eatOp("]") {
				if len(lit.Elems) > 0 {
					if err := p.expectOp(","); err != nil {
						return nil, err
					}
				}
				el, err := p.parseExpr()
				if err != nil {
					return nil, err
				}
				lit.Elems = append(lit.Elems, el)
			}
			return lit, nil
		}
		if t.text == "(" {
			p.pos++
			e, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			if err := p.expectOp(")"); err != nil {
				return nil, err
			}
			return e, nil
		}
	}
	return nil, errf("err_parse", t.line, "expression attendue, trouvé %q", t.text)
}
