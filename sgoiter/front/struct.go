package front

import (
	"regexp"
	"strconv"
	"strings"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/ir"
)

type TypedefInfo struct {
	BaseType ir.TypeName
	IsArray  bool
	ArrayLen int
}

var typedefEnv map[string]TypedefInfo

func harvestTypedefs(src string) map[string]TypedefInfo {
	m := map[string]TypedefInfo{}
	re := regexp.MustCompile(`typedef\s+([A-Za-z0-9_]+)\s+([A-Za-z0-9_]+)\s*\[\s*(\d+)\s*\]\s*;`)
	for _, match := range re.FindAllStringSubmatch(src, -1) {
		tname := match[1]
		alias := match[2]
		alen, _ := strconv.Atoi(match[3])
		m[alias] = TypedefInfo{
			BaseType: mapType(tname),
			IsArray:  true,
			ArrayLen: alen,
		}
	}
	typedefEnv = m
	return m
}

// harvestStructs parses struct Tag { fields }; and typedef struct { fields } Name;
func harvestStructs(src string) []ir.StructType {
	re := regexp.MustCompile(`(?:typedef\s+)?struct(?:\s+([A-Za-z_][A-Za-z0-9_]*))?\s*\{([^}]*)\}\s*([A-Za-z_][A-Za-z0-9_]*)?\s*;`)
	var out []ir.StructType
	for _, m := range re.FindAllStringSubmatch(src, -1) {
		tag, body, name := m[1], m[2], m[3]
		if name == "" {
			name = tag
		}
		if name == "" {
			continue
		}
		body = stripComments(body)
		var fields []ir.StructField
		for _, line := range strings.Split(body, ";") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			// Drop leading const/volatile
			for {
				if strings.HasPrefix(line, "const ") {
					line = strings.TrimSpace(line[6:])
					continue
				}
				if strings.HasPrefix(line, "volatile ") {
					line = strings.TrimSpace(line[9:])
					continue
				}
				break
			}
			// Pointer field: TYPE *name  or TYPE* name (monocypher argon2 inputs.pass)
			rePtr := regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*\*+\s*([A-Za-z_][A-Za-z0-9_]*)\s*$`)
			if pm := rePtr.FindStringSubmatch(line); pm != nil {
				sfType := mapType(pm[1])
				if td, ok := typedefEnv[pm[1]]; ok && td.IsArray {
					sfType = td.BaseType
				}
				// Pointer fields: ArrayLen=-1 marks pointer-to-elem for emit (slice in Go).
				fields = append(fields, ir.StructField{Name: pm[2], Type: sfType, ArrayLen: -1})
				continue
			}
			// TYPE name  or TYPE name[N]
			reF := regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s+([A-Za-z_][A-Za-z0-9_]*)\s*(?::\s*\d+)?\s*(?:\[\s*(\d+)\s*\])?$`)
			fm := reF.FindStringSubmatch(line)
			if fm == nil {
				continue
			}
			sfType := mapType(fm[1])
			sfLen := 0
			if td, ok := typedefEnv[fm[1]]; ok && td.IsArray {
				sfType = td.BaseType
				sfLen = td.ArrayLen
			}
			if fm[3] != "" {
				sfLen, _ = strconv.Atoi(fm[3])
			}
			sf := ir.StructField{Name: fm[2], Type: sfType, ArrayLen: sfLen}
			fields = append(fields, sf)
		}
		if len(fields) > 0 {
			out = append(out, ir.StructType{Name: name, Fields: fields})
		}
	}
	return out
}

// harvestGlobalsExtra: pointer-to-string and zero-filled arrays
func harvestGlobalsExtra(src string, base []ir.Global) []ir.Global {
	out := append([]ir.Global{}, base...)
	have := map[string]bool{}
	for _, g := range out {
		have[g.Name] = true
	}
	// static const u8 *name = (const u8*)"..."  (spaces after normalize)
	rePtr := regexp.MustCompile(`(?:static\s+)?(?:const\s+)?(?:u8|uint8_t)\s*\*\s*([A-Za-z_][A-Za-z0-9_]*)\s*=\s*\(\s*(?:const\s+)?(?:u8|uint8_t)\s*\*\)\s*"([^"]*)"`)
	for _, m := range rePtr.FindAllStringSubmatch(src, -1) {
		if have[m[1]] {
			continue
		}
		out = append(out, ir.Global{Name: m[1], Data: m[2], Type: ir.TypUint8})
		have[m[1]] = true
	}
	// static const u8 name[N] = {0};
	reZ := regexp.MustCompile(`(?:static\s+)?(?:const\s+)?(?:u8|uint8_t)\s+([A-Za-z_][A-Za-z0-9_]*)\s*\[\s*(\d+)\s*\]\s*=\s*\{\s*0\s*\}`)
	for _, m := range reZ.FindAllStringSubmatch(src, -1) {
		if have[m[1]] {
			continue
		}
		n, _ := strconv.Atoi(m[2])
		out = append(out, ir.Global{Name: m[1], Type: ir.TypUint8, ZeroLen: n})
		have[m[1]] = true
	}
	// 2D first: static const T name[R][C] = { {..}, {..} };
	re2D := regexp.MustCompile(`(?s)(?:static\s+)?(?:const\s+)?(?:u64|uint64_t|u32|uint32_t|u8|uint8_t)\s+([A-Za-z_][A-Za-z0-9_]*)\s*\[\s*(\d+)\s*\]\s*\[\s*(\d+)\s*\]\s*=\s*\{`)
	for _, m := range re2D.FindAllStringSubmatchIndex(src, -1) {
		name := src[m[2]:m[3]]
		if have[name] {
			continue
		}
		rows, _ := strconv.Atoi(src[m[4]:m[5]])
		cols, _ := strconv.Atoi(src[m[6]:m[7]])
		// m[1] is end of full match which ends at '{'
		braceAt := m[1] - 1
		if braceAt < 0 || braceAt >= len(src) || src[braceAt] != '{' {
			sub := src[m[0]:]
			bi := strings.IndexByte(sub, '{')
			if bi < 0 {
				continue
			}
			braceAt = m[0] + bi
		}
		body, ok := extractBalancedBraces(src, braceAt)
		if !ok {
			continue
		}
		typ := ir.TypUint64
		head := src[m[0]:m[1]]
		if strings.Contains(head, "u8") || strings.Contains(head, "uint8") {
			typ = ir.TypUint8
		} else if strings.Contains(head, "u32") || strings.Contains(head, "uint32") {
			typ = ir.TypUint32
		}
		csv := flattenCInitList(body)
		out = append(out, ir.Global{Name: name, Type: typ, InitCSV: csv, Rows: rows, Cols: cols})
		have[name] = true
	}
	// static const u64/u32/u8 name[N] = { hex, hex, ...};
	reArr := regexp.MustCompile(`(?s)(?:static\s+)?(?:const\s+)?(?:u64|uint64_t|u32|uint32_t|u8|uint8_t)\s+([A-Za-z_][A-Za-z0-9_]*)\s*\[\s*(\d+)\s*\]\s*=\s*\{([^}]+)\}`)
	for _, m := range reArr.FindAllStringSubmatch(src, -1) {
		if have[m[1]] {
			continue
		}
		typ := ir.TypUint64
		head := m[0]
		if strings.Contains(head, "u8") || strings.Contains(head, "uint8") {
			typ = ir.TypUint8
		} else if strings.Contains(head, "u32") || strings.Contains(head, "uint32") {
			typ = ir.TypUint32
		}
		arrLen, _ := strconv.Atoi(m[2])
		csv := strings.ReplaceAll(m[3], "\n", ",")
		csv = regexp.MustCompile(`,+`).ReplaceAllString(csv, ",")
		var parts []string
		for _, p := range strings.Split(csv, ",") {
			if s := strings.TrimSpace(p); s != "" {
				parts = append(parts, s)
			}
		}
		if arrLen > 0 && len(parts) < arrLen {
			for len(parts) < arrLen {
				parts = append(parts, "0")
			}
		}
		csv = strings.Join(parts, ", ")
		out = append(out, ir.Global{Name: m[1], Type: typ, InitCSV: csv})
		have[m[1]] = true
	}
	// static const fe name = { … }; or static const fe name[N] = { … }; (typedef i32 fe[10])
	reFe := regexp.MustCompile(`(?s)static\s+const\s+fe\s+([A-Za-z_][A-Za-z0-9_]*)\s*(?:\[\s*(\d+)\s*\])?\s*=\s*\{`)
	for _, m := range reFe.FindAllStringSubmatchIndex(src, -1) {
		name := src[m[2]:m[3]]
		if have[name] {
			continue
		}
		rows := 1
		if m[4] != -1 && m[5] != -1 {
			rows, _ = strconv.Atoi(src[m[4]:m[5]])
		}
		total := rows * 10
		// find opening brace of init
		braceAt := strings.Index(src[m[0]:m[1]], "{")
		if braceAt < 0 {
			continue
		}
		braceAt = m[0] + braceAt
		body, ok := extractBalancedBraces(src, braceAt)
		if !ok {
			continue
		}
		csv := flattenCInitList(body)
		if csv == "" {
			csv = "1" // fe_one = {1}
		}
		var parts []string
		for _, p := range strings.Split(csv, ",") {
			if s := strings.TrimSpace(p); s != "" {
				parts = append(parts, s)
			}
		}
		for len(parts) < total {
			parts = append(parts, "0")
		}
		csv = strings.Join(parts, ", ")
		out = append(out, ir.Global{Name: name, Type: ir.TypInt, InitCSV: csv})
		have[name] = true
	}
	return out
}

// extractBalancedBraces returns inner content of {...} starting at openIdx ('{').
func extractBalancedBraces(src string, openIdx int) (string, bool) {
	if openIdx < 0 || openIdx >= len(src) || src[openIdx] != '{' {
		return "", false
	}
	depth := 0
	for i := openIdx; i < len(src); i++ {
		switch src[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return src[openIdx+1 : i], true
			}
		}
	}
	return "", false
}

// flattenCInitList turns nested C brace lists into a flat CSV of integer tokens.
func flattenCInitList(body string) string {
	var b strings.Builder
	tok := ""
	flush := func() {
		tok = strings.TrimSpace(tok)
		tok = strings.TrimRight(tok, "uUlL")
		if tok == "" {
			return
		}
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(tok)
		tok = ""
	}
	for i := 0; i < len(body); i++ {
		c := body[i]
		switch c {
		case '{', '}', ',', '\n', '\r', '\t', ' ':
			flush()
		default:
			tok += string(c)
		}
	}
	flush()
	return b.String()
}

// foldTypedefs replaces typedef aliases (u8→uint8_t style already mapped; i8 etc.)
func foldTypedefs(src string) string {
	// typedef TYPE alias;
	re := regexp.MustCompile(`typedef\s+((?:unsigned\s+)?[A-Za-z_][A-Za-z0-9_]*)\s+([A-Za-z_][A-Za-z0-9_]*)\s*;`)
	type alias struct{ from, to string }
	var aliases []alias
	src2 := re.ReplaceAllStringFunc(src, func(m string) string {
		sm := re.FindStringSubmatch(m)
		if sm == nil {
			return m
		}
		to, from := sm[1], sm[2]
		// skip struct typedefs handled elsewhere
		if to == "struct" {
			return m
		}
		aliases = append(aliases, alias{from, to})
		return "/* typedef " + from + " */"
	})
	// longest first
	for i := 0; i < len(aliases); i++ {
		for j := i + 1; j < len(aliases); j++ {
			if len(aliases[j].from) > len(aliases[i].from) {
				aliases[i], aliases[j] = aliases[j], aliases[i]
			}
		}
	}
	for _, a := range aliases {
		reA := regexp.MustCompile(`\b` + regexp.QuoteMeta(a.from) + `\b`)
		src2 = reA.ReplaceAllString(src2, a.to)
	}
	return src2
}

func isStructType(name string, structs []ir.StructType) bool {
	for _, s := range structs {
		if strings.EqualFold(s.Name, name) {
			return true
		}
	}
	return false
}

func findStruct(name string, structs []ir.StructType) *ir.StructType {
	for i := range structs {
		if strings.EqualFold(structs[i].Name, name) {
			return &structs[i]
		}
	}
	return nil
}

func findStructByParam(p string, structs []ir.StructType) *ir.StructType {
	p = strings.TrimSpace(p)
	// Name * id  or Name id
	re := regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*\*?\s*[A-Za-z_]`)
	m := re.FindStringSubmatch(p)
	if m == nil {
		return nil
	}
	return findStruct(m[1], structs)
}

func fieldOf(st *ir.StructType, field string) *ir.StructField {
	if st == nil {
		return nil
	}
	for i := range st.Fields {
		if strings.EqualFold(st.Fields[i].Name, field) {
			return &st.Fields[i]
		}
	}
	return nil
}

func estimateStructSize(st *ir.StructType) int {
	n := 0
	for _, f := range st.Fields {
		el := 8
		switch f.Type {
		case ir.TypUint8:
			el = 1
		case ir.TypUint32, ir.TypInt:
			el = 4
		case ir.TypUint64:
			el = 8
		}
		al := f.ArrayLen
		if al == 0 {
			al = 1
		}
		n += el * al
	}
	if n < 16 {
		n = 16
	}
	return n
}

func indexArrow(expr string) int {
	depth := 0
	for i := 0; i < len(expr)-1; i++ {
		switch expr[i] {
		case '(':
			depth++
		case ')':
			depth--
		case '-':
			if depth == 0 && expr[i+1] == '>' {
				return i
			}
		}
	}
	return -1
}

func findTernaryColon(expr string, q int) int {
	depth := 0
	for i := q + 1; i < len(expr); i++ {
		switch expr[i] {
		case '(':
			depth++
		case ')':
			depth--
		case '?':
			depth++ // nested ternary
		case ':':
			if depth == 0 {
				return i
			}
			if depth > 0 {
				depth--
			}
		}
	}
	return -1
}

// indexByteAtDepth finds b at parenthesis depth want (0 = top-level).
func indexByteAtDepth(expr string, b byte, want int) int {
	depth := 0
	for i := 0; i < len(expr); i++ {
		switch expr[i] {
		case '(':
			depth++
		case ')':
			depth--
		default:
			if expr[i] == b && depth == want {
				return i
			}
		}
	}
	return -1
}
