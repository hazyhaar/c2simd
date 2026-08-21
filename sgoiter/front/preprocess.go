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
			keep := true
			if strings.HasPrefix(trim, "#ifdef SWIG") || strings.HasPrefix(trim, "#ifdef __cplusplus") || trim == "#if 0" || strings.HasPrefix(trim, "#if 0 ") || strings.HasPrefix(trim, "#ifdef ENABLE_LOCALES") || strings.HasPrefix(trim, "#ifdef CJSON_GLOBAL") {
				keep = false
			} else if strings.HasPrefix(trim, "#ifndef SWIG") || strings.HasPrefix(trim, "#ifndef __cplusplus") || trim == "#if 1" || strings.HasPrefix(trim, "#if 1 ") {
				keep = true
			}
			stack = append(stack, frame{keep: keep, seenElse: false})
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
			if strings.HasPrefix(trim, "%") {
				continue
			}
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
			for _, sub := range []string{"spec/c_sources", "include", "."} {
				cand := filepath.Join(d, sub)
				if st, err := os.Stat(cand); err == nil && st.IsDir() {
					roots = append(roots, cand)
				}
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
			// only fold headers and local .c data includes (e.g. utf8proc_data.c)
			if !strings.HasSuffix(name, ".h") && !strings.HasSuffix(name, ".c") {
				return m
			}
			for _, root := range roots {
				cand := filepath.Join(root, name)
				if _, err := os.Stat(cand); err != nil {
					candGen := cand + ".generic"
					if st, err := os.Stat(candGen); err == nil && !st.IsDir() {
						cand = candGen
					}
				}
				if visited[cand] {
					return "/* include " + name + " */"
				}
				if st, err := os.Stat(cand); err == nil && !st.IsDir() {
					// skip excessively huge files (>4MiB)
					if st.Size() > 4*1024*1024 {
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
			if strings.HasSuffix(name, "_export.h") || strings.HasSuffix(name, "_api.h") || strings.HasSuffix(name, "_config.h") || name == "config.h" || name == "endian.h" || name == "sys/endian.h" || name == "byteswap.h" || name == "windows.h" {
				return "/* optional include " + name + " omitted */"
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

var asmBarrierRe = regexp.MustCompile(`(?i)(^|[^A-Za-z0-9_])(?:__asm__|asm)[ \t]*(?:volatile[ \t]*)?\(`)

func foldDefines(src string) string {
	src = stripIfDefs(src)
	type objDef struct {
		name string
		body string
	}
	var objs []objDef
	funcRepl := map[string]string{
		"likely":           "(@ARG0@)",
		"unlikely":         "(@ARG0@)",
		"FASTLZ_LIKELY":    "(@ARG0@)",
		"FASTLZ_UNLIKELY":  "(@ARG0@)",
		"yyjson_likely":    "(@ARG0@)",
		"yyjson_unlikely":  "(@ARG0@)",
		"YYJSON_LIKELY":    "(@ARG0@)",
		"YYJSON_UNLIKELY":  "(@ARG0@)",
		"XXH_LIKELY":       "(@ARG0@)",
		"XXH_UNLIKELY":     "(@ARG0@)",
		"yyjson_constcast": "(@ARG0@)",
		"constcast":        "(@ARG0@)",
		"__builtin_expect": "@ARG0@",
	} // name -> body with @ARG0@ for single-arg simple

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
					funcRepl[name] = "(@ARG0@)"
				} else if b == "#"+p || b == "# "+p {
					funcRepl[name] = "\"@ARG0@\""
				} else if strings.Contains(b, "__builtin_expect") {
					funcRepl[name] = "(@ARG0@)"
				} else if strings.Contains(b, p) {
					re := regexp.MustCompile(`\b` + regexp.QuoteMeta(p) + `\b`)
					funcRepl[name] = re.ReplaceAllLiteralString(b, "@ARG0@")
				}
			} else if len(params) >= 2 && body != "" {
				// multi-arg: #define FOR(i,n) for (i = 0; i < n; ++i)
				b := body
				for i, p := range params {
					re := regexp.MustCompile(`\b` + regexp.QuoteMeta(p) + `\b`)
					b = re.ReplaceAllLiteralString(b, fmt.Sprintf("@ARG%d@", i))
				}
				funcRepl[name] = b
			}
			// drop #define line
			continue
		}
		if body == "" {
			objs = append(objs, objDef{name, ""})
			continue
		}
		if name == "UTF8PROC_VERSION" {
			body = `"2.11.3"`
		}
		objs = append(objs, objDef{name, body})
	}

	// extract C enum definitions and merge into constant pool
	enums := harvestEnums(src)
	for k, v := range enums {
		objs = append(objs, objDef{k, strconv.FormatInt(v, 10)})
	}
	objs = append(objs, objDef{"DBL_EPSILON", "2.2204460492503131e-16"})
	objs = append(objs, objDef{"FLT_EPSILON", "1.19209290e-7"})
	objs = append(objs, objDef{"INT_MAX", "2147483647"})
	objs = append(objs, objDef{"INT_MIN", "-2147483648"})
	objs = append(objs, objDef{"UINT_MAX", "4294967295"})
	objs = append(objs, objDef{"SIZE_MAX", "18446744073709551615"})
	objs = append(objs, objDef{"SSIZE_MAX", "9223372036854775807"})
	objs = append(objs, objDef{"NULL", "0"})
	objs = append(objs, objDef{"null", "0"})
	objs = append(objs, objDef{"UTF8PROC_VERSION", "\"2.11.3\""})
	objs = append(objs, objDef{"__LINE__", "1"})
	objs = append(objs, objDef{"__UINT64_TYPE__", "uint64_t"})
	objs = append(objs, objDef{"__INT64_TYPE__", "int64_t"})
	objs = append(objs, objDef{"__UINT32_TYPE__", "uint32_t"})
	objs = append(objs, objDef{"__INT32_TYPE__", "int32_t"})
	objs = append(objs, objDef{"__SIZE_TYPE__", "size_t"})
	objs = append(objs, objDef{"__UINTPTR_TYPE__", "uintptr_t"})

	// evaluate object macros to integers when possible (multi-pass)
	vals := map[string]int64{}
	for k, v := range enums {
		vals[k] = v
	}
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
		// empty / multi-stmt / do-while bodies / string literals: no paren wrap
		if d.body == "" || strings.Contains(d.body, ";") || strings.Contains(d.body, "{") || (strings.HasPrefix(d.body, "\"") && strings.HasSuffix(d.body, "\"")) {
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
			if !strings.Contains(text, name) {
				continue
			}
			re := regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\s*\(`)
			var b strings.Builder
			last := 0
			for {
				loc := re.FindStringIndex(text[last:])
				if loc == nil {
					b.WriteString(text[last:])
					break
				}
				start := last + loc[0]
				open := last + loc[1] - 1
				close, err := matchParenStr(text, open)
				if err != nil {
					b.WriteString(text[last:])
					break
				}
				b.WriteString(text[last:start])
				argStr := text[open+1 : close]
				args := splitDefineArgs(argStr)
				repl := body
				for i, a := range args {
					repl = strings.ReplaceAll(repl, fmt.Sprintf("@ARG%d@", i), a)
				}
				if len(args) > 0 {
					repl = strings.ReplaceAll(repl, "$ARG", strings.TrimSpace(argStr))
				}
				b.WriteString(repl)
				last = close + 1
			}
			text = b.String()
		}
		return text
	}
	expandObjs := func(text string) string {
		for _, name := range names {
			if !strings.Contains(text, name) {
				continue
			}
			re := regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\b`)
			text = re.ReplaceAllLiteralString(text, objRepl[name])
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

	text = strings.ReplaceAll(text, "##", "")
	reStrConcat := regexp.MustCompile(`"([^"\n]*)"\s*"([^"\n]*)"`)
	for pass := 0; pass < 8; pass++ {
		if !reStrConcat.MatchString(text) {
			break
		}
		text = reStrConcat.ReplaceAllString(text, `"$1$2"`)
	}
	// drop remaining # lines (include, if, pragma, undef, …)
	var final []string
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		final = append(final, line)
	}
	res := strings.Join(final, "\n")
	return foldSizeof(res)
}

func foldSizeof(src string) string {
	reSizeof := regexp.MustCompile(`\bsizeof\s*\(`)
	var b strings.Builder
	last := 0
	for {
		loc := reSizeof.FindStringIndex(src[last:])
		if loc == nil {
			b.WriteString(src[last:])
			break
		}
		start := last + loc[0]
		open := last + loc[1] - 1
		close, err := matchParenStr(src, open)
		if err != nil {
			b.WriteString(src[last:])
			break
		}
		b.WriteString(src[last:start])
		inner := strings.TrimSpace(src[open+1 : close])
		t := strings.TrimSpace(inner)
		t = strings.TrimPrefix(t, "struct ")
		t = strings.TrimPrefix(t, "union ")
		val := "16"
		switch t {
		case "char", "uint8_t", "int8_t", "u8", "i8", "bool", "_Bool", "unsigned char", "signed char":
			val = "1"
		case "short", "uint16_t", "int16_t", "u16", "i16", "unsigned short", "signed short":
			val = "2"
		case "int", "uint32_t", "int32_t", "u32", "i32", "unsigned int", "signed int", "float", "float32":
			val = "4"
		case "long", "long long", "uint64_t", "int64_t", "u64", "i64", "usize", "size_t", "uintptr_t", "double", "float64", "unsigned long", "unsigned long long":
			val = "8"
		case "uint128_t", "int128_t", "u128", "i128", "__uint128_t", "unsigned __int128":
			val = "16"
		default:
			if strings.HasSuffix(t, "*") {
				val = "8"
			} else if st := findStruct(t, structEnv); st != nil {
				val = strconv.Itoa(estimateStructSize(st))
			} else {
				// Identifiant de variable ou tableau local : ne pas replier ici,
				// laisser front.go évaluer la vraie taille depuis regs[var].localArr.
				b.WriteString(src[start : close+1])
				last = close + 1
				continue
			}
		}
		b.WriteString(val)
		last = close + 1
	}
	return b.String()
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
	reCast := regexp.MustCompile(`\(\s*(?:const\s+)?(?:uint(?:8|16|32|64)?_t|int(?:8|16|32|64)?_t|u(?:8|16|32|64)|i(?:8|16|32|64)|unsigned(?:\s+(?:int|char|short|long))?|signed(?:\s+(?:int|char|short|long))?|int|char|short|long|size_t|uintptr_t)\s*\)`)
	s = reCast.ReplaceAllString(s, "")
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

// harvestEnums parses C enum definitions into integer mappings.
func harvestEnums(src string) map[string]int64 {
	out := map[string]int64{}
	src = stripComments(src)
	re := regexp.MustCompile(`(?s)(?:typedef\s+)?enum(?:\s+[A-Za-z0-9_]+)?\s*\{(.*?)\}(?:\s*[A-Za-z0-9_]+)?\s*;`)
	for _, m := range re.FindAllStringSubmatch(src, -1) {
		body := m[1]
		val := int64(0)
		for _, item := range strings.Split(body, ",") {
			item = strings.TrimSpace(stripComments(item))
			if item == "" {
				continue
			}
			if eq := strings.IndexByte(item, '='); eq >= 0 {
				name := strings.TrimSpace(item[:eq])
				rhs := strings.TrimSpace(item[eq+1:])
				rhsClean := strings.TrimPrefix(strings.TrimPrefix(rhs, "(int)"), "(char)")
				rhsClean = strings.TrimSpace(rhsClean)
				if len(rhsClean) >= 3 && rhsClean[0] == '\'' && rhsClean[len(rhsClean)-1] == '\'' {
					inner := rhsClean[1 : len(rhsClean)-1]
					if inner == `\\` {
						val = int64('\\')
					} else if inner == `\0` {
						val = 0
					} else if len(inner) == 1 {
						val = int64(inner[0])
					}
					out[name] = val
				} else if v, ok := evalDefineExpr(rhsClean, out); ok {
					val = v
					out[name] = val
				} else if v, err := strconv.ParseInt(rhsClean, 0, 64); err == nil {
					val = v
					out[name] = val
				} else {
					out[name] = val
				}
			} else {
				out[item] = val
			}
			val++
		}
	}
	return out
}
