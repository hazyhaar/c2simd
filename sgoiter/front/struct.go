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
		baseTyp := mapType(tname)
		if tname == "i32" || tname == "int32_t" {
			baseTyp = ir.TypInt32
		}
		m[alias] = TypedefInfo{
			BaseType: baseTyp,
			IsArray:  true,
			ArrayLen: alen,
		}
	}
	typedefEnv = m
	return m
}

// harvestStructs parses struct/union Tag { fields }; and typedef struct/union { fields } Name;
func harvestStructs(src string) []ir.StructType {
	src = stripIfDefs(src)
	src = stripComments(src)
	src = regexp.MustCompile(`(?m)^\s*#.*$`).ReplaceAllString(src, "")
	src = regexp.MustCompile(`(?m)^\s*%.*$`).ReplaceAllString(src, "")
	re := regexp.MustCompile(`(?:typedef\s+)?(?:struct|union)(?:\s+([A-Za-z_][A-Za-z0-9_]*))?\s*\{([^}]*)\}\s*([A-Za-z_][A-Za-z0-9_]*)?\s*;`)
	var out []ir.StructType
	for _, m := range re.FindAllStringSubmatch(src, -1) {
		tag, body, name := m[1], m[2], m[3]
		if name == "" {
			name = tag
		}
		if name == "" {
			continue
		}
		var fields []ir.StructField
		for _, line := range strings.Split(body, ";") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			line = regexp.MustCompile(`\b(?:const|volatile|restrict|struct|union|CJSON_CDECL|CJSON_PUBLIC|YYJSON_INLINE|YYJSON_API|__cdecl|__stdcall|__fastcall)\b`).ReplaceAllString(line, "")
			line = regexp.MustCompile(`\s+`).ReplaceAllString(line, " ")
			line = strings.TrimSpace(line)
			mapFieldType := func(raw string) ir.TypeName {
				mapped := mapType(raw)
				if mapped == ir.TypInt && raw != "int" && raw != "signed" && raw != "int32_t" && raw != "i32" && raw != "int32" && raw != "cJSON_bool" {
					return ir.TypeName(raw)
				}
				return mapped
			}
			subLines := []string{line}
			if strings.Contains(line, ",") && !strings.Contains(line, "(") {
				parts := strings.Split(line, ",")
				first := strings.TrimSpace(parts[0])
				sp := strings.LastIndexAny(first, " \t")
				if sp > 0 {
					baseType := strings.TrimSpace(first[:sp])
					subLines = []string{first}
					for _, p := range parts[1:] {
						p = strings.TrimSpace(p)
						if p != "" {
							subLines = append(subLines, baseType+" "+p)
						}
					}
				}
			}
			for _, sline := range subLines {
				sline = strings.TrimSpace(sline)
				if sline == "" {
					continue
				}
				// Function pointer field: RET (*name)(ARGS) or RET *(*name)(ARGS) or (CALLCONV *name)(ARGS)
				reFuncPtr := regexp.MustCompile(`^(?:[A-Za-z_][A-Za-z0-9_]*\s*\*?\s*)?\(\s*(?:[A-Za-z_][A-Za-z0-9_]*\s+)?\*+([A-Za-z_][A-Za-z0-9_]*)\s*\)\s*\(([^)]*)\)$`)
				if fpm := reFuncPtr.FindStringSubmatch(sline); fpm != nil {
					fldName := fpm[1]
					argsRaw := strings.TrimSpace(fpm[2])
					retType := ""
					if idx := strings.Index(sline, "("); idx > 0 {
						retType = strings.TrimSpace(sline[:idx])
					}
					fnTyp := "any"
					if strings.Contains(retType, "*") && (strings.Contains(argsRaw, "size_t") && !strings.Contains(argsRaw, ",")) {
						fnTyp = "func(uint64) []byte"
					} else if (retType == "void" || retType == "") && (strings.Contains(argsRaw, "void") || strings.Contains(argsRaw, "pointer") || strings.Contains(argsRaw, "ptr")) && !strings.Contains(argsRaw, ",") {
						fnTyp = "func(any)"
					} else if strings.Contains(retType, "*") && strings.Contains(argsRaw, "void") && strings.Contains(argsRaw, "size_t") {
						fnTyp = "func(any, uint64) []byte"
					} else if strings.Contains(retType, "*") {
						fnTyp = "func(...any) []byte"
					} else {
						fnTyp = "func(...any) any"
					}
					fields = append(fields, ir.StructField{Name: fldName, Type: ir.TypeName(fnTyp), ArrayLen: 0})
					continue
				}
				// Pointer field: TYPE *name  or TYPE* name (monocypher argon2 inputs.pass)
				rePtr := regexp.MustCompile(`^((?:unsigned\s+)?[A-Za-z_][A-Za-z0-9_]*)\s*\*+\s*([A-Za-z_][A-Za-z0-9_]*)\s*$`)
				if pm := rePtr.FindStringSubmatch(sline); pm != nil {
					sfType := mapFieldType(pm[1])
					if pm[1] == "void" {
						sfType = ir.TypeName("any")
					}
					if td, ok := typedefEnv[pm[1]]; ok && td.IsArray {
						sfType = td.BaseType
					}
					// Pointer fields: ArrayLen=-1 marks pointer-to-elem for emit (slice in Go).
					fields = append(fields, ir.StructField{Name: pm[2], Type: sfType, ArrayLen: -1})
					continue
				}
				// TYPE name  or TYPE name[N]
				reF := regexp.MustCompile(`^((?:unsigned\s+)?[A-Za-z_][A-Za-z0-9_]*)\s+([A-Za-z_][A-Za-z0-9_]*)\s*(?::\s*\d+)?\s*(?:\[\s*(\d+)\s*\])?$`)
				fm := reF.FindStringSubmatch(sline)
				if fm == nil {
					continue
				}
				sfType := mapFieldType(fm[1])
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
	// static const T name = value; (scalar global constant integer)
	reScalarConst := regexp.MustCompile(`(?s)(?:static\s+)?(?:const\s+)?([A-Za-z_][A-Za-z0-9_]*)\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*([0-9a-fA-FxX]+|\d+)\s*;`)
	for _, m := range reScalarConst.FindAllStringSubmatch(src, -1) {
		typeStr := m[1]
		name := m[2]
		valStr := m[3]
		if have[name] {
			continue
		}
		if typeStr == "struct" || typeStr == "typedef" || typeStr == "return" || typeStr == "goto" || typeStr == "if" || typeStr == "while" || typeStr == "for" {
			continue
		}
		num, err := parseIntLit(valStr)
		if err != nil {
			continue
		}
		typ := mapType(typeStr)
		if typ == "" {
			typ = ir.TypUint64
		}
		out = append(out, ir.Global{Name: name, Type: typ, Value: num, InitCSV: valStr})
		have[name] = true
	}
	re2D := regexp.MustCompile(`(?s)(?:static\s+)?(?:const\s+)?([A-Za-z_][A-Za-z0-9_]*)\s+([A-Za-z_][A-Za-z0-9_]*)\s*\[\s*(\d*)\s*\]\s*\[\s*(\d+)\s*\]\s*=\s*\{`)
	for _, m := range re2D.FindAllStringSubmatchIndex(src, -1) {
		typeStr := src[m[2]:m[3]]
		name := src[m[4]:m[5]]
		if have[name] {
			continue
		}
		rows, _ := strconv.Atoi(src[m[6]:m[7]])
		cols, _ := strconv.Atoi(src[m[8]:m[9]])
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
		typ := mapType(typeStr)
		if typ == "" {
			typ = ir.TypUint64
		}
		csv := flattenCInitList(body)
		out = append(out, ir.Global{Name: name, Type: typ, InitCSV: csv, Rows: rows, Cols: cols})
		have[name] = true
	}
	// static const T name[N] = { ... }; or static const T name[] = { ... };
	reArrGen := regexp.MustCompile(`(?s)(?:static\s+)?(?:const\s+)?([A-Za-z_][A-Za-z0-9_]*)\s+([A-Za-z_][A-Za-z0-9_]*)\s*\[\s*(\d*)\s*\]\s*=\s*\{`)
	for _, m := range reArrGen.FindAllStringSubmatchIndex(src, -1) {
		typeStr := src[m[2]:m[3]]
		name := src[m[4]:m[5]]
		if have[name] {
			continue
		}
		if typeStr == "struct" || typeStr == "static" || typeStr == "const" || typeStr == "typedef" || typeStr == "return" {
			continue
		}
		braceAt := strings.Index(src[m[0]:], "{")
		if braceAt < 0 {
			continue
		}
		braceAt = m[0] + braceAt
		body, ok := extractBalancedBraces(src, braceAt)
		if !ok {
			continue
		}
		typ := mapType(typeStr)
		if isStructType(typeStr, structEnv) {
			typ = ir.TypeName(typeStr)
		}
		arrLen := 0
		if m[6] != -1 && m[7] != -1 {
			arrLen, _ = strconv.Atoi(src[m[6]:m[7]])
		}
		csv := flattenCInitList(body)
		out = append(out, ir.Global{Name: name, Type: typ, InitCSV: csv, ZeroLen: arrLen})
		have[name] = true
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
		out = append(out, ir.Global{Name: name, Type: ir.TypInt32, InitCSV: csv})
		have[name] = true
	}
	// static const gf: gf0, gf1 = {1}, _121665 = {0xDB41,1}, ... (typedef i64 gf[16])
	reGfBlock := regexp.MustCompile(`(?s)static\s+const\s+gf\s+([^;]+);`)
	for _, m := range reGfBlock.FindAllStringSubmatch(src, -1) {
		decls := m[1]
		for _, decl := range splitCSVWithBraces(decls) {
			decl = strings.TrimSpace(decl)
			if decl == "" {
				continue
			}
			eqIdx := strings.IndexByte(decl, '=')
			if eqIdx < 0 {
				name := strings.TrimSpace(decl)
				if identOnly(name) && !have[name] {
					out = append(out, ir.Global{Name: name, Type: ir.TypInt64, ZeroLen: 16})
					have[name] = true
				}
			} else {
				name := strings.TrimSpace(decl[:eqIdx])
				initPart := strings.TrimSpace(decl[eqIdx+1:])
				if identOnly(name) && !have[name] {
					csv := flattenCInitList(initPart)
					var parts []string
					for _, p := range strings.Split(csv, ",") {
						if s := strings.TrimSpace(p); s != "" {
							parts = append(parts, s)
						}
					}
					for len(parts) < 16 {
						parts = append(parts, "0")
					}
					out = append(out, ir.Global{Name: name, Type: ir.TypInt64, InitCSV: strings.Join(parts, ", ")})
					have[name] = true
				}
			}
		}
	}
	// General static const <struct/type> name[N] or name = { ... };
	reStructArr := regexp.MustCompile(`(?s)(?:static\s+)?(?:const\s+)?(?:struct\s+)?([A-Za-z_][A-Za-z0-9_]*)\s+([A-Za-z_][A-Za-z0-9_]*)\s*(?:\[\s*(\d+)\s*\])?\s*=\s*\{`)
	for _, m := range reStructArr.FindAllStringSubmatchIndex(src, -1) {
		name := src[m[4]:m[5]]
		if have[name] {
			continue
		}
		typeName := src[m[2]:m[3]]
		if typeName == "if" || typeName == "while" || typeName == "for" || typeName == "return" || typeName == "typedef" {
			continue
		}
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
			continue
		}
		gtyp := ir.TypInt32
		if isStructType(typeName, structEnv) {
			gtyp = ir.TypeName(typeName)
		} else if strings.Contains(typeName, "u8") || strings.Contains(typeName, "char") || strings.Contains(typeName, "uint8") {
			gtyp = ir.TypUint8
		} else if strings.Contains(typeName, "u64") || strings.Contains(typeName, "uint64") || strings.Contains(typeName, "size_t") {
			gtyp = ir.TypUint64
		} else if strings.Contains(typeName, "u32") || strings.Contains(typeName, "uint32") {
			gtyp = ir.TypUint32
		}
		out = append(out, ir.Global{Name: name, Type: gtyp, InitCSV: csv})
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

func isQuote(s string, i int, q byte) bool {
	if s[i] != q {
		return false
	}
	nSlash := 0
	for j := i - 1; j >= 0 && s[j] == '\\'; j-- {
		nSlash++
	}
	return nSlash%2 == 0
}

// flattenCInitList turns nested C brace lists into a flat CSV of integer tokens.
func flattenCInitList(body string) string {
	var b strings.Builder
	tok := ""
	inChar := false
	inString := false
	flush := func() {
		tok = strings.Trim(strings.TrimSpace(tok), "{}")
		tok = strings.TrimSpace(tok)
		switch tok {
		case "NULL", "null":
			tok = "0"
		case "UINT16_MAX":
			tok = "65535"
		case "UINT8_MAX":
			tok = "255"
		case "UINT32_MAX":
			tok = "4294967295"
		case "INT_MAX":
			tok = "2147483647"
		case "INT_MIN":
			tok = "-2147483648"
		case "SIZE_MAX":
			tok = "18446744073709551615"
		case "SSIZE_MAX":
			tok = "9223372036854775807"
		default:
			tok = strings.TrimRight(tok, "uUlL")
		}
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
		if isQuote(body, i, '\'') {
			inChar = !inChar
			tok += string(c)
			continue
		}
		if isQuote(body, i, '"') {
			inString = !inString
			tok += string(c)
			continue
		}
		if inChar || inString {
			tok += string(c)
			continue
		}
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
	re := regexp.MustCompile(`(?:__extension__\s+)?typedef\s+((?:unsigned\s+)?[A-Za-z_][A-Za-z0-9_]*)\s+([A-Za-z_][A-Za-z0-9_]*)\s*;`)
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
		if to == "unsigned __int64" || to == "__uint64_t" {
			to = "uint64_t"
		} else if to == "__int64" || to == "__int64_t" {
			to = "int64_t"
		} else if to == "unsigned __int128" || to == "__uint128_t" {
			to = "uint128_t"
		} else if to == "__int128" || to == "__int128_t" {
			to = "int128_t"
		}
		aliases = append(aliases, alias{from, to})
		return "/* typedef " + from + " */"
	})
	reStructTypedef := regexp.MustCompile(`(?:__extension__\s+)?typedef\s+(?:struct|union)\s+([A-Za-z_][A-Za-z0-9_]*)\s+([A-Za-z_][A-Za-z0-9_]*)\s*;`)
	for _, m := range reStructTypedef.FindAllStringSubmatch(src, -1) {
		aliases = append(aliases, alias{m[2], "struct " + m[1]})
	}
	// Ne pas écraser les pointeurs de fonction avec void* afin de préserver leur signature dans les structures
	reEnumTypedef := regexp.MustCompile(`(?s)typedef\s+enum\s*(?:[A-Za-z_][A-Za-z0-9_]*)?\s*\{(.*?)\}\s*([A-Za-z_][A-Za-z0-9_]*)\s*;`)
	for _, m := range reEnumTypedef.FindAllStringSubmatch(src, -1) {
		aliases = append(aliases, alias{m[2], "int"})
	}
	reEnumNameTypedef := regexp.MustCompile(`typedef\s+enum\s+([A-Za-z_][A-Za-z0-9_]*)\s+([A-Za-z_][A-Za-z0-9_]*)\s*;`)
	for _, m := range reEnumNameTypedef.FindAllStringSubmatch(src, -1) {
		aliases = append(aliases, alias{m[2], "int"})
	}
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
	return findStruct(name, structs) != nil
}

func findStruct(name string, structs []ir.StructType) *ir.StructType {
	name = strings.TrimSpace(name)
	name = strings.TrimPrefix(name, "struct ")
	for i := range structs {
		if strings.EqualFold(structs[i].Name, name) {
			return &structs[i]
		}
	}
	if strings.HasSuffix(name, "_t") {
		cand := strings.TrimSuffix(name, "_t") + "_struct"
		for i := range structs {
			if strings.EqualFold(structs[i].Name, cand) {
				return &structs[i]
			}
		}
	}
	return nil
}

func findStructByParam(p string, structs []ir.StructType) *ir.StructType {
	p = strings.TrimSpace(p)
	p = regexp.MustCompile(`\b(?:const|volatile|restrict|struct)\b`).ReplaceAllString(p, "")
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

func lastIndexArrow(expr string) int {
	depth := 0
	for i := len(expr) - 2; i >= 0; i-- {
		switch expr[i+1] {
		case ')', ']':
			depth++
		case '(', '[':
			depth--
		}
		if depth == 0 && expr[i] == '-' && expr[i+1] == '>' {
			return i
		}
	}
	return -1
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

func splitCSVWithBraces(s string) []string {
	var parts []string
	depth := 0
	start := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '{' {
			depth++
		} else if c == '}' {
			depth--
		} else if c == ',' && depth == 0 {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		parts = append(parts, s[start:])
	}
	return parts
}
