package front

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// foldDefines expands simple object-like #define and a few function-like shims.
// Order: collect definitions (last wins), evaluate integer exprs with known names,
// then textual replace longest-first. Function-like FASTLZ_LIKELY(x) → (x).
// stripIfDefs keeps #else branch when present, else the #if body; drops #elif chains simply.
func stripIfDefs(src string) string {
	var out []string
	lines := strings.Split(src, "\n")
	// depth stack: mode keep/skip
	type frame struct{ keep, seenElse bool }
	var stack []frame
	keeping := func() bool {
		for _, f := range stack {
			if !f.keep {
				return false
			}
		}
		return true
	}
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trim, "#if") || strings.HasPrefix(trim, "#ifdef") || strings.HasPrefix(trim, "#ifndef"):
			// prefer skipping arch-specific first branch if #else exists later — keep true branch by default
			stack = append(stack, frame{keep: true, seenElse: false})
			continue
		case strings.HasPrefix(trim, "#else"):
			if len(stack) == 0 {
				continue
			}
			// flip: drop if-body, keep else
			top := &stack[len(stack)-1]
			if !top.seenElse {
				top.keep = !top.keep
				top.seenElse = true
			}
			continue
		case strings.HasPrefix(trim, "#elif"):
			if len(stack) == 0 {
				continue
			}
			top := &stack[len(stack)-1]
			// skip elif branches after first
			top.keep = false
			continue
		case strings.HasPrefix(trim, "#endif"):
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			continue
		}
		if keeping() {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

// stripAsmBarriers removes inline-asm memory barriers — asm with an EMPTY
// instruction string (__asm__ volatile(""::"m"(x)) or asm("")) — as no-ops.
// Any asm with real instructions is left untouched (rejected later as err_asm).
func stripAsmBarriers(src string) string {
	var out strings.Builder
	i := 0
	for i < len(src) {
		rem := src[i:]
		m := asmBarrierRe.FindStringIndex(rem)
		if m == nil {
			out.WriteString(rem)
			break
		}
		out.WriteString(rem[:m[0]])
		start := i + m[0]
		// find balanced '(' after the asm/volatile prefix
		open := strings.IndexByte(src[start:], '(')
		if open < 0 {
			out.WriteString(src[i:])
			break
		}
		open += start
		depth := 0
		end := -1
		for j := open; j < len(src); j++ {
			switch src[j] {
			case '(':
				depth++
			case ')':
				depth--
				if depth == 0 {
					end = j
				}
			case '"':
				// skip string literal (may contain parens/escapes)
				j++
				for j < len(src) {
					if src[j] == '\\' && j+1 < len(src) {
						j += 2
						continue
					}
					if src[j] == '"' {
						break
					}
					j++
				}
			}
			if end >= 0 {
				break
			}
		}
		if end < 0 {
			out.WriteString(src[i:])
			break
		}
		inner := strings.TrimSpace(src[open+1 : end])
		if strings.HasPrefix(inner, `""`) {
			// empty instruction string → no-op barrier, drop entirely
			i = end + 1
			continue
		}
		out.WriteString(src[start : end+1])
		i = end + 1
	}
	return out.String()
}

// foldLocalIncludes inlines the CONTENT of local `#include "x.h"` headers into
// the source BEFORE analysis (S3). Resolution: relative to the .c file dir,
// then a spec/c_sources root found by walking up from CWD and the .c dir.
// Only *.h targets are folded; system <...> includes are left (dropped later).
// Missing local header -> ErrInclude.
func foldLocalIncludes(src, cDir string) (string, error) {
	re := regexp.MustCompile(`(?m)^[ \t]*#[ \t]*include[ \t]+"([^"]+)"[ \t]*$`)
	if re.MatchString(src) == false {
		return src, nil
	}
	var roots []string
	if cDir != "" {
		roots = append(roots, cDir)
	}
	// spec/c_sources roots from cDir up and from CWD up
	for _, base := range []string{cDir, "."} {
		if base == "" {
			continue
		}
		abs, err := filepath.Abs(base)
		if err != nil {
			continue
		}
		for d := abs; ; d = filepath.Dir(d) {
			cand := filepath.Join(d, "spec", "c_sources")
			if st, err := os.Stat(cand); err == nil && st.IsDir() {
				roots = append(roots, cand)
			}
			if filepath.Dir(d) == d {
				break
			}
		}
	}

	visited := map[string]bool{}
	var missing []string
	var errBad error
	var fold func(s string) string
	fold = func(s string) string {
		return re.ReplaceAllStringFunc(s, func(m string) string {
			name := re.FindStringSubmatch(m)[1]
			// only fold headers; .c data includes (utf8proc_data.c) stay for later drop
			if !strings.HasSuffix(name, ".h") {
				return m
			}
			for _, root := range roots {
				cand := filepath.Join(root, name)
				if visited[cand] {
					return "/* include " + name + " */"
				}
				if st, err := os.Stat(cand); err == nil && !st.IsDir() {
					// skip huge headers (yyjson.h ~330KiB) — avoid harvest hang
					if st.Size() > 64*1024 {
						visited[cand] = true
						return "/* include " + name + " omitted (large) */"
					}
					visited[cand] = true
					b, err := os.ReadFile(cand)
					if err != nil {
						missing = append(missing, name)
						return m
					}
					return "\n" + fold(string(b)) + "\n"
				}
			}
			missing = append(missing, name)
			return m
		})
	}
	out := fold(src)
	_ = errBad
	if len(missing) > 0 {
		return "", &Error{Code: ErrInclude, Message: "include not found: " + missing[0]}
	}
	return out, nil
}

var asmBarrierRe = regexp.MustCompile(`(?i)(^|[^A-Za-z0-9_])(?:__asm__|asm)[ \t]+volatile[ \t]*\(`)

func foldDefines(src string) string {
	src = stripIfDefs(src)
	type objDef struct {
		name string
		body string
	}
	var objs []objDef
	funcRepl := map[string]string{} // name -> body with $1 for first arg simple

	// join backslash-continued lines before define parse
	rawLines := strings.Split(src, "\n")
	var lines []string
	for i := 0; i < len(rawLines); i++ {
		ln := rawLines[i]
		for strings.HasSuffix(strings.TrimRight(ln, " \t"), "\\") && i+1 < len(rawLines) {
			ln = strings.TrimRight(ln, " \t")
			ln = strings.TrimSuffix(ln, "\\")
			i++
			ln = ln + " " + strings.TrimSpace(rawLines[i])
		}
		lines = append(lines, ln)
	}
	var out []string
	defRe := regexp.MustCompile(`^\s*#\s*define\s+([A-Za-z_][A-Za-z0-9_]*)(\(([^)]*)\))?\s*(.*)$`)
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		m := defRe.FindStringSubmatch(trim)
		if m == nil {
			out = append(out, line)
			continue
		}
		name, args, body := m[1], strings.TrimSpace(m[3]), strings.TrimSpace(m[4])
		body = strings.TrimSpace(body)
		if args != "" {
			params := splitDefineArgs(args)
			if len(params) == 1 {
				p := params[0]
				b := body
				if b == p || b == "("+p+")" {
					funcRepl[name] = "($ARG)"
				} else if strings.Contains(b, "__builtin_expect") {
					funcRepl[name] = "($ARG)"
				} else if strings.Contains(b, p) {
					funcRepl[name] = strings.ReplaceAll(b, p, "$ARG")
				}
			} else if len(params) >= 2 && body != "" {
				// multi-arg: #define FOR(i,n) for (i = 0; i < n; ++i)
				// NOTE: placeholders must NOT use $ — regexp.ReplaceAllString treats $ as group refs.
				b := body
				for i, p := range params {
					re := regexp.MustCompile(`\b` + regexp.QuoteMeta(p) + `\b`)
					b = re.ReplaceAllString(b, fmt.Sprintf("@ARG%d@", i))
				}
				funcRepl[name] = b
			}
			// drop #define line
			continue
		}
		if body == "" {
			// flag define like FLZ_ARCH64 — expand to 1
			objs = append(objs, objDef{name, "1"})
			continue
		}
		objs = append(objs, objDef{name, body})
	}

	// evaluate object macros to integers when possible (multi-pass)
	vals := map[string]int64{}
	for pass := 0; pass < 16; pass++ {
		progress := false
		for _, d := range objs {
			if _, ok := vals[d.name]; ok {
				continue
			}
			if n, ok := evalDefineExpr(d.body, vals); ok {
				vals[d.name] = n
				progress = true
			}
		}
		if !progress {
			break
		}
	}

	text := strings.Join(out, "\n")

	// object-like repl map
	objRepl := map[string]string{}
	for _, d := range objs {
		if n, ok := vals[d.name]; ok {
			objRepl[d.name] = strconv.FormatInt(n, 10)
			continue
		}
		// multi-stmt / do-while bodies: no paren wrap
		if strings.Contains(d.body, ";") || strings.Contains(d.body, "{") {
			objRepl[d.name] = d.body
		} else {
			objRepl[d.name] = "(" + d.body + ")"
		}
	}
	names := make([]string, 0, len(objRepl))
	for n := range objRepl {
		names = append(names, n)
	}
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			if len(names[j]) > len(names[i]) {
				names[i], names[j] = names[j], names[i]
			}
		}
	}

	expandFuncs := func(text string) string {
		// nested-paren aware call match: NAME( ... )
		for name, body := range funcRepl {
			for {
				re := regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\s*\(`)
				loc := re.FindStringIndex(text)
				if loc == nil {
					break
				}
				open := loc[1] - 1 // '('
				close, err := matchParenStr(text, open)
				if err != nil {
					break
				}
				argStr := text[open+1 : close]
				var repl string
				if strings.Contains(body, "@ARG0@") {
					args := splitDefineArgs(argStr)
					repl = body
					for i, a := range args {
						repl = strings.ReplaceAll(repl, fmt.Sprintf("@ARG%d@", i), a)
					}
				} else {
					repl = strings.ReplaceAll(body, "$ARG", strings.TrimSpace(argStr))
				}
				text = text[:loc[0]] + repl + text[close+1:]
			}
		}
		return text
	}
	expandObjs := func(text string) string {
		for _, name := range names {
			re := regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\b`)
			text = re.ReplaceAllString(text, objRepl[name])
		}
		return text
	}
	// multi-pass: funcs ↔ objects (nested macros in bodies)
	for pass := 0; pass < 8; pass++ {
		next := expandObjs(expandFuncs(text))
		if next == text {
			break
		}
		text = next
	}

	// drop remaining # lines (include, if, pragma, undef, …)
	var final []string
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		final = append(final, line)
	}
	return strings.Join(final, "\n")
}

func matchParenStr(s string, open int) (int, error) {
	depth := 0
	for i := open; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i, nil
			}
		}
	}
	return 0, fmt.Errorf("unclosed (")
}

func splitDefineArgs(s string) []string {
	var out []string
	var cur strings.Builder
	depth := 0
	for _, r := range s {
		switch r {
		case '(', '[':
			depth++
			cur.WriteRune(r)
		case ')', ']':
			depth--
			cur.WriteRune(r)
		case ',':
			if depth == 0 {
				p := strings.TrimSpace(cur.String())
				if p != "" {
					out = append(out, p)
				}
				cur.Reset()
				continue
			}
			cur.WriteRune(r)
		default:
			cur.WriteRune(r)
		}
	}
	if p := strings.TrimSpace(cur.String()); p != "" {
		out = append(out, p)
	}
	return out
}

// evalDefineExpr evaluates a tiny integer expression with known names.
func evalDefineExpr(expr string, vals map[string]int64) (int64, bool) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return 0, false
	}
	// replace known ids
	reID := regexp.MustCompile(`\b[A-Za-z_][A-Za-z0-9_]*\b`)
	replaced := reID.ReplaceAllStringFunc(expr, func(id string) string {
		if n, ok := vals[id]; ok {
			return strconv.FormatInt(n, 10)
		}
		// hex/decimal leave; unknown id fail later
		if _, err := strconv.ParseInt(strings.TrimRight(id, "uUlL"), 0, 64); err == nil {
			return id
		}
		return id
	})
	// if any letter remains (unknown id), fail
	if regexp.MustCompile(`[A-Za-z_]`).MatchString(replaced) {
		return 0, false
	}
	return evalIntExpr(replaced)
}

// evalIntExpr: + - * / << >> & | ^ and parens, integers.
func evalIntExpr(s string) (int64, bool) {
	s = strings.TrimSpace(s)
	var pos int
	var parse func() (int64, bool)
	peek := func() byte {
		for pos < len(s) && (s[pos] == ' ' || s[pos] == '\t') {
			pos++
		}
		if pos >= len(s) {
			return 0
		}
		return s[pos]
	}
	parsePrimary := func() (int64, bool) {
		if peek() == '(' {
			pos++
			v, ok := parse()
			if !ok || peek() != ')' {
				return 0, false
			}
			pos++
			return v, true
		}
		start := pos
		if peek() == '-' || peek() == '+' {
			pos++
		}
		for pos < len(s) && (s[pos] >= '0' && s[pos] <= '9' || s[pos] == 'x' || s[pos] == 'X' ||
			(s[pos] >= 'a' && s[pos] <= 'f') || (s[pos] >= 'A' && s[pos] <= 'F') ||
			s[pos] == 'u' || s[pos] == 'U' || s[pos] == 'l' || s[pos] == 'L') {
			pos++
		}
		tok := strings.TrimRight(s[start:pos], "uUlL")
		if tok == "" || tok == "-" || tok == "+" {
			return 0, false
		}
		n, err := strconv.ParseInt(tok, 0, 64)
		return n, err == nil
	}
	var parseOr func() (int64, bool)
	var parseAnd func() (int64, bool)
	var parseShift func() (int64, bool)
	var parseAdd func() (int64, bool)
	var parseMul func() (int64, bool)

	parseMul = func() (int64, bool) {
		v, ok := parsePrimary()
		if !ok {
			return 0, false
		}
		for {
			c := peek()
			if c != '*' && c != '/' && c != '%' {
				return v, true
			}
			op := c
			pos++
			r, ok := parsePrimary()
			if !ok {
				return 0, false
			}
			switch op {
			case '*':
				v *= r
			case '/':
				if r == 0 {
					return 0, false
				}
				v /= r
			case '%':
				if r == 0 {
					return 0, false
				}
				v %= r
			}
		}
	}
	parseAdd = func() (int64, bool) {
		v, ok := parseMul()
		if !ok {
			return 0, false
		}
		for {
			c := peek()
			if c != '+' && c != '-' {
				return v, true
			}
			op := c
			pos++
			r, ok := parseMul()
			if !ok {
				return 0, false
			}
			if op == '+' {
				v += r
			} else {
				v -= r
			}
		}
	}
	parseShift = func() (int64, bool) {
		v, ok := parseAdd()
		if !ok {
			return 0, false
		}
		for {
			if strings.HasPrefix(s[pos:], "<<") {
				pos += 2
				r, ok := parseAdd()
				if !ok {
					return 0, false
				}
				v <<= uint(r)
				continue
			}
			if strings.HasPrefix(s[pos:], ">>") {
				pos += 2
				r, ok := parseAdd()
				if !ok {
					return 0, false
				}
				v >>= uint(r)
				continue
			}
			return v, true
		}
	}
	parseAnd = func() (int64, bool) {
		v, ok := parseShift()
		if !ok {
			return 0, false
		}
		for peek() == '&' && !strings.HasPrefix(s[pos:], "&&") {
			pos++
			r, ok := parseShift()
			if !ok {
				return 0, false
			}
			v &= r
		}
		return v, true
	}
	parseOr = func() (int64, bool) {
		v, ok := parseAnd()
		if !ok {
			return 0, false
		}
		for {
			c := peek()
			if c == '|' && !strings.HasPrefix(s[pos:], "||") {
				pos++
				r, ok := parseAnd()
				if !ok {
					return 0, false
				}
				v |= r
				continue
			}
			if c == '^' {
				pos++
				r, ok := parseAnd()
				if !ok {
					return 0, false
				}
				v ^= r
				continue
			}
			return v, true
		}
	}
	parse = parseOr
	v, ok := parse()
	if !ok {
		return 0, false
	}
	// trailing junk?
	for pos < len(s) && (s[pos] == ' ' || s[pos] == '\t') {
		pos++
	}
	if pos != len(s) {
		return 0, false
	}
	return v, true
}
