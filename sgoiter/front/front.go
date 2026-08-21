// Package front — parseur C subset sgoiter → IR (harvest partiel, v0.3).
//
// v0.3 : pointeurs scalaires T*/index, for C-style simple, switch+fallthrough,
// littéraux ULL/LL, casts.
package front

import (
	"crypto/sha256"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/ir"
)

type ErrorCode string

const (
	ErrAsm     ErrorCode = "err_asm"
	ErrVarargs ErrorCode = "err_varargs"
	ErrGoto    ErrorCode = "err_goto"
	ErrMemory  ErrorCode = "err_memory"
	ErrFloat   ErrorCode = "err_float"
	ErrParse   ErrorCode = "err_parse"
	ErrEmpty   ErrorCode = "err_empty"
	ErrInclude ErrorCode = "err_include"
)

type Error struct {
	Code    ErrorCode
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("sgoiter/front: %s: %s", e.Code, e.Message)
}

type Result struct {
	Module  *ir.Module
	Skipped []string
}

const typeCandidates = `void|int|int8_t|int16_t|int32_t|int64_t|uint8_t|uint16_t|uint32_t|uint64_t|size_t|usize|uintptr_t|u8|u16|u32|u64|uint32|uint64|u128|i128|float|double|char|short|long|bool|_Bool|nk_f64_t|nk_f32_t|nk_i32_t|nk_u32_t|nk_i64_t|nk_u64_t|nk_size_t|nk_f16_t|nk_bf16_t|nk_i8_t|nk_u8_t|PCRE2_SPTR|PCRE2_SIZE|PCRE2_UCHAR|cJSON_bool|cJSON|printbuffer|parse_buffer|[A-Za-z0-9_]+_t`

func ParseFile(path string) (*ir.Module, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, &Error{Code: ErrParse, Message: err.Error()}
	}
	folded, err := foldLocalIncludes(string(b), filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	res, err := ParsePartial(folded, baseName(path))
	if err != nil {
		return nil, err
	}
	if res != nil && len(res.Skipped) > 0 {
		for _, s := range res.Skipped {
			fmt.Fprintf(os.Stderr, "sgoiter/front: skipped %s\n", s)
		}
	}
	if res == nil || res.Module == nil {
		return nil, &Error{Code: ErrParse, Message: "no function harvested"}
	}
	return res.Module, nil
}

// staticFuncPat matches a `static` (or `sv`, tweetnacl's shorthand) function
// definition in untouched C.
var staticFuncPat = regexp.MustCompile(`(?m)^\s*(?:static\s+\w[\w \t*]*?|sv)\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`)

// harvestStaticNames lists the functions a C file keeps to itself. They are not
// entry points, which is what a root-closure emission needs to know.
func harvestStaticNames(src string) map[string]bool {
	out := map[string]bool{}
	for _, m := range staticFuncPat.FindAllStringSubmatch(src, -1) {
		out[m[1]] = true
	}
	return out
}

func Parse(src, moduleName string) (*ir.Module, error) {
	res, err := ParsePartial(src, moduleName)
	if err != nil {
		return nil, err
	}
	return res.Module, nil
}

func ParsePartial(src, moduleName string) (*Result, error) {
	if err := rejectHard(src); err != nil {
		return nil, err
	}
	rawSrc := src
	m := ir.NewModule(moduleName)
	harvestTypedefs(rawSrc)
	m.Structs = harvestStructs(rawSrc)
	// normalize() erases the `static` keyword to simplify parsing, so the names
	// are collected from the untouched source beforehand.
	staticNames := harvestStaticNames(rawSrc)

	src = normalize(src)
	if strings.TrimSpace(src) == "" {
		return nil, &Error{Code: ErrEmpty, Message: "empty after normalize"}
	}

	// module-level type env for nested parse
	structEnv = m.Structs
	cleanRaw := stripComments(rawSrc)
	foldedRaw := foldDefines(cleanRaw)
	topLevelRaw := stripFunctionBodies(foldedRaw)
	m.Globals = harvestGlobalsExtra(topLevelRaw, harvestGlobals(topLevelRaw))

	reFunc := regexp.MustCompile(`(?s)(?:(?:static|inline|extern|__inline__|__forceinline|const|volatile|unsigned|signed|CJSON_CDECL|CJSON_PUBLIC|YYJSON_INLINE|YYJSON_API|YYJSON_FAST_INLINE|UTF8PROC_DLLEXPORT|__cdecl|__stdcall|__fastcall)\s+)*\b(` + typeCandidates + `)\b\s*(\*)?\s*(?:(?:CJSON_CDECL|CJSON_PUBLIC|YYJSON_INLINE|YYJSON_API|YYJSON_FAST_INLINE|UTF8PROC_DLLEXPORT|__cdecl|__stdcall|__fastcall)\s+)*([A-Za-z_][A-Za-z0-9_]*)\s*\(([^);]*)\)\s*\{`)
	locs := reFunc.FindAllStringSubmatchIndex(src, -1)
	if len(locs) == 0 {
		// No function headers matched. This could be a data-only file (e.g. ccitt_tables.c)
		// with only structs/globals. Return the module as is.
		return &Result{Module: m}, nil
	}

	var skipped []string
	for _, loc := range locs {
		ret := src[loc[2]:loc[3]]
		name := src[loc[6]:loc[7]]
		if loc[4] != -1 && loc[5] != -1 && strings.TrimSpace(src[loc[4]:loc[5]]) == "*" {
			ret = ret + "*"
		}
		paramsRaw := src[loc[8]:loc[9]]
		body, err := extractBlock(src, loc[1]-1)
		if err != nil {
			skipped = append(skipped, name+": block")
			continue
		}
		if err := rejectFuncBody(name, paramsRaw, body); err != nil {
			skipped = append(skipped, name+": "+err.Error())
			continue
		}
		pendingGlobals = nil
		fn, err := parseFunc(ret, name, paramsRaw, body, m.Globals, m.Structs)
		if err != nil {
			skipped = append(skipped, name+": "+err.Error())
			continue
		}
		// Hoisted local const tables (e.g. blake2b sigma[12][16])
		if len(pendingGlobals) > 0 {
			have := map[string]bool{}
			for _, g := range m.Globals {
				have[g.Name] = true
			}
			for _, g := range pendingGlobals {
				if !have[g.Name] {
					m.Globals = append(m.Globals, g)
					have[g.Name] = true
				}
			}
			// Re-bind regs that point at hoisted names via global: Sym already in body
			pendingGlobals = nil
		}
		fn.Flatten()
		fn.Static = staticNames[name]
		m.Funcs = append(m.Funcs, *fn)
	}
	if len(m.Funcs) == 0 {
		msg := "no function harvested"
		if len(skipped) > 0 {
			msg += "; skipped: " + strings.Join(skipped, " | ")
		}
		return nil, &Error{Code: ErrEmpty, Message: msg}
	}
	return &Result{Module: m, Skipped: skipped}, nil
}

func rejectHard(src string) error {
	s := stripComments(src)
	// memory barriers with an EMPTY instruction string (__asm__ volatile(""::"m"(x)))
	// are no-ops — accept and drop them (yyjson gcc_*_barrier). Any other asm stays.
	s = stripAsmBarriers(stripIfDefs(s))
	if regexp.MustCompile(`(?i)\basm\b|__asm__`).MatchString(s) {
		return &Error{Code: ErrAsm, Message: "inline assembly not in subset"}
	}
	return nil
}

func rejectFuncBody(name, params, body string) error {
	s := params + " " + body
	if regexp.MustCompile(`\bva_list\b|,\s*\.\.\.`).MatchString(s) {
		return &Error{Code: ErrVarargs, Message: "varargs"}
	}
	if regexp.MustCompile(`\bunion\b`).MatchString(s) {
		return &Error{Code: ErrMemory, Message: "union"}
	}
	// bare "struct Foo" without typedef still rejected if not typedef'd away
	if regexp.MustCompile(`\bstruct\s+[A-Za-z_]`).MatchString(s) && !strings.Contains(s, "typedef") {
		// allow if only in comments already stripped; reject anonymous usage
		if !regexp.MustCompile(`typedef\s+struct`).MatchString(s) {
			// params like crypto_poly1305_ctx* are typedef names, OK
		}
	}
	// multi-level pointers ***
	if strings.Contains(s, "***") {
		return &Error{Code: ErrMemory, Message: "multi-pointer"}
	}
	if name == "crypto_argon2" || name == "crypto_eddsa_check_equation" {
		return &Error{Code: ErrMemory, Message: "complex struct blocks"}
	}
	return nil
}

func normalize(s string) string {
	s = stripComments(s)
	s = foldDefines(s) // #define object/function-like + drop other #
	s = stripAsmBarriers(s)
	s = foldTypedefs(s)
	// preserve unsigned meaning before stripping keyword
	s = regexp.MustCompile(`\bunsigned\s+short\s+int\b`).ReplaceAllString(s, "uint16_t")
	s = regexp.MustCompile(`\bunsigned\s+short\b`).ReplaceAllString(s, "uint16_t")
	s = regexp.MustCompile(`\bsigned\s+short\s+int\b`).ReplaceAllString(s, "int16_t")
	s = regexp.MustCompile(`\bsigned\s+short\b`).ReplaceAllString(s, "int16_t")
	s = regexp.MustCompile(`\bshort\s+int\b`).ReplaceAllString(s, "int16_t")
	s = regexp.MustCompile(`\bshort\b`).ReplaceAllString(s, "int16_t")
	s = regexp.MustCompile(`\bunsigned\s+long\s+long\b`).ReplaceAllString(s, "uint64_t")
	s = regexp.MustCompile(`\bsigned\s+long\s+long\b`).ReplaceAllString(s, "int64_t")
	s = regexp.MustCompile(`\blong\s+long\b`).ReplaceAllString(s, "int64_t")
	s = regexp.MustCompile(`\b(?:unsigned\s+long\s+int|long\s+unsigned\s+int|long\s+unsigned|unsigned\s+long)\b`).ReplaceAllString(s, "uint64_t")
	s = regexp.MustCompile(`\b(?:signed\s+long\s+int|long\s+signed\s+int|long\s+signed|signed\s+long)\b`).ReplaceAllString(s, "int64_t")
	s = regexp.MustCompile(`\blong\s+int\b`).ReplaceAllString(s, "int64_t")
	s = regexp.MustCompile(`\blong\b`).ReplaceAllString(s, "int64_t")
	s = regexp.MustCompile(`\bunsigned\s+int\b`).ReplaceAllString(s, "uint32_t")
	s = regexp.MustCompile(`\bsigned\s+int\b`).ReplaceAllString(s, "int32_t")
	s = regexp.MustCompile(`\bunsigned\s+char\b`).ReplaceAllString(s, "uint8_t")
	s = regexp.MustCompile(`\bsigned\s+char\b`).ReplaceAllString(s, "int8_t")
	s = regexp.MustCompile(`\bunsigned\b`).ReplaceAllString(s, "uint32_t")
	s = regexp.MustCompile(`\bsigned\b`).ReplaceAllString(s, "int32_t")
	s = regexp.MustCompile(`\bvolatile\b`).ReplaceAllString(s, " ")
	s = regexp.MustCompile(`\bstatic\b`).ReplaceAllString(s, " ")
	s = regexp.MustCompile(`\bconst\b`).ReplaceAllString(s, " ")
	s = regexp.MustCompile(`\b(CJSON_CDECL|CJSON_PUBLIC|YYJSON_INLINE|YYJSON_API|YYJSON_FAST_INLINE|UTF8PROC_DLLEXPORT|__cdecl|__stdcall|__fastcall)\b`).ReplaceAllString(s, " ")
	s = regexp.MustCompile(`\bNULL\b`).ReplaceAllString(s, "0")
	// strip multi-pointer casts: (char**)&x -> &x
	s = regexp.MustCompile(`\(\s*(?:const\s+)?[A-Za-z_][A-Za-z0-9_]*\s*\*+\s*\*+\s*\)`).ReplaceAllString(s, "")
	// strip function prototypes ending with semicolon
	s = regexp.MustCompile(`(?m)^\s*(?:(?:static|inline|extern|__inline__|__forceinline|const|volatile|unsigned|signed)\s+)*\b(?:`+typeCandidates+`)\b\s*\*?\s*[A-Za-z_][A-Za-z0-9_]*\s*\([^);]*\)\s*;\s*$`).ReplaceAllString(s, "")
	reParenCall := regexp.MustCompile(`\(\s*([A-Za-z_][A-Za-z0-9_]*)\s*\)\s*\(`)
	s = reParenCall.ReplaceAllStringFunc(s, func(m string) string {
		sm := reParenCall.FindStringSubmatch(m)
		if sm != nil && !isTypeToken(sm[1]) {
			return sm[1] + "("
		}
		return m
	})
	// collapse whitespace (keep string literals intact via crude protect)
	s = protectStringsAndCollapseWS(s)
	return s
}

func protectStringsAndCollapseWS(s string) string {
	var out strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '"' {
			// copy string literal
			j := i + 1
			out.WriteByte('"')
			for j < len(s) {
				out.WriteByte(s[j])
				if s[j] == '\\' && j+1 < len(s) {
					j++
					out.WriteByte(s[j])
					j++
					continue
				}
				if s[j] == '"' {
					j++
					break
				}
				j++
			}
			i = j
			continue
		}
		if s[i] == '\'' {
			j := i + 1
			out.WriteByte('\'')
			for j < len(s) {
				out.WriteByte(s[j])
				if s[j] == '\\' && j+1 < len(s) {
					j++
					out.WriteByte(s[j])
					j++
					continue
				}
				if s[j] == '\'' {
					j++
					break
				}
				j++
			}
			i = j
			continue
		}
		if s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r' {
			out.WriteByte(' ')
			for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
				i++
			}
			continue
		}
		out.WriteByte(s[i])
		i++
	}
	return out.String()
}

func stripComments(s string) string {
	var out strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '"' {
			out.WriteByte(s[i])
			i++
			for i < len(s) {
				out.WriteByte(s[i])
				if s[i] == '\\' && i+1 < len(s) {
					i++
					out.WriteByte(s[i])
					i++
					continue
				}
				if s[i] == '"' {
					i++
					break
				}
				i++
			}
			continue
		}
		if s[i] == '\'' {
			out.WriteByte(s[i])
			i++
			for i < len(s) {
				out.WriteByte(s[i])
				if s[i] == '\\' && i+1 < len(s) {
					i++
					out.WriteByte(s[i])
					i++
					continue
				}
				if s[i] == '\'' {
					i++
					break
				}
				i++
			}
			continue
		}
		if i+1 < len(s) && s[i] == '/' && s[i+1] == '*' {
			i += 2
			for i+1 < len(s) && !(s[i] == '*' && s[i+1] == '/') {
				if s[i] == '\n' {
					out.WriteByte('\n')
				}
				i++
			}
			i += 2
			continue
		}
		if i+1 < len(s) && s[i] == '/' && s[i+1] == '/' {
			i += 2
			for i < len(s) && s[i] != '\n' {
				i++
			}
			continue
		}
		out.WriteByte(s[i])
		i++
	}
	return out.String()
}

func extractBlock(src string, openIdx int) (string, error) {
	if openIdx < 0 || openIdx >= len(src) || src[openIdx] != '{' {
		return "", &Error{Code: ErrParse, Message: "block start"}
	}
	depth := 0
	inChar := false
	inString := false
	for i := openIdx; i < len(src); i++ {
		if src[i] == '\\' && (inChar || inString) {
			i++
			continue
		}
		if src[i] == '\'' && !inString {
			inChar = !inChar
			continue
		}
		if src[i] == '"' && !inChar {
			inString = !inString
			continue
		}
		if inChar || inString {
			continue
		}
		switch src[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return src[openIdx+1 : i], nil
			}
		}
	}
	return "", &Error{Code: ErrParse, Message: "unclosed block"}
}

func stripFunctionBodies(src string) string {
	reFunc := regexp.MustCompile(`(?s)(?:(?:static|inline|extern|__inline__|__forceinline|const|volatile|unsigned|signed|CJSON_CDECL|CJSON_PUBLIC|YYJSON_INLINE|YYJSON_API|YYJSON_FAST_INLINE|UTF8PROC_DLLEXPORT|__cdecl|__stdcall|__fastcall)\s+)*\b(` + typeCandidates + `)\b\s*(\*)?\s*(?:(?:CJSON_CDECL|CJSON_PUBLIC|YYJSON_INLINE|YYJSON_API|YYJSON_FAST_INLINE|UTF8PROC_DLLEXPORT|__cdecl|__stdcall|__fastcall)\s+)*([A-Za-z_][A-Za-z0-9_]*)\s*\(([^);]*)\)\s*\{`)
	locs := reFunc.FindAllStringSubmatchIndex(src, -1)
	if len(locs) == 0 {
		return src
	}
	var sb strings.Builder
	last := 0
	for _, loc := range locs {
		openBrace := loc[1] - 1
		if openBrace < last {
			continue
		}
		sb.WriteString(src[last:loc[1]])
		body, err := extractBlock(src, openBrace)
		if err == nil {
			sb.WriteString("}")
			last = openBrace + 1 + len(body) + 1
		} else {
			last = loc[1]
		}
	}
	if last < len(src) {
		sb.WriteString(src[last:])
	}
	return sb.String()
}

type regInfo struct {
	v          ir.Value
	typ        ir.TypeName
	ptr        bool
	scale      int
	base       ir.Value // byte offset from root when hasBase
	hasBase    bool
	offSlot    ir.Value // mutable offset vreg (for ptr != end loops)
	offSlotSet bool
	elemIndex  bool
	localArr   int
	structName string // if pointer-to-struct or struct value
	cols       int    // 2D table row stride (0 = 1D)
}

var ptrMeta = map[ir.Value]regInfo{}
var structEnv []ir.StructType

func notePtr(v ir.Value, elem ir.TypeName, scale int, base ir.Value) {
	if scale == 0 {
		switch elem {
		case ir.TypUint32, ir.TypFloat32:
			scale = 4
		case ir.TypUint64, ir.TypFloat64, ir.TypInt64:
			scale = 8
		default:
			scale = 1
		}
	}
	ri := regInfo{v: v, typ: elem, ptr: true, scale: scale, elemIndex: true}
	if base != ir.NoVal {
		ri.hasBase = true
		ri.base = base
	}
	ptrMeta[v] = ri
}

// offSlotOnce holds one-time offSlot initializers (must not re-run in ForCondPrep).
var offSlotOnce []ir.Instr

// funcOffSlotPrologue: offSlot zero-inits hoisted to function entry (not inside loops).
var funcOffSlotPrologue []ir.Instr

func clearPtrMeta() {
	ptrMeta = map[ir.Value]regInfo{}
	offSlotOnce = nil
	funcOffSlotPrologue = nil
}

// scalePtrIndex maps C pointer index to root slice + byte/element index.
// Cast (uint64_t*)p with elemIndex=false → idx*8 on root []byte.
func scalePtrIndex(f *ir.Func, bv, iv ir.Value) (root, idx ir.Value, elem ir.TypeName, extra []ir.Instr) {
	root = bv
	idx = iv
	elem = ir.TypUint8
	scale := 1
	elemIndex := false
	var off ir.Value = ir.NoVal
	if pm, ok := ptrMeta[bv]; ok {
		if pm.typ != "" {
			elem = pm.typ
		}
		elemIndex = pm.elemIndex || pm.localArr > 0 || pm.structName != ""
		scale = pm.scale
		if elemIndex || scale == 0 {
			scale = 1
		}
		root = pm.v
		if pm.hasBase {
			off = pm.base
		}
	}
	if scale != 1 && !elemIndex {
		sc := f.Alloc()
		extra = append(extra, ir.Instr{Op: ir.OpConst, Dst: sc, Imm: int64(scale)})
		scaled := f.Alloc()
		extra = append(extra, ir.Instr{Op: ir.OpMul, Dst: scaled, Args: []ir.Value{iv, sc}})
		idx = scaled
	}
	if off != ir.NoVal {
		sum := f.Alloc()
		extra = append(extra, ir.Instr{Op: ir.OpAdd, Dst: sum, Args: []ir.Value{off, idx}})
		idx = sum
	}
	return root, idx, elem, extra
}

func harvestGlobals(src string) []ir.Global {
	// static const char/u8 name[N] = "..."; or static const char/u8 name[] = "...";
	re := regexp.MustCompile(`(?s)(?:static\s+)?(?:const\s+)?(?:unsigned\s+)?(?:char|u8|uint8_t)\s+([A-Za-z_][A-Za-z0-9_]*)\s*\[\s*\d*\s*\]\s*=\s*((?:"[^"]*"\s*)+);`)
	var out []ir.Global
	for _, m := range re.FindAllStringSubmatch(src, -1) {
		strParts := regexp.MustCompile(`"([^"]*)"`).FindAllStringSubmatch(m[2], -1)
		var combined string
		for _, p := range strParts {
			combined += p[1]
		}
		out = append(out, ir.Global{Name: m[1], Data: combined, Type: ir.TypUint8})
	}
	return out
}

func parseFunc(ret, name, paramsRaw, body string, globals []ir.Global, structs []ir.StructType) (*ir.Func, error) {
	clearPtrMeta()
	hoistedLocalGlobal = map[ir.Value]string{}
	structEnv = structs
	retT := mapType(ret)
	retClean := strings.TrimSpace(strings.TrimPrefix(strings.TrimSuffix(ret, "*"), "struct "))
	if isStructType(retClean, structs) {
		if strings.HasSuffix(ret, "*") {
			retT = ir.TypeName("*" + retClean)
		} else {
			retT = ir.TypeName(retClean)
		}
	}
	f := &ir.Func{Name: name, Result: retT}
	regs := map[string]regInfo{}

	// parse params first so globals don't shadow param names
	// (globals injected after params)

	paramsRaw = strings.TrimSpace(paramsRaw)
	if paramsRaw != "" && paramsRaw != "void" {
		for _, p := range splitCSV(paramsRaw) {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			typ, pname, ptr, alen, err := parseDeclarator(p)
			if err != nil {
				return nil, err
			}
			pClean := strings.TrimSpace(strings.TrimPrefix(p, "const "))
			if idx := strings.IndexAny(pClean, " \t*"); idx > 0 {
				pClean = pClean[:idx]
			}
			if td, ok := typedefEnv[pClean]; ok && td.IsArray {
				if alen > 0 {
					typ = ir.TypeName(fmt.Sprintf("[%d]%s", td.ArrayLen, td.BaseType))
					ptr = true
				} else {
					typ = td.BaseType
					ptr = true
					alen = td.ArrayLen
				}
			}
			r := f.Alloc()
			sname := ""
			if isStructType(string(typ), structs) || isStructType(p, structs) {
				// parseDeclarator may return struct name as typ via mapType default int — fix:
			}
			// detect struct pointer: crypto_poly1305_ctx *ctx
			if st := findStructByParam(p, structs); st != nil {
				sname = st.Name
				typ = ir.TypeName(st.Name)
				ptr = true
			}
			f.Params = append(f.Params, ir.Param{Name: pname, Type: typ, Ptr: ptr || alen > 0, ArrayLen: alen, StructName: sname, Reg: r})
			ri := regInfo{v: r, typ: typ, ptr: ptr || alen > 0, structName: sname, elemIndex: ptr || alen > 0}
			ptrMeta[r] = ri
			if ri.ptr && sname == "" {
				ri.scale = 1
				if typ == ir.TypUint32 {
					ri.scale = 4
				}
				if typ == ir.TypUint64 {
					ri.scale = 8
				}
				if alen > 0 {
					ri.scale = 1
					ri.elemIndex = true
				}
				notePtr(r, typ, ri.scale, ir.NoVal)
			}
			if sname != "" {
				notePtr(r, typ, 1, ir.NoVal)
				pm := ptrMeta[r]
				pm.structName = sname
				ptrMeta[r] = pm
			}
			regs[pname] = ri
		}
	}

	// Lazy offSlot: only if body bumps the pointer (name += / -=).
	// Avoids v=0 noise on kernels that only index p[i] (fnv/crc/xor).
	for pname, ri := range regs {
		if !ri.ptr || ri.structName != "" || ri.offSlotSet {
			continue
		}
		if ri.elemIndex && (ri.typ == ir.TypUint32 || ri.typ == ir.TypUint64) {
			continue
		}
		if ri.typ != ir.TypUint8 && ri.typ != "" && ri.typ != ir.TypInt {
			continue
		}
		if !bodyBumpsPtr(body, pname) {
			continue
		}
		slot := f.Alloc()
		z := f.Alloc()
		funcOffSlotPrologue = append(funcOffSlotPrologue,
			ir.Instr{Op: ir.OpConst, Dst: z, Imm: 0, Elem: ir.TypUint64},
			ir.Instr{Op: ir.OpMov, Dst: slot, Args: []ir.Value{z}, Elem: ir.TypUint64, Sym: "offslot"},
		)
		ri.offSlot = slot
		ri.offSlotSet = true
		ri.hasBase = true
		ri.base = slot
		regs[pname] = ri
		ptrMeta[ri.v] = ri
	}

	// globals referenced in body only; never shadow params
	var globalInit []ir.Instr
	for _, g := range globals {
		if _, shadow := regs[g.Name]; shadow {
			continue
		}
		if !regexp.MustCompile(`\b` + regexp.QuoteMeta(g.Name) + `\b`).MatchString(body) {
			continue
		}
		// skip single-letter globals without data unless they have InitCSV/ZeroLen
		// (monocypher uses L[8], A, d as real tables — must not drop)
		if len(g.Name) <= 1 && g.Data == "" && g.ZeroLen == 0 && g.InitCSV == "" {
			continue
		}
		isScalarGlobal := g.Value != 0 || (g.ZeroLen == 0 && g.Rows == 0 && g.Cols == 0 && g.Data == "" && g.InitCSV != "" && !strings.Contains(g.InitCSV, ","))
		r := f.Alloc()
		globalInit = append(globalInit, ir.Instr{Op: ir.OpMov, Dst: r, Sym: "global:" + g.Name})
		et := g.Type
		if et == "" {
			et = ir.TypUint8
		}
		if isScalarGlobal {
			regs[g.Name] = regInfo{v: r, typ: et, ptr: false, scale: 1}
			continue
		}
		ri := regInfo{v: r, typ: et, ptr: true, scale: 1, elemIndex: et == ir.TypUint8 || g.Data != "" || g.ZeroLen > 0 || g.InitCSV != "", cols: g.Cols}
		if et == ir.TypUint32 || et == ir.TypUint64 {
			ri.elemIndex = true
			ri.scale = 1
		}
		if g.Cols > 0 {
			ri.elemIndex = true
			ri.scale = 1
		}
		if isStructType(string(g.Type), structs) {
			ri.structName = string(g.Type)
		}
		regs[g.Name] = ri
		notePtr(r, et, 1, ir.NoVal)
		pm := ptrMeta[r]
		pm.elemIndex = ri.elemIndex
		pm.cols = ri.cols
		pm.structName = ri.structName
		ptrMeta[r] = pm
	}

	stmts, err := parseBlock(f, regs, body)
	if err != nil {
		return nil, err
	}
	// prepend global binds + offSlot prologue (hoisted out of loops)
	var head []ir.Stmt
	for _, ins := range globalInit {
		head = append(head, ir.Stmt{Kind: ir.SKInstr, Ins: ins})
	}
	for _, ins := range funcOffSlotPrologue {
		head = append(head, ir.Stmt{Kind: ir.SKInstr, Ins: ins})
	}
	funcOffSlotPrologue = nil
	f.Stmts = append(head, stmts...)
	return f, nil
}

// parseDeclarator returns typ, name, ptr, arrayLen, err.
func parseDeclarator(p string) (ir.TypeName, string, bool, int, error) {
	p = strings.TrimSpace(p)
	p = regexp.MustCompile(`\b(?:const|volatile|restrict|struct)\b`).ReplaceAllString(p, "")
	p = regexp.MustCompile(`\s+`).ReplaceAllString(p, " ")
	p = strings.TrimSpace(p)
	// TYPE * name
	// TYPE name
	// TYPE name[N]
	// TYPE name[]
	// TYPE * name[N]  — reject multi for now
	reArr := regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s+([A-Za-z_][A-Za-z0-9_]*)\s*\[\s*(\d+)\s*\]$`)
	if m := reArr.FindStringSubmatch(p); m != nil {
		n, _ := strconv.Atoi(m[3])
		return mapType(m[1]), m[2], true, n, nil // array decays to ptr
	}
	reArr2D := regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s+([A-Za-z_][A-Za-z0-9_]*)\s*\[\s*\]\s*\[\s*(\d+)\s*\]$`)
	if m := reArr2D.FindStringSubmatch(p); m != nil {
		typ := ir.TypeName(fmt.Sprintf("[][%s]%s", m[3], string(mapType(m[1]))))
		return typ, m[2], false, 0, nil
	}
	rePtrArr := regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*\(\s*\*\s*([A-Za-z_][A-Za-z0-9_]*)\s*\)\s*\[\s*(\d+)\s*\]$`)
	if m := rePtrArr.FindStringSubmatch(p); m != nil {
		typ := ir.TypeName(fmt.Sprintf("[][%s]%s", m[3], string(mapType(m[1]))))
		return typ, m[2], false, 0, nil
	}
	reArrEmpty := regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s+([A-Za-z_][A-Za-z0-9_]*)\s*\[\s*\]$`)
	if m := reArrEmpty.FindStringSubmatch(p); m != nil {
		return mapType(m[1]), m[2], true, 0, nil // T name[] decays to pointer
	}
	re := regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*(\*+)?\s*([A-Za-z_][A-Za-z0-9_]*)$`)
	m := re.FindStringSubmatch(p)
	if m == nil {
		return "", "", false, 0, &Error{Code: ErrParse, Message: "declarator: " + p}
	}
	typ := mapType(m[1])
	ptr := m[2] != ""
	if m[2] == "**" {
		if typ == ir.TypUint8 || typ == ir.TypInt8 || m[1] == "char" || m[1] == "unsigned char" {
			typ = ir.TypeName("*[]byte")
		} else {
			typ = ir.TypeName("*[]" + string(typ))
		}
	} else if m[1] == "void" && ptr {
		typ = ir.TypUint8
	}
	if isStructType(m[1], structEnv) {
		typ = ir.TypeName(m[1])
	}
	return typ, m[3], ptr, 0, nil
}

func parseBlock(f *ir.Func, regs map[string]regInfo, body string) ([]ir.Stmt, error) {
	var out []ir.Stmt
	rest := strings.TrimSpace(body)
	for rest != "" {
		rest = strings.TrimSpace(rest)
		if rest == "" {
			break
		}
		// do {
		if strings.HasPrefix(rest, "do") && nextNonSpace(rest[2:]) == '{' {
			st, n, err := parseDoWhile(f, regs, rest)
			if err != nil {
				return nil, err
			}
			out = append(out, st)
			rest = rest[n:]
			continue
		}
		// standalone label: ident:
		if reLbl := regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*:`).FindStringSubmatch(rest); reLbl != nil && reLbl[1] != "default" && reLbl[1] != "case" {
			lbl := reLbl[1]
			out = append(out, ir.Stmt{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpLabel, Sym: lbl}})
			colon := strings.IndexByte(rest, ':')
			rest = strings.TrimSpace(rest[colon+1:])
			continue
		}
		// if (
		if strings.HasPrefix(rest, "if") && nextNonSpace(rest[2:]) == '(' {
			st, n, err := parseIf(f, regs, rest)
			if err != nil {
				return nil, err
			}
			out = append(out, st)
			rest = rest[n:]
			continue
		}
		// while (
		if strings.HasPrefix(rest, "while") && nextNonSpace(rest[5:]) == '(' {
			st, n, err := parseWhile(f, regs, rest)
			if err != nil {
				return nil, err
			}
			out = append(out, st)
			rest = rest[n:]
			continue
		}
		// for (
		if strings.HasPrefix(rest, "for") && nextNonSpace(rest[3:]) == '(' {
			st, n, err := parseFor(f, regs, rest)
			if err != nil {
				return nil, err
			}
			out = append(out, st)
			rest = rest[n:]
			continue
		}
		// switch (
		if strings.HasPrefix(rest, "switch") && nextNonSpace(rest[6:]) == '(' {
			st, n, err := parseSwitch(f, regs, rest)
			if err != nil {
				return nil, err
			}
			out = append(out, st)
			rest = rest[n:]
			continue
		}
		// local/static array with brace initializer: TYPE name[N] = { … };
		// (indexStmtEnd stops at '{' so these must be peeled first)
		if n, stmts, err := tryArrayInitDecl(f, regs, rest); err != nil {
			return nil, err
		} else if n > 0 {
			out = append(out, stmts...)
			rest = rest[n:]
			continue
		}
		// statement ending with ;
		semi, err := indexStmtEnd(rest)
		if err != nil {
			return nil, err
		}
		if semi < 0 {
			// bare block?
			if strings.HasPrefix(rest, "{") {
				blk, err := extractBlock(rest, 0)
				if err != nil {
					return nil, err
				}
				inner, err := parseBlock(f, regs, blk)
				if err != nil {
					return nil, err
				}
				out = append(out, inner...)
				// consume {blk}
				end := 1 + len(blk) + 1
				rest = rest[end:]
				continue
			}
			return nil, &Error{Code: ErrParse, Message: "stmt tail: " + trunc(rest, 60)}
		}
		st := strings.TrimSpace(rest[:semi])
		rest = rest[semi+1:]
		if st == "" {
			continue
		}
		insList, err := parseSimpleStmt(f, regs, st)
		if err != nil {
			return nil, err
		}
		for _, ins := range insList {
			out = append(out, ir.Stmt{Kind: ir.SKInstr, Ins: ins})
		}
	}
	return out, nil
}

func nextNonSpace(s string) byte {
	s = strings.TrimLeft(s, " \t\n\r")
	if s == "" {
		return 0
	}
	return s[0]
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func indexStmtEnd(s string) (int, error) {
	depth := 0
	braceDepth := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\'' {
			j := i + 1
			for j < len(s) {
				if s[j] == '\\' {
					j += 2
					continue
				}
				if s[j] == '\'' {
					break
				}
				j++
			}
			if j < len(s) {
				i = j
				continue
			}
		}
		if c == '"' {
			j := i + 1
			for j < len(s) {
				if s[j] == '\\' {
					j += 2
					continue
				}
				if s[j] == '"' {
					break
				}
				j++
			}
			if j < len(s) {
				i = j
				continue
			}
		}
		switch c {
		case '(':
			depth++
		case ')':
			depth--
		case '{':
			if depth == 0 && strings.TrimSpace(s[:i]) == "" {
				return -1, nil
			}
			braceDepth++
		case '}':
			braceDepth--
		case ';':
			if depth == 0 && braceDepth <= 0 {
				return i, nil
			}
		}
	}
	return -1, nil
}

func parsePostIncAssign(f *ir.Func, regs map[string]regInfo, st string) ([]ir.Instr, error) {
	// forms: *a++ = *b++;  *a++ = e;
	st = strings.TrimSpace(st)
	if !strings.HasPrefix(st, "*") {
		return nil, &Error{Code: ErrParse, Message: "postinc"}
	}
	eq := strings.Index(st, "=")
	if eq < 0 {
		return nil, &Error{Code: ErrParse, Message: "postinc ="}
	}
	lhs := strings.TrimSpace(st[:eq])
	rhs := strings.TrimSpace(st[eq+1:])
	// lhs *name++
	reL := regexp.MustCompile(`^\*\s*([A-Za-z_][A-Za-z0-9_]*)\s*\+\+$`)
	ml := reL.FindStringSubmatch(lhs)
	if ml == nil {
		return nil, &Error{Code: ErrParse, Message: "postinc lhs"}
	}
	ld, ok := regs[ml[1]]
	if !ok || !ld.ptr {
		return nil, &Error{Code: ErrParse, Message: "postinc lhs ptr"}
	}
	var ins []ir.Instr
	var val ir.Value
	reR := regexp.MustCompile(`^\*\s*([A-Za-z_][A-Za-z0-9_]*)\s*\+\+$`)
	if mr := reR.FindStringSubmatch(rhs); mr != nil {
		rs, ok := regs[mr[1]]
		if !ok || !rs.ptr {
			return nil, &Error{Code: ErrParse, Message: "postinc rhs ptr"}
		}
		val = f.Alloc()
		ins = append(ins,
			ir.Instr{Op: ir.OpLoad, Dst: val, Args: []ir.Value{rs.v}, Elem: rs.typ},
			ir.Instr{Op: ir.OpMov, Dst: rs.v, Args: []ir.Value{rs.v}, Sym: "ptr_adv1"},
		)
	} else {
		v, more, err := pe(f, regs, rhs)
		if err != nil {
			return nil, err
		}
		ins = append(ins, more...)
		val = v
	}
	ins = append(ins,
		ir.Instr{Op: ir.OpStore, Args: []ir.Value{ld.v, val}, Elem: ld.typ},
		ir.Instr{Op: ir.OpMov, Dst: ld.v, Args: []ir.Value{ld.v}, Sym: "ptr_adv1"},
	)
	return ins, nil
}

func parseDoWhile(f *ir.Func, regs map[string]regInfo, s string) (ir.Stmt, int, error) {
	// do { body } while (cond);
	rest := strings.TrimSpace(s[2:])
	if !strings.HasPrefix(rest, "{") {
		return ir.Stmt{}, 0, &Error{Code: ErrParse, Message: "do {"}
	}
	bodySrc, err := extractBlock(rest, 0)
	if err != nil {
		return ir.Stmt{}, 0, err
	}
	bodyEnd := 1 + len(bodySrc) + 1 // relative to rest
	after := strings.TrimSpace(rest[bodyEnd:])
	if !strings.HasPrefix(after, "while") {
		return ir.Stmt{}, 0, &Error{Code: ErrParse, Message: "do while"}
	}
	after = strings.TrimSpace(after[5:])
	if !strings.HasPrefix(after, "(") {
		return ir.Stmt{}, 0, &Error{Code: ErrParse, Message: "while ("}
	}
	closeP, err := matchParen(after, 0)
	if err != nil {
		return ir.Stmt{}, 0, err
	}
	cond := strings.TrimSpace(after[1:closeP])
	// optional ;
	consumedFromAfter := closeP + 1
	tail := after[consumedFromAfter:]
	semiExtra := 0
	if strings.HasPrefix(strings.TrimSpace(tail), ";") {
		// find ;
		for i, c := range tail {
			if c == ';' {
				semiExtra = i + 1
				break
			}
		}
	}
	f.Body = nil
	// support --count / count-- as cond
	cl, cop, cr, err := parseCond(f, regs, cond)
	if err != nil {
		return ir.Stmt{}, 0, err
	}
	prep := f.Body
	f.Body = nil
	body, err := parseBlock(f, regs, bodySrc)
	if err != nil {
		return ir.Stmt{}, 0, err
	}
	// total from start of s: "do" + ws + {body} + ws + while(cond);
	// compute via lengths from s
	// s = do ...
	// find end position
	endPos := 2 // "do"
	// skip to {
	i := strings.Index(s[2:], "{")
	if i < 0 {
		return ir.Stmt{}, 0, &Error{Code: ErrParse, Message: "do brace"}
	}
	endPos = 2 + i + 1 + len(bodySrc) + 1
	// from endPos find while
	j := strings.Index(s[endPos:], "while")
	if j < 0 {
		return ir.Stmt{}, 0, &Error{Code: ErrParse, Message: "while kw"}
	}
	openP := strings.Index(s[endPos:], "(")
	if openP < 0 {
		return ir.Stmt{}, 0, &Error{Code: ErrParse, Message: "while ("}
	}
	closeP, err = matchParen(s[endPos:], openP)
	if err != nil {
		return ir.Stmt{}, 0, err
	}
	endPos = endPos + closeP + 1
	for endPos < len(s) && (s[endPos] == ' ' || s[endPos] == '\t' || s[endPos] == '\n' || s[endPos] == '\r') {
		endPos++
	}
	if endPos < len(s) && s[endPos] == ';' {
		endPos++
	}
	_ = prep
	_ = semiExtra
	st := ir.Stmt{
		Kind:      ir.SKDoWhile,
		DoBody:    body,
		CondLeft:  cl,
		CondOp:    cop,
		CondRight: cr,
		ForInit:   prep, // reuse ForInit as cond prep
	}
	return st, endPos, nil
}

func parseWhile(f *ir.Func, regs map[string]regInfo, s string) (ir.Stmt, int, error) {
	// while (cond) { body } | while (cond) stmt;
	i := strings.Index(s, "(")
	if i < 0 {
		return ir.Stmt{}, 0, &Error{Code: ErrParse, Message: "while ("}
	}
	closeP, err := matchParen(s, i)
	if err != nil {
		return ir.Stmt{}, 0, err
	}
	cond := strings.TrimSpace(s[i+1 : closeP])
	f.Body = nil
	cl, cop, cr, err := parseCond(f, regs, cond)
	if err != nil {
		return ir.Stmt{}, 0, err
	}
	prep := f.Body
	f.Body = nil
	restRaw := s[closeP+1:]
	body, bodyN, err := parseIfBranch(f, regs, restRaw) // same brace/stmt logic
	if err != nil {
		return ir.Stmt{}, 0, err
	}
	var once []ir.Instr
	if len(offSlotOnce) > 0 {
		once = append(once, offSlotOnce...)
		offSlotOnce = nil
	}
	// emit as SKFor; offSlot once in ForInit; expression temps in ForCondPrep
	st := ir.Stmt{
		Kind:        ir.SKFor,
		ForInit:     once,
		ForCondPrep: prep,
		CondLeft:    cl,
		CondOp:      cop,
		CondRight:   cr,
		ForBody:     body,
	}
	return st, closeP + 1 + bodyN, nil
}

func parseIf(f *ir.Func, regs map[string]regInfo, s string) (ir.Stmt, int, error) {
	// if (cond) stmt; | if (cond) { body } [else ...]
	i := strings.Index(s, "(")
	if i < 0 {
		return ir.Stmt{}, 0, &Error{Code: ErrParse, Message: "if ("}
	}
	closeP, err := matchParen(s, i)
	if err != nil {
		return ir.Stmt{}, 0, err
	}
	condRaw := stripOuterParens(strings.TrimSpace(s[i+1 : closeP]))
	neg := false
	for {
		if strings.HasPrefix(condRaw, "!") {
			neg = !neg
			condRaw = stripOuterParens(strings.TrimSpace(condRaw[1:]))
		} else {
			break
		}
	}
	f.Body = nil
	cl, cop, cr, err := parseCond(f, regs, condRaw)
	if err != nil {
		return ir.Stmt{}, 0, err
	}
	prep := f.Body
	f.Body = nil
	restRaw := s[closeP+1:]
	thenBody, thenN, err := parseIfBranch(f, regs, restRaw)
	if err != nil {
		return ir.Stmt{}, 0, err
	}
	total := closeP + 1 + thenN
	var elseBody []ir.Stmt
	tail := strings.TrimLeft(restRaw[thenN:], " \t\n\r")
	if strings.HasPrefix(tail, "else") && (len(tail) == 4 || !isIdentChar(tail[4])) {
		elseRel := strings.Index(restRaw[thenN:], "else")
		elseAt := thenN + elseRel
		afterElse := restRaw[elseAt+4:]
		eb, en, err := parseIfBranch(f, regs, afterElse)
		if err != nil {
			return ir.Stmt{}, 0, err
		}
		elseBody = eb
		total = closeP + 1 + elseAt + 4 + en
	}
	st := ir.Stmt{
		Kind:      ir.SKIf,
		ForInit:   prep,
		CondLeft:  cl,
		CondOp:    cop,
		CondRight: cr,
		CondNot:   neg,
		ThenBody:  thenBody,
		ElseBody:  elseBody,
	}
	return st, total, nil
}

func parseIfBranch(f *ir.Func, regs map[string]regInfo, s string) ([]ir.Stmt, int, error) {
	sTrim := strings.TrimLeft(s, " \t\n\r")
	pad := len(s) - len(sTrim)
	if strings.HasPrefix(sTrim, "{") {
		bodySrc, err := extractBlock(sTrim, 0)
		if err != nil {
			return nil, 0, err
		}
		body, err := parseBlock(f, regs, bodySrc)
		if err != nil {
			return nil, 0, err
		}
		return body, pad + 1 + len(bodySrc) + 1, nil
	}
	// nested if / else if
	if strings.HasPrefix(sTrim, "if") && (len(sTrim) == 2 || !isIdentChar(sTrim[2])) {
		st, n, err := parseIf(f, regs, sTrim)
		if err != nil {
			return nil, 0, err
		}
		return []ir.Stmt{st}, pad + n, nil
	}
	// nested do-while
	if strings.HasPrefix(sTrim, "do") && (len(sTrim) == 2 || !isIdentChar(sTrim[2])) {
		st, n, err := parseDoWhile(f, regs, sTrim)
		if err != nil {
			return nil, 0, err
		}
		return []ir.Stmt{st}, pad + n, nil
	}
	// nested while
	if strings.HasPrefix(sTrim, "while") && (len(sTrim) == 5 || !isIdentChar(sTrim[5])) {
		st, n, err := parseWhile(f, regs, sTrim)
		if err != nil {
			return nil, 0, err
		}
		return []ir.Stmt{st}, pad + n, nil
	}
	// nested for
	if strings.HasPrefix(sTrim, "for") && (len(sTrim) == 3 || !isIdentChar(sTrim[3])) {
		st, n, err := parseFor(f, regs, sTrim)
		if err != nil {
			return nil, 0, err
		}
		return []ir.Stmt{st}, pad + n, nil
	}
	// single stmt until ;
	semi, err := indexStmtEnd(sTrim)
	if err != nil {
		return nil, 0, err
	}
	if semi < 0 {
		return nil, 0, &Error{Code: ErrParse, Message: "if branch"}
	}
	ins, err := parseSimpleStmt(f, regs, strings.TrimSpace(sTrim[:semi]))
	if err != nil {
		return nil, 0, err
	}
	var body []ir.Stmt
	for _, in := range ins {
		body = append(body, ir.Stmt{Kind: ir.SKInstr, Ins: in})
	}
	return body, pad + semi + 1, nil
}

func parseFor(f *ir.Func, regs map[string]regInfo, s string) (ir.Stmt, int, error) {
	// for (init; cond; post) { body }
	i := strings.Index(s, "(")
	if i < 0 {
		return ir.Stmt{}, 0, &Error{Code: ErrParse, Message: "for ("}
	}
	closeP, err := matchParen(s, i)
	if err != nil {
		return ir.Stmt{}, 0, err
	}
	inner := s[i+1 : closeP]
	parts := splitSemi(inner)
	if len(parts) != 3 {
		return ir.Stmt{}, 0, &Error{Code: ErrParse, Message: "for parts"}
	}
	var initIns, postIns, condPrep []ir.Instr
	if strings.TrimSpace(parts[0]) != "" {
		initIns, err = parseSimpleStmt(f, regs, strings.TrimSpace(parts[0]))
		if err != nil {
			return ir.Stmt{}, 0, err
		}
	}
	cond := strings.TrimSpace(parts[1])
	f.Body = nil
	cl, cop, cr, err := parseCond(f, regs, cond)
	if err != nil {
		return ir.Stmt{}, 0, err
	}
	condPrep = f.Body
	f.Body = nil
	// offSlot first-time inits belong in ForInit (once), not CondPrep
	if len(offSlotOnce) > 0 {
		initIns = append(initIns, offSlotOnce...)
		offSlotOnce = nil
	}
	// Pure Const from cond (e.g. i < 8) → ForInit once; keeps ForCondPrep empty → for i < 8
	var eachPrep []ir.Instr
	for _, ins := range condPrep {
		if ins.Op == ir.OpConst {
			initIns = append(initIns, ins)
		} else {
			eachPrep = append(eachPrep, ins)
		}
	}
	condPrep = eachPrep
	if strings.TrimSpace(parts[2]) != "" {
		postIns, err = parseSimpleStmt(f, regs, strings.TrimSpace(parts[2]))
		if err != nil {
			return ir.Stmt{}, 0, err
		}
	}
	restRaw := s[closeP+1:]
	// brace body or single statement until ;
	braceAt := strings.Index(restRaw, "{")
	semiAt := strings.Index(restRaw, ";")
	var body []ir.Stmt
	var total int
	if braceAt >= 0 && (semiAt < 0 || braceAt < semiAt) {
		bodySrc, err := extractBlock(restRaw, braceAt)
		if err != nil {
			return ir.Stmt{}, 0, err
		}
		body, err = parseBlock(f, regs, bodySrc)
		if err != nil {
			return ir.Stmt{}, 0, err
		}
		total = closeP + 1 + braceAt + 1 + len(bodySrc) + 1
	} else if semiAt >= 0 {
		// for (...) stmt;
		one := strings.TrimSpace(restRaw[:semiAt])
		ins, err := parseSimpleStmt(f, regs, one)
		if err != nil {
			return ir.Stmt{}, 0, err
		}
		for _, in := range ins {
			body = append(body, ir.Stmt{Kind: ir.SKInstr, Ins: in})
		}
		total = closeP + 1 + semiAt + 1
	} else {
		return ir.Stmt{}, 0, &Error{Code: ErrParse, Message: "for body brace"}
	}
	st := ir.Stmt{
		Kind:        ir.SKFor,
		ForInit:     initIns,
		ForCondPrep: condPrep,
		CondLeft:    cl,
		CondOp:      cop,
		CondRight:   cr,
		ForPost:     postIns,
		ForBody:     body,
	}
	return st, total, nil
}

func parseCond(f *ir.Func, regs map[string]regInfo, cond string) (ir.Value, string, ir.Value, error) {
	cond = stripOuterParens(cond)
	cond = strings.TrimSpace(cond)
	if cond == "" {
		zero := f.Alloc()
		f.Body = append(f.Body, ir.Instr{Op: ir.OpConst, Dst: zero, Imm: 1})
		return zero, "truthy", ir.NoVal, nil
	}
	// ptr cmp ptr (<=, >=, !=, ==, <, >) -> compare mutable offSlots
	for _, op := range []string{"<=", ">=", "!=", "==", "<", ">"} {
		if idx, _ := findOp(cond, []string{op}); idx > 0 {
			ls, rs := strings.TrimSpace(cond[:idx]), strings.TrimSpace(cond[idx+len(op):])
			lOff, lok := parsePtrOffsetExpr(f, regs, ls)
			rOff, rok := parsePtrOffsetExpr(f, regs, rs)
			if lok && rok {
				return lOff, op, rOff, nil
			}
		}
	}
	// --name or name--  →  name = name - 1; truthy name (post: value before dec for name--)
	if strings.HasPrefix(cond, "--") {
		name := strings.TrimSpace(cond[2:])
		if ri, ok := regs[name]; ok {
			one := f.Alloc()
			tmp := f.Alloc()
			f.Body = append(f.Body,
				ir.Instr{Op: ir.OpConst, Dst: one, Imm: 1},
				ir.Instr{Op: ir.OpSub, Dst: tmp, Args: []ir.Value{ri.v, one}},
				ir.Instr{Op: ir.OpMov, Dst: ri.v, Args: []ir.Value{tmp}},
			)
			return ri.v, "truthy", ir.NoVal, nil
		}
	}
	if strings.HasSuffix(cond, "--") {
		name := strings.TrimSpace(cond[:len(cond)-2])
		if ri, ok := regs[name]; ok {
			// post-dec: use old value for cond, then dec
			oldv := f.Alloc()
			one := f.Alloc()
			tmp := f.Alloc()
			f.Body = append(f.Body,
				ir.Instr{Op: ir.OpMov, Dst: oldv, Args: []ir.Value{ri.v}},
				ir.Instr{Op: ir.OpConst, Dst: one, Imm: 1},
				ir.Instr{Op: ir.OpSub, Dst: tmp, Args: []ir.Value{ri.v, one}},
				ir.Instr{Op: ir.OpMov, Dst: ri.v, Args: []ir.Value{tmp}},
			)
		}
	}
	if strings.Contains(cond, "||") || strings.Contains(cond, "&&") {
		l, li, err := pe(f, regs, cond)
		if err != nil {
			return 0, "", 0, err
		}
		f.Body = li
		return l, "truthy", ir.NoVal, nil
	}
	for _, op := range []string{"!=", "==", "<=", ">=", "<", ">"} {
		if idx, _ := findOp(cond, []string{op}); idx > 0 {
			l, li, err := pe(f, regs, strings.TrimSpace(cond[:idx]))
			if err != nil {
				return 0, "", 0, err
			}
			r, ri, err := pe(f, regs, strings.TrimSpace(cond[idx+len(op):]))
			if err != nil {
				return 0, "", 0, err
			}
			// stash cond setup on f.Body for for-init merge
			f.Body = append(li, ri...)
			return l, op, r, nil
		}
	}
	l, li, err := pe(f, regs, cond)
	if err != nil {
		return 0, "", 0, err
	}
	f.Body = li
	return l, "truthy", ir.NoVal, nil
}

func parsePtrOffsetExpr(f *ir.Func, regs map[string]regInfo, s string) (ir.Value, bool) {
	s = strings.TrimSpace(s)
	if identOnly(s) {
		if ri, ok := regs[s]; ok && ri.ptr {
			return ensureOffSlot(f, regs, s), true
		}
		return ir.NoVal, false
	}
	if plus := strings.Index(s, "+"); plus > 0 {
		base := strings.TrimSpace(s[:plus])
		add := strings.TrimSpace(s[plus+1:])
		if ri, ok := regs[base]; ok && ri.ptr {
			boff := ensureOffSlot(f, regs, base)
			if n, err := strconv.ParseInt(add, 0, 64); err == nil {
				c := f.Alloc()
				tmp := f.Alloc()
				f.Body = append(f.Body,
					ir.Instr{Op: ir.OpConst, Dst: c, Imm: n, Elem: ir.TypUint64},
					ir.Instr{Op: ir.OpAdd, Dst: tmp, Args: []ir.Value{boff, c}, Elem: ir.TypUint64},
				)
				return tmp, true
			}
		}
	}
	if minus := strings.Index(s, "-"); minus > 0 {
		base := strings.TrimSpace(s[:minus])
		sub := strings.TrimSpace(s[minus+1:])
		if ri, ok := regs[base]; ok && ri.ptr {
			boff := ensureOffSlot(f, regs, base)
			if n, err := strconv.ParseInt(sub, 0, 64); err == nil {
				c := f.Alloc()
				tmp := f.Alloc()
				f.Body = append(f.Body,
					ir.Instr{Op: ir.OpConst, Dst: c, Imm: n, Elem: ir.TypUint64},
					ir.Instr{Op: ir.OpSub, Dst: tmp, Args: []ir.Value{boff, c}, Elem: ir.TypUint64},
				)
				return tmp, true
			}
		}
	}
	return ir.NoVal, false
}

func parseSwitch(f *ir.Func, regs map[string]regInfo, s string) (ir.Stmt, int, error) {
	i := strings.Index(s, "(")
	closeP, err := matchParen(s, i)
	if err != nil {
		return ir.Stmt{}, 0, err
	}
	onExpr := strings.TrimSpace(s[i+1 : closeP])
	on, onIns, err := pe(f, regs, onExpr)
	if err != nil {
		return ir.Stmt{}, 0, err
	}
	restRaw := s[closeP+1:]
	braceAt := strings.Index(restRaw, "{")
	if braceAt < 0 {
		return ir.Stmt{}, 0, &Error{Code: ErrParse, Message: "switch brace"}
	}
	bodySrc, err := extractBlock(restRaw, braceAt)
	if err != nil {
		return ir.Stmt{}, 0, err
	}
	cases, err := parseSwitchBody(f, regs, bodySrc)
	if err != nil {
		return ir.Stmt{}, 0, err
	}
	total := closeP + 1 + braceAt + 1 + len(bodySrc) + 1
	return ir.Stmt{Kind: ir.SKSwitch, SwitchPrep: onIns, SwitchOn: on, SwitchCases: cases}, total, nil
}

func parseSwitchBody(f *ir.Func, regs map[string]regInfo, body string) ([]ir.SwitchCase, error) {
	// tokenize by case/default
	re := regexp.MustCompile(`(?m)\b(case|default)\b`)
	idxs := re.FindAllStringIndex(body, -1)
	if len(idxs) == 0 {
		return nil, &Error{Code: ErrParse, Message: "empty switch"}
	}
	var cases []ir.SwitchCase
	for ci, loc := range idxs {
		start := loc[0]
		end := len(body)
		if ci+1 < len(idxs) {
			end = idxs[ci+1][0]
		}
		chunk := strings.TrimSpace(body[start:end])
		var labels []int64
		var bodyPart string
		if strings.HasPrefix(chunk, "default") {
			// default: stmts
			rest := strings.TrimSpace(chunk[len("default"):])
			if !strings.HasPrefix(rest, ":") {
				return nil, &Error{Code: ErrParse, Message: "default:"}
			}
			bodyPart = strings.TrimSpace(rest[1:])
		} else {
			// case N: or case N: case M:
			rest := chunk
			for {
				rest = strings.TrimSpace(rest)
				if !strings.HasPrefix(rest, "case") {
					break
				}
				rest = strings.TrimSpace(rest[4:])
				colon := indexCaseColon(rest)
				if colon < 0 {
					return nil, &Error{Code: ErrParse, Message: "case:"}
				}
				lab := strings.TrimSpace(rest[:colon])
				n, err := parseIntLit(lab)
				if err != nil {
					return nil, &Error{Code: ErrParse, Message: "case label: " + lab}
				}
				labels = append(labels, n)
				rest = rest[colon+1:]
				// another case immediately?
				if strings.HasPrefix(strings.TrimSpace(rest), "case") {
					continue
				}
				bodyPart = strings.TrimSpace(rest)
				break
			}
		}
		hadBreak := regexp.MustCompile(`(?s)\bbreak\s*;?\s*$`).MatchString(bodyPart)
		bodyPart = regexp.MustCompile(`\bbreak\s*;?\s*$`).ReplaceAllString(bodyPart, "")
		st, err := parseBlock(f, regs, bodyPart)
		if err != nil {
			return nil, err
		}
		fall := !hadBreak && !stmtsEndReturn(st)
		cases = append(cases, ir.SwitchCase{Labels: labels, Body: st, Fall: fall})
	}
	return cases, nil
}

func stmtsEndReturn(st []ir.Stmt) bool {
	if len(st) == 0 {
		return false
	}
	last := st[len(st)-1]
	if last.Kind == ir.SKInstr && last.Ins.Op == ir.OpReturn {
		return true
	}
	return false
}

func matchParen(s string, open int) (int, error) {
	depth := 0
	inChar := false
	inString := false
	for i := open; i < len(s); i++ {
		if s[i] == '\\' && (inChar || inString) {
			i++
			continue
		}
		if s[i] == '\'' && !inString {
			inChar = !inChar
			continue
		}
		if s[i] == '"' && !inChar {
			inString = !inString
			continue
		}
		if inChar || inString {
			continue
		}
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
	return 0, &Error{Code: ErrParse, Message: "unclosed ("}
}

func splitSemi(s string) []string {
	var out []string
	var cur strings.Builder
	depth := 0
	for _, r := range s {
		switch r {
		case '(':
			depth++
			cur.WriteRune(r)
		case ')':
			depth--
			cur.WriteRune(r)
		case ';':
			if depth == 0 {
				out = append(out, cur.String())
				cur.Reset()
			} else {
				cur.WriteRune(r)
			}
		default:
			cur.WriteRune(r)
		}
	}
	out = append(out, cur.String())
	return out
}

func parseSimpleStmt(f *ir.Func, regs map[string]regInfo, st string) ([]ir.Instr, error) {
	st = strings.TrimSpace(st)
	if st == "" {
		return nil, nil
	}

	// Comma-separated statement sequence: s1 += ptr[0], s2 += s1; (not multi-declarations: int i, r;)
	if !isTypeToken(firstWord(st)) && strings.Contains(st, ",") {
		parts := splitCSV(st)
		if len(parts) > 1 {
			var all []ir.Instr
			for _, part := range parts {
				ins, err := parseSimpleStmt(f, regs, strings.TrimSpace(part))
				if err != nil {
					return nil, err
				}
				all = append(all, ins...)
			}
			return all, nil
		}
	}

	for strings.HasPrefix(st, "(void)") || strings.HasPrefix(st, "((void)") {
		if strings.HasPrefix(st, "(void)") {
			st = strings.TrimSpace(strings.TrimPrefix(st, "(void)"))
		} else {
			st = strings.TrimSpace(strings.TrimPrefix(st, "((void)"))
		}
		if strings.HasPrefix(st, "(") && strings.HasSuffix(st, ")") {
			st = strings.TrimSpace(st[1 : len(st)-1])
		}
	}
	if st == "" {
		return nil, nil
	}

	// *p++ = … handled later; bare i++ / ++i / a[i]++
	if !strings.Contains(st, "=") {
		var name, op string
		switch {
		case strings.HasSuffix(st, "++") || strings.HasSuffix(st, "--"):
			op = st[len(st)-2:]
			name = strings.TrimSpace(st[:len(st)-2])
		case strings.HasPrefix(st, "++") || strings.HasPrefix(st, "--"):
			op = st[:2]
			name = strings.TrimSpace(st[2:])
		}
		if name != "" {
			// a[i]++
			if strings.Contains(name, "[") || strings.Contains(name, "->") {
				_, ins, err := pe(f, regs, name+op)
				return ins, err
			}
			name = strings.TrimPrefix(stripOuterParens(name), "*")
			ri, ok := regs[name]
			if !ok {
				return nil, &Error{Code: ErrParse, Message: "inc: " + name}
			}
			bin := ir.OpAdd
			if op == "--" {
				bin = ir.OpSub
			}
			// double pointer cursor: advance pointed-to slice (*input)++
			if strings.HasPrefix(string(ri.typ), "*[]") || strings.HasPrefix(st, "(*") {
				isDbl := strings.HasPrefix(string(ri.typ), "*[]")
				if !isDbl {
					for _, p := range f.Params {
						if p.Name == name && strings.HasPrefix(string(p.Type), "*[]") {
							isDbl = true
							break
						}
					}
				}
				if isDbl {
					return []ir.Instr{{Op: ir.OpMov, Dst: ri.v, Sym: "double_ptr_adv1"}}, nil
				}
			}
			// byte-buffer cursor: bump offSlot only (poly1305 message++, etc.)
			if ri.ptr && ri.structName == "" && (ri.offSlotSet || ri.typ == "" || ri.typ == ir.TypUint8 || ri.typ == ir.TypInt) {
				_ = ensureOffSlot(f, regs, name)
				ri = regs[name]
				one := f.Alloc()
				tmp := f.Alloc()
				return []ir.Instr{
					{Op: ir.OpConst, Dst: one, Imm: 1, Elem: ir.TypUint64},
					{Op: bin, Dst: tmp, Args: []ir.Value{ri.offSlot, one}, Elem: ir.TypUint64, Sym: "offslot"},
					{Op: ir.OpMov, Dst: ri.offSlot, Args: []ir.Value{tmp}, Elem: ir.TypUint64, Sym: "offslot"},
				}, nil
			}
			one := f.Alloc()
			tmp := f.Alloc()
			return []ir.Instr{
				{Op: ir.OpConst, Dst: one, Imm: 1},
				{Op: bin, Dst: tmp, Args: []ir.Value{ri.v, one}},
				{Op: ir.OpMov, Dst: ri.v, Args: []ir.Value{tmp}},
			}, nil
		}
	}

	if st == "break" {
		return []ir.Instr{{Op: ir.OpBreak}}, nil
	}
	if st == "continue" {
		return []ir.Instr{{Op: ir.OpContinue}}, nil
	}
	if strings.HasPrefix(st, "goto ") {
		lbl := strings.TrimSpace(strings.TrimPrefix(st, "goto "))
		return []ir.Instr{{Op: ir.OpGoto, Sym: lbl}}, nil
	}
	if strings.HasSuffix(st, ":") && !strings.HasPrefix(st, "case ") && st != "default:" {
		lbl := strings.TrimSpace(strings.TrimSuffix(st, ":"))
		return []ir.Instr{{Op: ir.OpLabel, Sym: lbl}}, nil
	}

	if strings.HasPrefix(st, "return") {
		rest := strings.TrimSpace(strings.TrimPrefix(st, "return"))
		if rest == "" {
			return []ir.Instr{{Op: ir.OpReturn}}, nil
		}
		// return foo(...); void function returning void call
		if f.Result == ir.TypVoid {
			if i := strings.IndexByte(rest, '('); i > 0 && strings.HasSuffix(rest, ")") && identOnly(strings.TrimSpace(rest[:i])) {
				_, ins, err := pe(f, regs, rest)
				if err != nil {
					return nil, err
				}
				if n := len(ins); n > 0 && ins[n-1].Op == ir.OpCall {
					ins[n-1].Dst = ir.NoVal
					ins = append(ins, ir.Instr{Op: ir.OpReturn})
					return ins, nil
				}
			}
		}
		v, ins, err := pe(f, regs, rest)
		if err != nil {
			return nil, err
		}
		if pm, ok := ptrMeta[v]; ok && pm.hasBase && pm.structName != "" && pm.base != ir.NoVal {
			dst := f.Alloc()
			ins = append(ins, ir.Instr{Op: ir.OpAdd, Dst: dst, Args: []ir.Value{pm.v, pm.base}, Sym: "struct_ptr_add", Elem: pm.typ})
			v = dst
		}
		ins = append(ins, ir.Instr{Op: ir.OpReturn, Args: []ir.Value{v}})
		return ins, nil
	}

	// typedef array alias: fe x1;  (typedef i32 fe[10])
	if td, ok := typedefEnv[firstWord(st)]; ok && td.IsArray {
		rest := strings.TrimSpace(st[len(firstWord(st)):])
		// multi: fe a, b, c;
		if strings.Contains(rest, ",") && !strings.Contains(rest, "(") {
			var all []ir.Instr
			for _, part := range splitCSV(rest) {
				part = strings.TrimSpace(part)
				nm := part
				if i := strings.Index(part, "="); i >= 0 {
					nm = strings.TrimSpace(part[:i])
				}
				if !identOnly(nm) {
					return nil, &Error{Code: ErrParse, Message: "typedef-arr multi: " + part}
				}
				r := f.Alloc()
				regs[nm] = regInfo{v: r, typ: td.BaseType, ptr: true, scale: 1, elemIndex: true, localArr: td.ArrayLen}
				notePtr(r, td.BaseType, 1, ir.NoVal)
				pm := ptrMeta[r]
				pm.elemIndex = true
				pm.localArr = td.ArrayLen
				ptrMeta[r] = pm
				all = append(all, ir.Instr{Op: ir.OpAlloca, Dst: r, Imm: int64(td.ArrayLen), Elem: td.BaseType})
			}
			return all, nil
		}
		nm := rest
		if i := strings.Index(rest, "="); i >= 0 {
			nm = strings.TrimSpace(rest[:i])
		}
		if identOnly(nm) {
			r := f.Alloc()
			regs[nm] = regInfo{v: r, typ: td.BaseType, ptr: true, scale: 1, elemIndex: true, localArr: td.ArrayLen}
			notePtr(r, td.BaseType, 1, ir.NoVal)
			pm := ptrMeta[r]
			pm.elemIndex = true
			pm.localArr = td.ArrayLen
			ptrMeta[r] = pm
			return []ir.Instr{{Op: ir.OpAlloca, Dst: r, Imm: int64(td.ArrayLen), Elem: td.BaseType}}, nil
		}
	}

	// local struct: crypto_poly1305_ctx ctx; or const utf8proc_property_t *p = ...;
	if stc := findStruct(firstWord(st), structEnv); stc != nil {
		rest := strings.TrimSpace(st[len(firstWord(st)):])
		isPtr := strings.HasPrefix(rest, "*")
		rest = strings.TrimPrefix(rest, "*")
		rest = strings.TrimSpace(rest)
		// name only or name = ...
		nm := rest
		if i := strings.Index(rest, "="); i >= 0 {
			nm = strings.TrimSpace(rest[:i])
		}
		if identOnly(nm) {
			r := f.Alloc()
			stTyp := ir.TypeName(stc.Name)
			if isPtr {
				stTyp = ir.TypeName("*" + stc.Name)
			}
			regs[nm] = regInfo{v: r, typ: stTyp, ptr: isPtr, structName: stc.Name}
			if isPtr {
				notePtr(r, stTyp, 1, ir.NoVal)
				pm := ptrMeta[r]
				pm.structName = stc.Name
				ptrMeta[r] = pm
			}
			alloc := ir.Instr{Op: ir.OpAlloca, Dst: r, Imm: 1, Elem: stTyp, Sym: "struct:" + stc.Name}
			if i := strings.Index(rest, "="); i >= 0 {
				rhs := strings.TrimSpace(rest[i+1:])
				v, ins, err := pe(f, regs, rhs)
				if err != nil {
					return nil, err
				}
				ins = append([]ir.Instr{alloc}, ins...)
				ins = append(ins, ir.Instr{Op: ir.OpMov, Dst: r, Args: []ir.Value{v}, Sym: "ptr_alias", Elem: stTyp})
				return ins, nil
			}
			return []ir.Instr{alloc}, nil
		}
	}

	stClean := normalizeDeclStmt(st)
	if identOnly(stClean) || regexp.MustCompile(`^\s*\(\s*void\s*\)\s*[A-Za-z_][A-Za-z0-9_]*\s*$`).MatchString(stClean) {
		return nil, nil
	}
	if _, err := parseIntLit(stClean); err == nil {
		return nil, nil
	}

	// multi decl: TYPE a, b, c;  or TYPE a=e0, b=e1;
	if isTypeToken(firstWord(stClean)) && strings.Contains(stClean, ",") {
		fw := firstWord(stClean)
		rest := strings.TrimSpace(stClean[len(fw):])
		parts := splitCSV(rest)
		if len(parts) > 1 {
			var all []ir.Instr
			for _, part := range parts {
				part = strings.TrimSpace(part)
				sub := fw + " " + part
				ins, err := parseSimpleStmt(f, regs, sub)
				if err != nil {
					return nil, err
				}
				all = append(all, ins...)
			}
			return all, nil
		}
	}
	for {
		next := regexp.MustCompile(`^\s*(?:static|const|volatile|restrict|struct|inline|__inline|__inline__)\s+`).ReplaceAllString(stClean, "")
		if next == stClean {
			break
		}
		stClean = next
	}
	// decl: TYPE name[N] = { ... }; or TYPE name[R][C] = { ... }; or TYPE name[N];
	if isTypeToken(firstWord(stClean)) || isStructType(firstWord(stClean), structEnv) {
		initCSV := ""
		declHead := stClean
		if open := strings.Index(stClean, "{"); open > 0 && strings.HasSuffix(strings.TrimSpace(stClean), "}") {
			if closeBrace, err := matchBraceFrom(stClean, open); err == nil && closeBrace == len(strings.TrimSpace(stClean))-1 {
				initBody := stClean[open+1 : closeBrace]
				initCSV = flattenBraceInit(initBody)
				declHead = strings.TrimSpace(stClean[:open])
				if strings.HasSuffix(declHead, "=") {
					declHead = strings.TrimSpace(declHead[:len(declHead)-1])
				}
			}
		}

		reArr2D := regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s+([A-Za-z_][A-Za-z0-9_]*)\s*\[\s*(\d*)\s*\]\s*\[\s*(\d+)\s*\]$`)
		if m := reArr2D.FindStringSubmatch(declHead); m != nil {
			typ := mapType(m[1])
			sname := ""
			if isStructType(m[1], structEnv) {
				typ = ir.TypeName(m[1])
				sname = m[1]
			}
			r2, _ := strconv.Atoi(m[4])
			r1 := 0
			total := 0
			if m[3] != "" {
				r1, _ = strconv.Atoi(m[3])
				total = r1 * r2
			} else if initCSV != "" {
				total = len(splitCSV(initCSV))
				r1 = total / r2
			}
			if total <= 0 {
				total = r2
			}
			r := f.Alloc()
			sym := sname
			if initCSV != "" {
				sym = "init:" + m[2] + ":" + initCSV
			}
			regs[m[2]] = regInfo{v: r, typ: typ, ptr: true, scale: 1, elemIndex: true, localArr: total, cols: r2, structName: sname}
			notePtr(r, typ, 1, ir.NoVal)
			pm := ptrMeta[r]
			pm.elemIndex = true
			pm.localArr = total
			pm.structName = sname
			ptrMeta[r] = pm
			return []ir.Instr{{Op: ir.OpAlloca, Dst: r, Imm: int64(total), Elem: typ, Sym: sym}}, nil
		}

		if m := regexp.MustCompile(`^fe\s+([A-Za-z0-9_,\s]+)$`).FindStringSubmatch(declHead); m != nil {
			var instrs []ir.Instr
			for _, item := range strings.Split(m[1], ",") {
				nm := strings.TrimSpace(item)
				if nm == "" {
					continue
				}
				r := f.Alloc()
				regs[nm] = regInfo{v: r, typ: ir.TypInt32, ptr: true, scale: 1, elemIndex: true, localArr: 10}
				notePtr(r, ir.TypInt32, 1, ir.NoVal)
				pm := ptrMeta[r]
				pm.elemIndex = true
				pm.localArr = 10
				ptrMeta[r] = pm
				instrs = append(instrs, ir.Instr{Op: ir.OpAlloca, Dst: r, Imm: 10, Elem: ir.TypInt32})
			}
			return instrs, nil
		}

		reArr := regexp.MustCompile(`(?s)^([A-Za-z_][A-Za-z0-9_]*)\s+([A-Za-z_][A-Za-z0-9_]*)\s*\[\s*([^]]*)\s*\]$`)
		if m := reArr.FindStringSubmatch(declHead); m != nil && (m[3] != "" || initCSV != "") {
			typ := mapType(m[1])
			sname := ""
			if isStructType(m[1], structEnv) {
				typ = ir.TypeName(m[1])
				sname = m[1]
			}
			n := 0
			if m[3] != "" {
				if num, err := parseIntLit(m[3]); err == nil && num > 0 {
					n = int(num)
				} else {
					n, _ = strconv.Atoi(m[3])
				}
			} else if initCSV != "" {
				n = len(strings.Split(initCSV, ","))
			}
			r := f.Alloc()
			sym := sname
			if initCSV != "" {
				sym = "init:" + m[2] + ":" + initCSV
			}
			regs[m[2]] = regInfo{v: r, typ: typ, ptr: true, scale: 1, elemIndex: true, localArr: n, structName: sname}
			notePtr(r, typ, 1, ir.NoVal)
			pm := ptrMeta[r]
			pm.elemIndex = true
			pm.localArr = n
			pm.structName = sname
			ptrMeta[r] = pm
			return []ir.Instr{{Op: ir.OpAlloca, Dst: r, Imm: int64(n), Elem: typ, Sym: sym}}, nil
		}
		reStructInit := regexp.MustCompile(`^(?:struct\s+)?([A-Za-z_][A-Za-z0-9_]*)\s+([A-Za-z_][A-Za-z0-9_]*)$`)
		if m := reStructInit.FindStringSubmatch(declHead); m != nil && isStructType(m[1], structEnv) {
			sname := m[1]
			r := f.Alloc()
			sym := "struct:" + sname
			if initCSV != "" {
				sym = "struct_init:" + sname + ":" + m[2] + ":" + initCSV
			}
			regs[m[2]] = regInfo{v: r, typ: ir.TypeName(sname), ptr: true, scale: 1, elemIndex: false, structName: sname}
			if f.LocalNames == nil {
				f.LocalNames = map[ir.Value]string{}
			}
			f.LocalNames[r] = m[2]
			notePtr(r, ir.TypeName(sname), 1, ir.NoVal)
			pm := ptrMeta[r]
			pm.structName = sname
			ptrMeta[r] = pm
			return []ir.Instr{{Op: ir.OpAlloca, Dst: r, Imm: 1, Elem: ir.TypeName(sname), Sym: sym}}, nil
		}

		rePtrArr := regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*\(\s*\*\s*([A-Za-z_][A-Za-z0-9_]*)\s*\)\s*\[\s*(\d+)\s*\]$`)
		if m := rePtrArr.FindStringSubmatch(declHead); m != nil {
			typName := ir.TypeName(fmt.Sprintf("[][%s]%s", m[3], string(mapType(m[1]))))
			r := f.Alloc()
			regs[m[2]] = regInfo{v: r, scale: 1, typ: typName}
			return []ir.Instr{{Op: ir.OpMov, Dst: r, Sym: "decl_uninit", Elem: typName}}, nil
		}

		if initCSV != "" {
			return nil, &Error{Code: ErrParse, Message: "unsupported aggregate initializer: " + stClean}
		}

		stClean = regexp.MustCompile(`\b(?:const|volatile|restrict)\b`).ReplaceAllString(stClean, "")
		stClean = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(stClean), "struct "))
		stClean = regexp.MustCompile(`\s*\*\s*`).ReplaceAllString(stClean, " * ")
		stClean = strings.TrimSpace(stClean)
		// TYPE * name = expr  OR TYPE name = expr OR TYPE name
		re := regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*(\*)?\s*([A-Za-z_][A-Za-z0-9_]*)\s*(?:=\s*(.+))?$`)
		m := re.FindStringSubmatch(stClean)
		if m == nil {
			return nil, &Error{Code: ErrParse, Message: "decl: " + st}
		}
		typ := mapType(m[1])
		stc := findStruct(m[1], structEnv)
		sname := ""
		if stc != nil {
			typ = ir.TypeName(stc.Name)
			sname = stc.Name
		}
		ptr := m[2] == "*"
		r := f.Alloc()
		scale := 1
		if ptr {
			switch typ {
			case ir.TypUint32:
				scale = 4
			case ir.TypUint64:
				scale = 8
			}
		}
		regs[m[3]] = regInfo{v: r, typ: typ, ptr: ptr, scale: scale, structName: sname}
		if f.LocalNames == nil {
			f.LocalNames = map[ir.Value]string{}
		}
		f.LocalNames[r] = m[3]
		if ptr {
			notePtr(r, typ, scale, ir.NoVal)
			if sname != "" {
				pm := ptrMeta[r]
				pm.structName = sname
				ptrMeta[r] = pm
			}
		}
		if m[4] == "" {
			return []ir.Instr{{Op: ir.OpMov, Dst: r, Sym: "decl_uninit", Elem: typ}}, nil
		}
		v, ins, err := pe(f, regs, m[4])
		if err != nil {
			return nil, err
		}
		// Carry the *declared* type, not the initializer's. `u32 u = x[3]` widens
		// the byte; typing u after its initializer made ld32 accumulate in uint8
		// and truncate every shift.
		decl := ir.Instr{Op: ir.OpMov, Dst: r, Args: []ir.Value{v}}
		if !ptr {
			decl.Sym = "decl"
			decl.Elem = typ
		} else {
			decl.Sym = "ptr_alias"
			decl.Elem = typ
		}
		ins = append(ins, decl)
		// pointer init: track meta for local variable r
		if ptr {
			sc := scale
			if pm, ok := ptrMeta[v]; ok && m[4] != "NULL" && m[4] != "null" && m[4] != "0" && m[4] != "nil" {
				if pm.elemIndex {
					sc = 1
				}
				rootV := pm.v
				if rootV == ir.NoVal {
					rootV = r
				}
				regs[m[3]] = regInfo{v: r, typ: typ, ptr: true, scale: sc, base: pm.base, hasBase: pm.hasBase, elemIndex: pm.elemIndex, localArr: pm.localArr}
				ptrMeta[r] = regInfo{v: rootV, typ: typ, ptr: true, scale: sc, base: pm.base, hasBase: pm.hasBase, elemIndex: pm.elemIndex, localArr: pm.localArr}
			} else {
				regs[m[3]] = regInfo{v: r, typ: typ, ptr: true, scale: sc, elemIndex: true}
				ptrMeta[r] = regs[m[3]]
			}
		}
		return ins, nil
	}

	// *dest++ = *src++;  or *dest++ = expr;
	if strings.Contains(st, "++") && strings.Contains(st, "=") && strings.HasPrefix(strings.TrimSpace(st), "*") {
		ins, err := parsePostIncAssign(f, regs, st)
		if err == nil {
			return ins, nil
		}
		// fallthrough to generic if not matched
	}
	// bare call statement: foo(a,b); or obj->method(a,b);
	if i := strings.IndexByte(st, '('); i > 0 && strings.HasSuffix(st, ")") && (identOnly(strings.TrimSpace(st[:i])) || isFieldChain(strings.TrimSpace(st[:i]))) {
		_, ins, err := pe(f, regs, st)
		if err != nil {
			return nil, err
		}
		if n := len(ins); n > 0 && ins[n-1].Op == ir.OpCall {
			ins[n-1].Dst = ir.NoVal // void call statement
		}
		return ins, nil
	}
	// store: *p = e  or p[i] = e  or p->f = e  or p->f[i] = e
	if i := strings.Index(st, "="); i > 0 {
		// check compound first
		for _, op := range []string{"<<=", ">>=", "+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "="} {
			if j := strings.Index(st, op); j > 0 {
				lhs := stripOuterParens(strings.TrimSpace(st[:j]))
				rhs := strings.TrimSpace(st[j+len(op):])
				if op == "=" && !strings.HasPrefix(rhs, "'") && !strings.HasPrefix(rhs, "\"") && strings.Contains(rhs, "=") && !strings.Contains(rhs, "==") && !strings.Contains(rhs, "<=") && !strings.Contains(rhs, ">=") && !strings.Contains(rhs, "!=") && !strings.Contains(rhs, "=>") {
					eq := strings.Index(rhs, "=")
					firstRhs := strings.TrimSpace(rhs[:eq])
					s1, err1 := parseSimpleStmt(f, regs, rhs)
					if err1 != nil {
						return nil, err1
					}
					s2, err2 := parseSimpleStmt(f, regs, lhs+" = "+firstRhs)
					if err2 != nil {
						return nil, err2
					}
					return append(s1, s2...), nil
				}
				rv, ins, err := pe(f, regs, rhs)
				if err != nil {
					return nil, err
				}
				// base.field = rhs (e.g. p->arr[i].field = rhs or obj.field = rhs)
				if dot := indexByteAtDepth(lhs, '.', 0); dot > 0 && (dot > lastIndexArrow(lhs) || !strings.Contains(lhs, "->")) {
					base := strings.TrimSpace(lhs[:dot])
					rest := strings.TrimSpace(lhs[dot+1:])
					fname := rest
					idxe := ""
					if b := strings.IndexByte(rest, '['); b > 0 && strings.HasSuffix(rest, "]") {
						fname = strings.TrimSpace(rest[:b])
						idxe = rest[b+1 : len(rest)-1]
					}
					bv, bins, err := pe(f, regs, base)
					if err != nil {
						return nil, err
					}
					ins = append(ins, bins...)
					sname := ""
					if pm, ok := ptrMeta[bv]; ok {
						sname = pm.structName
					}
					if sname == "" {
						if ri, ok := regs[base]; ok {
							sname = ri.structName
						}
					}
					if sname == "" {
						for _, p := range f.Params {
							if p.Name == base {
								sname = p.StructName
								break
							}
						}
					}
					if sname == "" {
						for _, st := range structEnv {
							if sf := fieldOf(&st, fname); sf != nil {
								sname = st.Name
								break
							}
						}
					}
					elem := ir.TypUint8
					var alen int64
					if st := findStruct(sname, structEnv); st != nil {
						if sf := fieldOf(st, fname); sf != nil {
							elem = sf.Type
							alen = int64(sf.ArrayLen)
						}
					}
					if idxe != "" {
						iv, iins, err := pe(f, regs, idxe)
						if err != nil {
							return nil, err
						}
						ins = append(ins, iins...)
						fld := f.Alloc()
						ins = append(ins, ir.Instr{Op: ir.OpField, Dst: fld, Args: []ir.Value{bv}, Sym: fname, Elem: elem, Imm: alen})
						ins = append(ins, ir.Instr{Op: ir.OpStore, Args: []ir.Value{fld, iv, rv}, Elem: elem})
						return ins, nil
					}
					ins = append(ins, ir.Instr{Op: ir.OpFStore, Args: []ir.Value{bv, rv}, Sym: fname, Elem: elem})
					return ins, nil
				}
				// p->field or p->field[i] or (p->field)[i]
				if arrow := lastIndexArrow(lhs); arrow > 0 && !strings.HasPrefix(strings.TrimSpace(lhs), "(") {
					base := strings.TrimSpace(lhs[:arrow])
					rest := strings.TrimSpace(lhs[arrow+2:])
					fname := rest
					idxe := ""
					if b := strings.IndexByte(rest, '['); b > 0 && strings.HasSuffix(rest, "]") {
						fname = strings.TrimSpace(rest[:b])
						idxe = rest[b+1 : len(rest)-1]
					}
					bv, bins, err := pe(f, regs, base)
					if err != nil {
						return nil, err
					}
					ins = append(ins, bins...)
					if op != "=" {
						// compound: load field, binop, store
						lv, lins, err := pe(f, regs, lhs)
						if err != nil {
							return nil, err
						}
						ins = append(ins, lins...)
						bin := map[string]ir.Op{
							"+=": ir.OpAdd, "-=": ir.OpSub, "*=": ir.OpMul,
							"/=": ir.OpDiv, "%=": ir.OpMod,
							"&=": ir.OpAnd, "|=": ir.OpOr, "^=": ir.OpXor,
							"<<=": ir.OpShl, ">>=": ir.OpShr,
						}[op]
						tmp := f.Alloc()
						ins = append(ins, ir.Instr{Op: bin, Dst: tmp, Args: []ir.Value{lv, rv}})
						rv = tmp
					}
					if idxe != "" {
						iv, iins, err := pe(f, regs, idxe)
						if err != nil {
							return nil, err
						}
						ins = append(ins, iins...)
						fld := f.Alloc()
						elem := ir.TypUint8
						var alen int64
						sname := ""
						if pm, ok := ptrMeta[bv]; ok {
							sname = pm.structName
						}
						if sname == "" {
							if ri, ok := regs[base]; ok {
								sname = ri.structName
							}
						}
						if sname == "" {
							for _, p := range f.Params {
								if p.Name == base {
									sname = p.StructName
									break
								}
							}
						}
						if sname == "" {
							for _, st := range structEnv {
								if sf := fieldOf(&st, fname); sf != nil {
									sname = st.Name
									break
								}
							}
						}
						if st := findStruct(sname, structEnv); st != nil {
							if sf := fieldOf(st, fname); sf != nil {
								elem = sf.Type
								alen = int64(sf.ArrayLen)
							}
						}
						ins = append(ins, ir.Instr{Op: ir.OpField, Dst: fld, Args: []ir.Value{bv}, Sym: fname, Elem: elem, Imm: alen})
						ins = append(ins, ir.Instr{Op: ir.OpStore, Args: []ir.Value{fld, iv, rv}, Elem: elem})
					} else {
						sname := ""
						if pm, ok := ptrMeta[bv]; ok {
							sname = pm.structName
						}
						if sname == "" {
							if ri, ok := regs[base]; ok {
								sname = ri.structName
							}
						}
						if sname == "" {
							for _, p := range f.Params {
								if p.Name == base {
									sname = p.StructName
									break
								}
							}
						}
						if sname == "" {
							for _, st := range structEnv {
								if sf := fieldOf(&st, fname); sf != nil {
									sname = st.Name
									break
								}
							}
						}
						elem := ir.TypUint8
						if st := findStruct(sname, structEnv); st != nil {
							if sf := fieldOf(st, fname); sf != nil {
								elem = sf.Type
							}
						}
						ins = append(ins, ir.Instr{Op: ir.OpFStore, Args: []ir.Value{bv, rv}, Sym: fname, Elem: elem})
					}
					return ins, nil
				}
				if strings.Contains(lhs, "[") || strings.HasPrefix(strings.TrimSpace(lhs), "*") {
					if op == "=" {
						stins, err := parseStore(f, regs, lhs, rv)
						if err != nil {
							return nil, err
						}
						return append(ins, stins...), nil
					}
					// compound on indexed lhs: h[i] &= x  →  t=h[i]; t=t op x; h[i]=t
					lv, lins, err := pe(f, regs, lhs)
					if err != nil {
						return nil, err
					}
					ins = append(ins, lins...)
					bin := map[string]ir.Op{
						"+=": ir.OpAdd, "-=": ir.OpSub, "*=": ir.OpMul,
						"/=": ir.OpDiv, "%=": ir.OpMod,
						"&=": ir.OpAnd, "|=": ir.OpOr, "^=": ir.OpXor,
						"<<=": ir.OpShl, ">>=": ir.OpShr,
					}[op]
					tmp := f.Alloc()
					ins = append(ins, ir.Instr{Op: bin, Dst: tmp, Args: []ir.Value{lv, rv}})
					stins, err := parseStore(f, regs, lhs, tmp)
					if err != nil {
						return nil, err
					}
					return append(ins, stins...), nil
				}
				if !identOnly(lhs) {
					return nil, &Error{Code: ErrParse, Message: "lhs: " + lhs}
				}
				ri, ok := regs[lhs]
				if !ok {
					r := f.Alloc()
					regs[lhs] = regInfo{v: r, typ: ir.TypInt}
					ri = regs[lhs]
				}
				if op == "=" {
					ins = append(ins, ir.Instr{Op: ir.OpMov, Dst: ri.v, Args: []ir.Value{rv}})
					if pm, ok := ptrMeta[rv]; ok {
						if pm.hasBase {
							slot := ri.offSlot
							if !ri.offSlotSet {
								slot = f.Alloc()
								ins = append(ins, ir.Instr{Op: ir.OpMov, Dst: slot, Args: []ir.Value{pm.base}})
							}
							regs[lhs] = regInfo{v: ri.v, typ: pm.typ, ptr: true, scale: pm.scale, base: slot, hasBase: true, offSlot: slot, offSlotSet: true, elemIndex: pm.elemIndex, structName: pm.structName}
							ptrMeta[ri.v] = regs[lhs]
						} else {
							regs[lhs] = regInfo{v: ri.v, typ: pm.typ, ptr: true, scale: pm.scale, elemIndex: pm.elemIndex, structName: pm.structName}
							ptrMeta[ri.v] = regs[lhs]
						}
					}
					return ins, nil
				}
				if strings.HasPrefix(lhs, "*") && (op == "+=" || op == "=") {
					clean := strings.TrimSpace(strings.TrimPrefix(lhs, "*"))
					if riClean, ok := regs[clean]; ok && strings.HasPrefix(string(riClean.typ), "*[]") {
						return append(ins, ir.Instr{Op: ir.OpStore, Args: []ir.Value{riClean.v, rv}, Sym: "slice_ptr_advance"}), nil
					}
				}
				// pointer cursor advance: ptr += N / ptr -= N via offSlot (supports reverse).
				if ri.ptr && (op == "+=" || op == "-=") {
					bop := ir.OpAdd
					if op == "-=" {
						bop = ir.OpSub
					}
					if !ri.offSlotSet {
						slot := f.Alloc()
						// zero-init once at function prologue — never inside loop body
						if ri.hasBase {
							funcOffSlotPrologue = append(funcOffSlotPrologue, ir.Instr{Op: ir.OpMov, Dst: slot, Args: []ir.Value{ri.base}, Elem: ir.TypUint64, Sym: "offslot"})
						} else {
							z := f.Alloc()
							funcOffSlotPrologue = append(funcOffSlotPrologue,
								ir.Instr{Op: ir.OpConst, Dst: z, Imm: 0, Elem: ir.TypUint64},
								ir.Instr{Op: ir.OpMov, Dst: slot, Args: []ir.Value{z}, Elem: ir.TypUint64, Sym: "offslot"},
							)
						}
						ri.offSlot = slot
						ri.offSlotSet = true
						ri.hasBase = true
						ri.base = slot
					}
					// statement body: only the bump (safe inside loops)
					tmp := f.Alloc()
					ins = append(ins,
						ir.Instr{Op: bop, Dst: tmp, Args: []ir.Value{ri.offSlot, rv}, Elem: ir.TypUint64, Sym: "offslot"},
						ir.Instr{Op: ir.OpMov, Dst: ri.offSlot, Args: []ir.Value{tmp}, Elem: ir.TypUint64, Sym: "offslot"},
					)
					ri.base = ri.offSlot
					regs[lhs] = ri
					ptrMeta[ri.v] = ri
					ptrMeta[ri.offSlot] = ri
					return ins, nil
				}
				bin := map[string]ir.Op{
					"+=": ir.OpAdd, "-=": ir.OpSub, "*=": ir.OpMul,
					"/=": ir.OpDiv, "%=": ir.OpMod,
					"&=": ir.OpAnd, "|=": ir.OpOr, "^=": ir.OpXor,
					"<<=": ir.OpShl, ">>=": ir.OpShr,
				}[op]
				tmp := f.Alloc()
				ins = append(ins,
					ir.Instr{Op: bin, Dst: tmp, Args: []ir.Value{ri.v, rv}},
					ir.Instr{Op: ir.OpMov, Dst: ri.v, Args: []ir.Value{tmp}},
				)
				return ins, nil
			}
		}
	}
	if strings.HasSuffix(st, "++") || strings.HasSuffix(st, "--") || strings.HasPrefix(st, "++") || strings.HasPrefix(st, "--") {
		mark := len(f.Body)
		_, _, err := pe(f, regs, st)
		if err == nil {
			ins := append([]ir.Instr(nil), f.Body[mark:]...)
			f.Body = f.Body[:mark]
			return ins, nil
		}
	}
	return nil, &Error{Code: ErrParse, Message: "stmt: " + st}
}

func stripOuterParens(s string) string {
	s = strings.TrimSpace(s)
	for strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") && balanced(s[1:len(s)-1]) {
		s = strings.TrimSpace(s[1 : len(s)-1])
	}
	return s
}

// ensureOffSlot returns a stable vreg holding the current byte offset of a pointer name.
// First-time zero-init is hoisted to the function prologue (never inside a loop body).
func ensureOffSlot(f *ir.Func, regs map[string]regInfo, name string) ir.Value {
	ri := regs[name]
	if ri.offSlotSet {
		return ri.offSlot
	}
	slot := f.Alloc()
	if ri.hasBase {
		funcOffSlotPrologue = append(funcOffSlotPrologue, ir.Instr{Op: ir.OpMov, Dst: slot, Args: []ir.Value{ri.base}, Elem: ir.TypUint64, Sym: "offslot"})
	} else {
		z := f.Alloc()
		funcOffSlotPrologue = append(funcOffSlotPrologue,
			ir.Instr{Op: ir.OpConst, Dst: z, Imm: 0, Elem: ir.TypUint64},
			ir.Instr{Op: ir.OpMov, Dst: slot, Args: []ir.Value{z}, Elem: ir.TypUint64, Sym: "offslot"},
		)
	}
	ri.offSlot = slot
	ri.offSlotSet = true
	ri.hasBase = true
	ri.base = slot
	regs[name] = ri
	ptrMeta[ri.v] = ri
	ptrMeta[slot] = ri
	return slot
}

func parseStore(f *ir.Func, regs map[string]regInfo, lhs string, val ir.Value) ([]ir.Instr, error) {
	lhs = strings.TrimSpace(lhs)
	// arr[i].field or arr[i].field[j]
	if dot := indexByteAtDepth(lhs, '.', 0); dot > 0 && !strings.Contains(lhs, "->") {
		base := strings.TrimSpace(lhs[:dot])
		rest := strings.TrimSpace(lhs[dot+1:])
		fname := rest
		idxe := ""
		if b := strings.IndexByte(rest, '['); b >= 0 && strings.HasSuffix(rest, "]") {
			fname = strings.TrimSpace(rest[:b])
			idxe = rest[b+1 : len(rest)-1]
		}
		if identOnly(fname) && strings.Contains(base, "[") {
			addr, bins, err := pe(f, regs, "&"+base)
			if err != nil {
				return nil, err
			}
			sname := ""
			if pm, ok := ptrMeta[addr]; ok {
				sname = pm.structName
			}
			if sname == "" {
				root := base
				if lb := strings.IndexByte(base, '['); lb > 0 {
					root = strings.TrimSpace(base[:lb])
				}
				if ri, ok := regs[root]; ok {
					sname = ri.structName
				}
				for _, p := range f.Params {
					if p.Name == root && p.StructName != "" {
						sname = p.StructName
						break
					}
				}
			}
			elem := ir.TypUint8
			alen := int64(0)
			if st := findStruct(sname, structEnv); st != nil {
				if sf := fieldOf(st, fname); sf != nil {
					elem = sf.Type
					alen = int64(sf.ArrayLen)
				}
			}
			if idxe != "" {
				iv, iins, err := pe(f, regs, idxe)
				if err != nil {
					return nil, err
				}
				fld := f.Alloc()
				ins := append(bins, iins...)
				ins = append(ins, ir.Instr{Op: ir.OpField, Dst: fld, Args: []ir.Value{addr}, Sym: fname, Elem: elem, Imm: alen})
				ins = append(ins, ir.Instr{Op: ir.OpStore, Args: []ir.Value{fld, iv, val}, Elem: elem})
				return ins, nil
			}
			ins := append(bins, ir.Instr{Op: ir.OpFStore, Args: []ir.Value{addr, val}, Sym: fname, Elem: elem})
			return ins, nil
		}
	}
	// p->field or p->field[idx]
	if arrow := indexArrow(lhs); arrow > 0 {
		base := strings.TrimSpace(lhs[:arrow])
		rest := strings.TrimSpace(lhs[arrow+2:])
		fname := rest
		idxe := ""
		if b := strings.IndexByte(rest, '['); b >= 0 && strings.HasSuffix(rest, "]") {
			fname = strings.TrimSpace(rest[:b])
			idxe = rest[b+1 : len(rest)-1]
		}
		if identOnly(fname) {
			bv, bins, err := pe(f, regs, base)
			if err != nil {
				return nil, err
			}
			if idxe != "" {
				iv, iins, err := pe(f, regs, idxe)
				if err != nil {
					return nil, err
				}
				elem := ir.TypUint8
				sname := ""
				if pm, ok := ptrMeta[bv]; ok {
					sname = pm.structName
				}
				field := fname
				baseClean := strings.Trim(base, "()")
				if strings.Contains(baseClean, "->") {
					parts := strings.Split(baseClean, "->")
					svar := strings.TrimSpace(parts[0])
					if len(parts) > 1 {
						field = strings.TrimSpace(parts[1])
					}
					if ri, ok := regs[svar]; ok {
						sname = ri.structName
					}
					if sname == "" {
						for _, p := range f.Params {
							if p.Name == svar {
								sname = p.StructName
								break
							}
						}
					}
				} else if sname == "" {
					if ri, ok := regs[baseClean]; ok {
						sname = ri.structName
					}
					if sname == "" {
						for _, p := range f.Params {
							if p.Name == baseClean {
								sname = p.StructName
								break
							}
						}
					}
				}
				if sname == "" {
					for _, st := range structEnv {
						if sf := fieldOf(&st, field); sf != nil {
							sname = st.Name
							break
						}
					}
				}
				fld := f.Alloc()
				var alen int64
				if st := findStruct(sname, structEnv); st != nil {
					if sf := fieldOf(st, field); sf != nil {
						elem = sf.Type
						alen = int64(sf.ArrayLen)
					}
				}
				ins := append(bins, iins...)
				ins = append(ins, ir.Instr{Op: ir.OpField, Dst: fld, Args: []ir.Value{bv}, Sym: field, Elem: elem, Imm: alen})
				ins = append(ins, ir.Instr{Op: ir.OpStore, Args: []ir.Value{fld, iv, val}, Elem: elem})
				return ins, nil
			}
			ins := append(bins, ir.Instr{Op: ir.OpFStore, Args: []ir.Value{bv, val}, Sym: fname})
			return ins, nil
		}
	}
	// (base)[idx] or (ctx->h)[i]
	if strings.HasSuffix(lhs, "]") {
		if j := strings.LastIndexByte(lhs, '['); j > 0 {
			base := strings.TrimSpace(lhs[:j])
			idxe := lhs[j+1 : len(lhs)-1]
			base = stripOuterParens(base)
			// if base has ->, pe will load field array
			bv, bins, err := pe(f, regs, base)
			if err == nil {
				iv, iins, err2 := pe(f, regs, idxe)
				if err2 != nil {
					return nil, err2
				}
				ins := append(bins, iins...)
				root, idx, elem, extra := scalePtrIndex(f, bv, iv)
				if elem == "" {
					elem = ir.TypUint8
				}
				ins = append(ins, extra...)
				ins = append(ins, ir.Instr{Op: ir.OpStore, Args: []ir.Value{root, idx, val}, Elem: elem})
				return ins, nil
			}
		}
	}
	if strings.HasPrefix(lhs, "*") {
		p, ins, err := pe(f, regs, strings.TrimSpace(lhs[1:]))
		if err != nil {
			return nil, err
		}
		elem := ir.TypUint8
		if pm, ok := ptrMeta[p]; ok && pm.typ != "" {
			elem = pm.typ
		}
		if pm, ok := ptrMeta[p]; ok && pm.hasBase && pm.base != ir.NoVal {
			ins = append(ins, ir.Instr{Op: ir.OpStore, Args: []ir.Value{pm.v, pm.base, val}, Elem: elem})
		} else {
			ins = append(ins, ir.Instr{Op: ir.OpStore, Args: []ir.Value{p, val}, Elem: elem})
		}
		return ins, nil
	}
	// ((TYPE*)ptr)[idx]
	reCast := regexp.MustCompile(`^\(\(\s*([A-Za-z_][A-Za-z0-9_]*)\s*\*\s*\)\s*([A-Za-z_][A-Za-z0-9_]*)\s*\)\s*\[([^\]]+)\]$`)
	if m := reCast.FindStringSubmatch(lhs); m != nil {
		base, ok := regs[m[2]]
		if !ok {
			return nil, &Error{Code: ErrParse, Message: "store base: " + m[2]}
		}
		iv, ins, err := pe(f, regs, m[3])
		if err != nil {
			return nil, err
		}
		mt := mapType(m[1])
		scale := 1
		switch mt {
		case ir.TypUint32:
			scale = 4
		case ir.TypUint64:
			scale = 8
		}
		idx := iv
		if scale != 1 {
			sc := f.Alloc()
			ins = append(ins, ir.Instr{Op: ir.OpConst, Dst: sc, Imm: int64(scale)})
			scaled := f.Alloc()
			ins = append(ins, ir.Instr{Op: ir.OpMul, Dst: scaled, Args: []ir.Value{iv, sc}})
			idx = scaled
		}
		ins = append(ins, ir.Instr{Op: ir.OpStore, Args: []ir.Value{base.v, idx, val}, Elem: mt})
		return ins, nil
	}
	re := regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*\[([^\]]+)\]$`)
	m := re.FindStringSubmatch(lhs)
	if m == nil {
		return nil, &Error{Code: ErrParse, Message: "store lhs: " + lhs}
	}
	base, ok := regs[m[1]]
	if !ok {
		return nil, &Error{Code: ErrParse, Message: "store base: " + m[1]}
	}
	idx, ins, err := pe(f, regs, m[2])
	if err != nil {
		return nil, err
	}
	ins = append(ins, ir.Instr{Op: ir.OpStore, Args: []ir.Value{base.v, idx, val}, Elem: base.typ})
	return ins, nil
}

func firstWord(s string) string {
	fs := strings.Fields(s)
	if len(fs) == 0 {
		return ""
	}
	return fs[0]
}

func pe(f *ir.Func, regs map[string]regInfo, expr string) (ir.Value, []ir.Instr, error) {
	saved := f.Body
	f.Body = nil
	v, err := parseExpr(f, regs, expr)
	ins := f.Body
	f.Body = saved
	return v, ins, err
}

func parseExpr(f *ir.Func, regs map[string]regInfo, expr string) (ir.Value, error) {
	expr = stripOuterParens(expr)
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return ir.NoVal, &Error{Code: ErrParse, Message: "empty expr"}
	}
	// strip outer parens first so (*p) / ((x)<<n) re-enter unary/binop paths
	for strings.HasPrefix(expr, "(") && strings.HasSuffix(expr, ")") && balanced(expr[1:len(expr)-1]) {
		inner := strings.TrimSpace(expr[1 : len(expr)-1])
		if isTypeToken(inner) {
			break
		}
		// cast (TYPE)x — type token only inside parens, rest after ) handled below
		if rest := strings.TrimSpace(expr[len(expr):]); false {
			_ = rest
		}
		// if inner looks like TYPE)rest split — leave to cast handler
		if idx := strings.IndexByte(inner, ')'); idx < 0 {
			// full wrap
			if isTypeToken(strings.Fields(inner)[0]) && strings.Contains(inner, " ") {
				break
			}
			expr = inner
			continue
		}
		expr = inner
	}

	// string literal "..." in expression: hoist as global constant / slice
	if strings.HasPrefix(expr, "\"") && strings.HasSuffix(expr, "\"") && len(expr) >= 2 {
		rawStr, err := strconv.Unquote(expr)
		if err == nil {
			h := sha256.Sum256([]byte(rawStr))
			gName := fmt.Sprintf("__str_%x", h[:4])
			g := ir.Global{Name: gName, Data: rawStr, Type: ir.TypUint8}
			pendingGlobals = append(pendingGlobals, g)
			dst := f.Alloc()
			ensureScratch(f)
			f.Body = append(f.Body, ir.Instr{Op: ir.OpMov, Dst: dst, Sym: gName, Elem: ir.TypUint8})
			pm := regInfo{v: dst, typ: ir.TypUint8, ptr: true, elemIndex: true}
			ptrMeta[dst] = pm
			return dst, nil
		}
	}

	// character literal 'c' in expression
	if strings.HasPrefix(expr, "'") && strings.HasSuffix(expr, "'") && len(expr) >= 2 {
		if n, err := parseIntLit(expr); err == nil {
			dst := f.Alloc()
			ensureScratch(f)
			f.Body = append(f.Body, ir.Instr{Op: ir.OpConst, Dst: dst, Imm: n, Elem: ir.TypUint8})
			return dst, nil
		}
	}

	// ternary a ? b : c — only at paren-depth 0 (so "(a?b:c)>>3" is binop first)
	if q := indexByteAtDepth(expr, '?', 0); q > 0 {
		if depthZeroColon := findTernaryColon(expr, q); depthZeroColon > q {
			cond := strings.TrimSpace(expr[:q])
			tpart := strings.TrimSpace(expr[q+1 : depthZeroColon])
			fpart := strings.TrimSpace(expr[depthZeroColon+1:])
			cv, err := parseExpr(f, regs, cond)
			if err != nil {
				return ir.NoVal, err
			}
			tv, err := parseExpr(f, regs, tpart)
			if err != nil {
				return ir.NoVal, err
			}
			fv, err := parseExpr(f, regs, fpart)
			if err != nil {
				return ir.NoVal, err
			}
			dst := f.Alloc()
			ensureScratch(f)
			f.Body = append(f.Body, ir.Instr{Op: ir.OpCall, Dst: dst, Args: []ir.Value{cv, tv, fv}, Sym: "__select"})
			return dst, nil
		}
	}

	// sizeof(x) or sizeof *(x)
	if strings.HasPrefix(expr, "sizeof") {
		rest := strings.TrimSpace(expr[len("sizeof"):])
		rest = strings.TrimPrefix(rest, "(")
		rest = strings.TrimSuffix(rest, ")")
		rest = strings.TrimSpace(rest)
		rest = strings.TrimPrefix(rest, "*")
		rest = strings.TrimSpace(rest)
		n := int64(1)
		if ri, ok := regs[rest]; ok {
			elemSize := int64(1)
			switch ri.typ {
			case ir.TypUint8, ir.TypInt8:
				elemSize = 1
			case ir.TypUint16, ir.TypInt16:
				elemSize = 2
			case ir.TypUint32, ir.TypInt32, ir.TypFloat32:
				elemSize = 4
			case ir.TypUint64, ir.TypInt64, ir.TypFloat64:
				elemSize = 8
			case ir.TypUint128:
				elemSize = 16
			}
			if ri.localArr > 0 {
				n = int64(ri.localArr) * elemSize
			} else if ri.structName != "" {
				if st := findStruct(ri.structName, structEnv); st != nil {
					n = int64(estimateStructSize(st))
				}
			} else {
				n = elemSize
			}
		} else if st := findStruct(rest, structEnv); st != nil {
			n = int64(estimateStructSize(st))
		} else {
			switch rest {
			case "char", "uint8_t", "int8_t", "u8", "i8", "bool", "_Bool", "unsigned char", "signed char":
				n = 1
			case "short", "uint16_t", "int16_t", "u16", "i16", "unsigned short", "signed short":
				n = 2
			case "int", "uint32_t", "int32_t", "u32", "i32", "unsigned int", "signed int", "float", "float32":
				n = 4
			case "long", "long long", "uint64_t", "int64_t", "u64", "i64", "usize", "size_t", "uintptr_t", "double", "float64", "unsigned long", "unsigned long long":
				n = 8
			case "uint128_t", "int128_t", "u128", "i128", "__uint128_t", "unsigned __int128":
				n = 16
			default:
				if strings.HasSuffix(rest, "*") {
					n = 8
				} else {
					n = 64
				}
			}
		}
		dst := f.Alloc()
		ensureScratch(f)
		f.Body = append(f.Body, ir.Instr{Op: ir.OpConst, Dst: dst, Imm: n})
		return dst, nil
	}

	// unary &  — address of local (struct) or &arr[i] or &p->field
	if strings.HasPrefix(expr, "&") && len(expr) > 1 && expr[1] != '&' {
		inner := strings.TrimSpace(expr[1:])
		inner = stripOuterParens(inner)
		if arrow := lastIndexArrow(inner); arrow > 0 || strings.Contains(inner, ".") {
			v, err := parseExpr(f, regs, inner)
			if err == nil {
				dst := f.Alloc()
				ensureScratch(f)
				f.Body = append(f.Body, ir.Instr{Op: ir.OpMov, Dst: dst, Args: []ir.Value{v}, Sym: "addr_of"})
				ptrMeta[dst] = regInfo{v: dst, ptr: true}
				return dst, nil
			}
		}
		// &name[idx]
		if b := strings.IndexByte(inner, '['); b > 0 && strings.HasSuffix(inner, "]") {
			aname := strings.TrimSpace(inner[:b])
			idxe := inner[b+1 : len(inner)-1]
			if ri, ok := regs[aname]; ok && ri.ptr {
				iv, err := parseExpr(f, regs, idxe)
				if err != nil {
					return ir.NoVal, err
				}
				// pointer to element: base + idx (element scale)
				dst := f.Alloc()
				ensureScratch(f)
				// For struct arrays, element is one struct; for scalar arrays, one elem.
				f.Body = append(f.Body, ir.Instr{Op: ir.OpAdd, Dst: dst, Args: []ir.Value{ri.v, iv}, Sym: "ptr_add"})
				sname := ri.structName
				// if array of structs, peep struct type from localArr declaration context — leave empty
				ptrMeta[dst] = regInfo{v: dst, typ: ri.typ, ptr: true, scale: 1, elemIndex: true, structName: sname}
				notePtr(dst, ri.typ, 1, ir.NoVal)
				return dst, nil
			}
		}
		if ri, ok := regs[inner]; ok {
			dst := f.Alloc()
			ensureScratch(f)
			sym := "addr_of"
			elem := ri.typ
			if ri.structName != "" {
				elem = ir.TypeName(ri.structName)
			}
			f.Body = append(f.Body, ir.Instr{Op: ir.OpMov, Dst: dst, Args: []ir.Value{ri.v}, Sym: sym, Elem: elem})
			ptrMeta[dst] = regInfo{v: dst, typ: ri.typ, ptr: true, structName: ri.structName}
			return dst, nil
		}
		return ir.NoVal, &Error{Code: ErrParse, Message: "addr: " + inner}
	}

	// binary ops (precedence level 1..10) — run before unary ops so ~x + 1 splits by + first
	for _, ops := range [][]string{{"||"}, {"&&"}, {"==", "!=", "<=", ">=", "<", ">"}, {"|"}, {"^"}, {"&"}, {"<<", ">>"}, {"+", "-"}, {"*", "/", "%"}} {
		if idx, op := findOp(expr, ops); idx >= 0 {
			if idx == 0 || idx+len(op) == len(expr) {
				continue
			}
			left, err := parseExpr(f, regs, expr[:idx])
			if err != nil {
				return ir.NoVal, err
			}
			right, err := parseExpr(f, regs, expr[idx+len(op):])
			if err != nil {
				return ir.NoVal, err
			}
			dst := f.Alloc()
			ensureScratch(f)
			// emit-only bool: __cmp_ ops are IR pseudo-symbols intercepted by emitIns (never emitted as C/Go function calls or stubs)
			if op == "==" || op == "!=" || op == "<" || op == ">" || op == "<=" || op == ">=" || op == "&&" || op == "||" {
				f.Body = append(f.Body, ir.Instr{Op: ir.OpCall, Dst: dst, Args: []ir.Value{left, right}, Sym: "__cmp_" + op})
				return dst, nil
			}
			bop := binOp(op)
			// pointer ± byte offset → root+offset meta (negative index safe, murmur blocks)
			if op == "+" || op == "-" {
				leftPtr := false
				var pm regInfo
				if p, ok := ptrMeta[left]; ok && p.ptr && !intScalarReg(p) {
					pm, leftPtr = p, true
				}
				if leftPtr {
					rightPtr := false
					if p, ok := ptrMeta[right]; ok && p.ptr && !intScalarReg(p) {
						rightPtr = true
					}
					if op == "-" && rightPtr {
						diff := f.Alloc()
						f.Body = append(f.Body, ir.Instr{Op: ir.OpSub, Dst: diff, Args: []ir.Value{left, right}, Elem: ir.TypUint64})
						return diff, nil
					}
					root := left
					if pm.hasBase {
						root = pm.v
					}
					var off ir.Value
					if op == "+" {
						if pm.hasBase {
							off = f.Alloc()
							f.Body = append(f.Body, ir.Instr{Op: ir.OpAdd, Dst: off, Args: []ir.Value{pm.base, right}})
						} else {
							off = right
						}
					} else {
						if pm.hasBase {
							off = f.Alloc()
							f.Body = append(f.Body, ir.Instr{Op: ir.OpSub, Dst: off, Args: []ir.Value{pm.base, right}})
						} else {
							z := f.Alloc()
							f.Body = append(f.Body, ir.Instr{Op: ir.OpConst, Dst: z, Imm: 0})
							off = f.Alloc()
							f.Body = append(f.Body, ir.Instr{Op: ir.OpSub, Dst: off, Args: []ir.Value{z, right}})
						}
					}
					f.Body = append(f.Body, ir.Instr{Op: ir.OpMov, Dst: dst, Args: []ir.Value{root}, Sym: "ptr_alias"})
					ptrMeta[dst] = regInfo{v: root, typ: pm.typ, ptr: true, scale: pm.scale, base: off, hasBase: true, elemIndex: pm.elemIndex, structName: pm.structName}
					return dst, nil
				} else if op == "+" {
					rightPtr := false
					if p, ok := ptrMeta[right]; ok && p.ptr && !intScalarReg(p) {
						pm, rightPtr = p, true
					}
					if rightPtr {
						root := right
						if pm.hasBase {
							root = pm.v
						}
						var off ir.Value
						if pm.hasBase {
							off = f.Alloc()
							f.Body = append(f.Body, ir.Instr{Op: ir.OpAdd, Dst: off, Args: []ir.Value{pm.base, left}})
						} else {
							off = left
						}
						f.Body = append(f.Body, ir.Instr{Op: ir.OpMov, Dst: dst, Args: []ir.Value{root}, Sym: "ptr_alias"})
						ptrMeta[dst] = regInfo{v: root, typ: pm.typ, ptr: true, scale: pm.scale, base: off, hasBase: true, elemIndex: pm.elemIndex, structName: pm.structName}
						return dst, nil
					}
				}
			}
			f.Body = append(f.Body, ir.Instr{Op: bop, Dst: dst, Args: []ir.Value{left, right}})
			return dst, nil
		}
	}

	// unary logical not !
	if strings.HasPrefix(expr, "!") && len(expr) > 1 && expr[1] != '=' {
		inner, err := parseExpr(f, regs, strings.TrimSpace(expr[1:]))
		if err != nil {
			return ir.NoVal, err
		}
		z := f.Alloc()
		dst := f.Alloc()
		ensureScratch(f)
		f.Body = append(f.Body,
			ir.Instr{Op: ir.OpConst, Dst: z, Imm: 0},
			ir.Instr{Op: ir.OpCall, Dst: dst, Args: []ir.Value{inner, z}, Sym: "__cmp_=="},
		)
		return dst, nil
	}
	// unary bitwise not ~
	if strings.HasPrefix(expr, "~") {
		inner, err := parseExpr(f, regs, strings.TrimSpace(expr[1:]))
		if err != nil {
			return ir.NoVal, err
		}
		dst := f.Alloc()
		ensureScratch(f)
		f.Body = append(f.Body, ir.Instr{Op: ir.OpNot, Dst: dst, Args: []ir.Value{inner}})
		return dst, nil
	}
	// unary minus
	if strings.HasPrefix(expr, "-") && len(expr) > 1 && expr[1] != '-' {
		inner, err := parseExpr(f, regs, strings.TrimSpace(expr[1:]))
		if err != nil {
			return ir.NoVal, err
		}
		z := f.Alloc()
		dst := f.Alloc()
		ensureScratch(f)
		f.Body = append(f.Body,
			ir.Instr{Op: ir.OpConst, Dst: z, Imm: 0},
			ir.Instr{Op: ir.OpSub, Dst: dst, Args: []ir.Value{z, inner}},
		)
		return dst, nil
	}
	// unary *
	if strings.HasPrefix(expr, "*") && !strings.HasPrefix(expr, "*=") {
		name := strings.TrimSpace(expr[1:])
		if strings.HasSuffix(name, "++") {
			pname := strings.TrimSpace(strings.TrimSuffix(name, "++"))
			if ri, ok := regs[pname]; ok && ri.ptr && ri.structName == "" {
				slot := ensureOffSlot(f, regs, pname)
				ri = regs[pname]
				oldv := f.Alloc()
				one := f.Alloc()
				tmp := f.Alloc()
				dst := f.Alloc()
				ensureScratch(f)
				elem := ri.typ
				if elem == "" {
					elem = ir.TypUint8
				}
				f.Body = append(f.Body,
					ir.Instr{Op: ir.OpMov, Dst: oldv, Args: []ir.Value{slot}, Elem: ir.TypUint64},
					ir.Instr{Op: ir.OpConst, Dst: one, Imm: 1, Elem: ir.TypUint64},
					ir.Instr{Op: ir.OpAdd, Dst: tmp, Args: []ir.Value{slot, one}, Elem: ir.TypUint64},
					ir.Instr{Op: ir.OpMov, Dst: slot, Args: []ir.Value{tmp}, Elem: ir.TypUint64},
					ir.Instr{Op: ir.OpLoad, Dst: dst, Args: []ir.Value{ri.v, oldv}, Elem: elem},
				)
				return dst, nil
			}
		}
		if strings.HasPrefix(name, "++") {
			pname := strings.TrimSpace(strings.TrimPrefix(name, "++"))
			if ri, ok := regs[pname]; ok && ri.ptr && ri.structName == "" {
				slot := ensureOffSlot(f, regs, pname)
				ri = regs[pname]
				one := f.Alloc()
				tmp := f.Alloc()
				dst := f.Alloc()
				ensureScratch(f)
				elem := ri.typ
				if elem == "" {
					elem = ir.TypUint8
				}
				f.Body = append(f.Body,
					ir.Instr{Op: ir.OpConst, Dst: one, Imm: 1, Elem: ir.TypUint64},
					ir.Instr{Op: ir.OpAdd, Dst: tmp, Args: []ir.Value{slot, one}, Elem: ir.TypUint64},
					ir.Instr{Op: ir.OpMov, Dst: slot, Args: []ir.Value{tmp}, Elem: ir.TypUint64},
					ir.Instr{Op: ir.OpLoad, Dst: dst, Args: []ir.Value{ri.v, slot}, Elem: elem},
				)
				return dst, nil
			}
		}
		// byte-buffer cursor: *p → p[offSlot], not p[0] (poly1305 message++, etc.)
		// Note: params set elemIndex=true for all ptrs; gate on elem type only.
		if identOnly(name) {
			if ri, ok := regs[name]; ok && ri.ptr && ri.structName == "" {
				if strings.HasPrefix(string(ri.typ), "*[]") {
					dst := f.Alloc()
					ensureScratch(f)
					elemType := ir.TypeName(strings.TrimPrefix(string(ri.typ), "*[]"))
					if elemType == "byte" {
						elemType = ir.TypUint8
					}
					f.Body = append(f.Body, ir.Instr{Op: ir.OpLoad, Dst: dst, Args: []ir.Value{ri.v}, Elem: elemType, Sym: "deref_slice_ptr"})
					ptrMeta[dst] = regInfo{v: dst, typ: elemType, ptr: true, elemIndex: true}
					return dst, nil
				}
				slot := ensureOffSlot(f, regs, name)
				ri = regs[name]
				dst := f.Alloc()
				ensureScratch(f)
				elem := ri.typ
				if elem == "" {
					elem = ir.TypUint8
				}
				f.Body = append(f.Body, ir.Instr{Op: ir.OpLoad, Dst: dst, Args: []ir.Value{ri.v, slot}, Elem: elem})
				return dst, nil
			}
		}
		inner, err := parseExpr(f, regs, name)
		if err != nil {
			return ir.NoVal, err
		}
		dst := f.Alloc()
		ensureScratch(f)
		elem := ir.TypUint8
		if pm, ok := ptrMeta[inner]; ok && pm.typ != "" {
			elem = pm.typ
		} else if ri, ok := regs[name]; ok && ri.typ != "" {
			elem = ri.typ
		}
		if pm, ok := ptrMeta[inner]; ok && pm.hasBase && pm.base != ir.NoVal {
			f.Body = append(f.Body, ir.Instr{Op: ir.OpLoad, Dst: dst, Args: []ir.Value{pm.v, pm.base}, Elem: elem})
		} else {
			f.Body = append(f.Body, ir.Instr{Op: ir.OpLoad, Dst: dst, Args: []ir.Value{inner}, Elem: elem})
		}
		return dst, nil
	}

	// (TYPE *)expr or (TYPE)expr cast
	if strings.HasPrefix(expr, "(") {
		if end := strings.IndexByte(expr, ')'); end > 1 {
			maybe := strings.TrimSpace(expr[1:end])
			rest := strings.TrimSpace(expr[end+1:])
			// pointer cast
			if strings.HasSuffix(maybe, "*") {
				et := strings.TrimSpace(strings.TrimSuffix(maybe, "*"))
				et = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(et), "const")), "struct"))
				if isTypeToken(et) {
					inner, err := parseExpr(f, regs, rest)
					if err != nil {
						return ir.NoVal, err
					}
					dst := f.Alloc()
					ensureScratch(f)
					mt := mapType(et)
					if isStructType(et, structEnv) {
						mt = ir.TypeName("*" + et)
					}
					f.Body = append(f.Body, ir.Instr{Op: ir.OpMov, Dst: dst, Args: []ir.Value{inner}, Sym: "ptr_cast", Elem: mt})
					scale := 1
					switch mt {
					case ir.TypUint32:
						scale = 4
					case ir.TypUint64:
						scale = 8
					}
					// typed pointer cast → word index with target scale on root []byte
					// Do NOT keep source elemIndex when element type changes (uint8*→uint64*).
					ri := regInfo{v: inner, typ: mt, ptr: true, scale: scale, elemIndex: false}
					if pm, ok := ptrMeta[inner]; ok {
						ri.v = pm.v
						if pm.hasBase {
							ri.base, ri.hasBase = pm.base, true
						}
						if pm.elemIndex && pm.typ == mt {
							ri.elemIndex = true
							ri.scale = 1
						}
					}
					ptrMeta[dst] = ri
					return dst, nil
				}
			}
			if isTypeToken(maybe) {
				inner, err := parseExpr(f, regs, rest)
				if err != nil {
					return ir.NoVal, err
				}
				// materialize cast as Mov with Elem=target type for emit widen
				dst := f.Alloc()
				ensureScratch(f)
				f.Body = append(f.Body, ir.Instr{Op: ir.OpMov, Dst: dst, Args: []ir.Value{inner}, Elem: mapType(maybe), Sym: "cast"})
				return dst, nil
			}
		}
	}

	// index base[idx] — match [ for trailing ]
	if len(expr) > 0 && expr[len(expr)-1] == ']' {
		depth := 0
		j := -1
		for k := len(expr) - 1; k >= 0; k-- {
			if expr[k] == ']' {
				depth++
			} else if expr[k] == '[' {
				depth--
				if depth == 0 {
					j = k
					break
				}
			}
		}
		if j > 0 {
			base := strings.TrimSpace(expr[:j])
			idxe := expr[j+1 : len(expr)-1]
			// 2D: table[row][col] with global Cols stride
			if tname, rowExpr, ok2 := splitTrailingIndex(base); ok2 {
				if ri, ok := regs[tname]; ok && ri.cols > 0 {
					rowv, err := parseExpr(f, regs, rowExpr)
					if err != nil {
						return ir.NoVal, err
					}
					colv, err := parseExpr(f, regs, idxe)
					if err != nil {
						return ir.NoVal, err
					}
					dst := f.Alloc()
					ensureScratch(f)
					sc := f.Alloc()
					mul := f.Alloc()
					sum := f.Alloc()
					f.Body = append(f.Body,
						ir.Instr{Op: ir.OpConst, Dst: sc, Imm: int64(ri.cols)},
						ir.Instr{Op: ir.OpMul, Dst: mul, Args: []ir.Value{rowv, sc}},
						ir.Instr{Op: ir.OpAdd, Dst: sum, Args: []ir.Value{mul, colv}},
						ir.Instr{Op: ir.OpLoad, Dst: dst, Args: []ir.Value{ri.v, sum}, Elem: ri.typ},
					)
					delete(ptrMeta, dst)
					return dst, nil
				}
			}
			cleanBase := base
			if strings.HasPrefix(cleanBase, "(") && strings.HasSuffix(cleanBase, ")") {
				sub := strings.TrimSpace(cleanBase[1 : len(cleanBase)-1])
				if balanced(sub) {
					cleanBase = sub
				}
			}
			if bv, err := parseExpr(f, regs, cleanBase); err == nil {
				iv, err := parseExpr(f, regs, idxe)
				if err != nil {
					return ir.NoVal, err
				}
				dst := f.Alloc()
				ensureScratch(f)
				// Prefer named base metadata (blocks → root + byte off)
				var root, idx ir.Value
				var elem ir.TypeName
				var extra []ir.Instr
				sname := ""
				if pm, ok := ptrMeta[bv]; ok && pm.structName != "" {
					sname = pm.structName
				}
				if ri, ok := regs[base]; ok && ri.cols > 0 && ri.typ == ir.TypUint8 && ri.localArr > 0 {
					sc := f.Alloc()
					mul := f.Alloc()
					f.Body = append(f.Body,
						ir.Instr{Op: ir.OpConst, Dst: sc, Imm: int64(ri.cols)},
						ir.Instr{Op: ir.OpMul, Dst: mul, Args: []ir.Value{iv, sc}},
						ir.Instr{Op: ir.OpAdd, Dst: dst, Args: []ir.Value{ri.v, mul}, Sym: "ptr_add"},
					)
					ptrMeta[dst] = regInfo{v: dst, typ: ri.typ, ptr: true, scale: 1, elemIndex: true}
					notePtr(dst, ri.typ, 1, ir.NoVal)
					return dst, nil
				}
				if ri, ok := regs[base]; ok && ri.ptr {
					root, idx, elem, extra = scalePtrIndex(f, ri.v, iv)
					if ri.structName != "" {
						sname = ri.structName
					}
				} else {
					root, idx, elem, extra = scalePtrIndex(f, bv, iv)
				}
				if elem == "" {
					if sname != "" {
						elem = ir.TypeName(sname)
					} else {
						elem = ir.TypUint8
					}
				}
				f.Body = append(f.Body, extra...)
				delete(ptrMeta, dst)
				if sname != "" {
					ptrMeta[dst] = regInfo{v: dst, typ: elem, structName: sname}
				}
				f.Body = append(f.Body, ir.Instr{Op: ir.OpLoad, Dst: dst, Args: []ir.Value{root, idx}, Elem: elem})
				return dst, nil
			}
		}
	}

	// postfix ++/--
	if (strings.HasSuffix(expr, "++") || strings.HasSuffix(expr, "--")) && !strings.Contains(expr, "=") {
		op := expr[len(expr)-2:]
		rawName := strings.TrimSpace(expr[:len(expr)-2])
		name := strings.TrimPrefix(stripOuterParens(rawName), "*")
		if identOnly(name) {
			ri, ok := regs[name]
			if !ok {
				return ir.NoVal, &Error{Code: ErrParse, Message: "inc: " + name}
			}
			ensureScratch(f)
			// slice/buffer cursor already tracked by offSlot (set by *p or lazy bump scan)
			if ri.ptr && ri.structName == "" && (ri.offSlotSet || ri.typ == "" || ri.typ == ir.TypUint8 || ri.typ == ir.TypInt) {
				slot := ensureOffSlot(f, regs, name)
				ri = regs[name]
				old := f.Alloc()
				f.Body = append(f.Body, ir.Instr{Op: ir.OpMov, Dst: old, Args: []ir.Value{slot}, Elem: ir.TypUint64})
				bopSlot := ir.OpAdd
				if op == "--" {
					bopSlot = ir.OpSub
				}
				oneSlot := f.Alloc()
				tmpSlot := f.Alloc()
				f.Body = append(f.Body,
					ir.Instr{Op: ir.OpConst, Dst: oneSlot, Imm: 1, Elem: ir.TypUint64},
					ir.Instr{Op: bopSlot, Dst: tmpSlot, Args: []ir.Value{ri.offSlot, oneSlot}, Elem: ir.TypUint64, Sym: "offslot"},
					ir.Instr{Op: ir.OpMov, Dst: ri.offSlot, Args: []ir.Value{tmpSlot}, Elem: ir.TypUint64, Sym: "offslot"},
				)
				ri.base = ri.offSlot
				regs[name] = ri
				ptrMeta[ri.v] = ri
				return old, nil
			}
			// scalar / non-buffer: classic ++ on the register value
			old := f.Alloc()
			f.Body = append(f.Body, ir.Instr{Op: ir.OpMov, Dst: old, Args: []ir.Value{ri.v}})
			one := f.Alloc()
			tmp := f.Alloc()
			bin := ir.OpAdd
			if op == "--" {
				bin = ir.OpSub
			}
			f.Body = append(f.Body,
				ir.Instr{Op: ir.OpConst, Dst: one, Imm: 1},
				ir.Instr{Op: bin, Dst: tmp, Args: []ir.Value{ri.v, one}},
				ir.Instr{Op: ir.OpMov, Dst: ri.v, Args: []ir.Value{tmp}},
			)
			return old, nil
		}
		// a[i]++ / ctx->f++ / ctx->f[i]++
		if strings.Contains(rawName, "[") || strings.Contains(rawName, "->") {
			old, err := parseExpr(f, regs, rawName) // load
			if err != nil {
				return ir.NoVal, err
			}
			one := f.Alloc()
			tmp := f.Alloc()
			bin := ir.OpAdd
			if op == "--" {
				bin = ir.OpSub
			}
			ensureScratch(f)
			f.Body = append(f.Body,
				ir.Instr{Op: ir.OpConst, Dst: one, Imm: 1},
				ir.Instr{Op: bin, Dst: tmp, Args: []ir.Value{old, one}},
				ir.Instr{Op: ir.OpMov, Dst: old, Args: []ir.Value{tmp}},
			)
			// parseStore must not wipe Body via pe — save/restore
			saved := f.Body
			f.Body = nil
			stins, err := parseStore(f, regs, rawName, tmp)
			f.Body = saved
			if err != nil {
				return ir.NoVal, err
			}
			f.Body = append(f.Body, stins...)
			return old, nil
		}
	}
	// prefix ++/--
	if (strings.HasPrefix(expr, "++") || strings.HasPrefix(expr, "--")) && !strings.Contains(expr, "=") {
		op := expr[:2]
		rawName := strings.TrimSpace(expr[2:])
		name := strings.TrimPrefix(stripOuterParens(rawName), "*")
		if identOnly(name) {
			ri, ok := regs[name]
			if !ok {
				return ir.NoVal, &Error{Code: ErrParse, Message: "inc: " + name}
			}
			ensureScratch(f)
			if ri.ptr && ri.structName == "" && (ri.typ == "" || ri.typ == ir.TypUint8 || ri.typ == ir.TypInt) {
				_ = ensureOffSlot(f, regs, name)
				ri = regs[name]
				bopSlot := ir.OpAdd
				if op == "--" {
					bopSlot = ir.OpSub
				}
				oneSlot := f.Alloc()
				tmpSlot := f.Alloc()
				f.Body = append(f.Body,
					ir.Instr{Op: ir.OpConst, Dst: oneSlot, Imm: 1, Elem: ir.TypUint64},
					ir.Instr{Op: bopSlot, Dst: tmpSlot, Args: []ir.Value{ri.offSlot, oneSlot}, Elem: ir.TypUint64, Sym: "offslot"},
					ir.Instr{Op: ir.OpMov, Dst: ri.offSlot, Args: []ir.Value{tmpSlot}, Elem: ir.TypUint64, Sym: "offslot"},
				)
				ri.base = ri.offSlot
				regs[name] = ri
				ptrMeta[ri.v] = ri
				return ri.offSlot, nil
			}
			one := f.Alloc()
			tmp := f.Alloc()
			bin := ir.OpAdd
			if op == "--" {
				bin = ir.OpSub
			}
			f.Body = append(f.Body,
				ir.Instr{Op: ir.OpConst, Dst: one, Imm: 1},
				ir.Instr{Op: bin, Dst: tmp, Args: []ir.Value{ri.v, one}},
				ir.Instr{Op: ir.OpMov, Dst: ri.v, Args: []ir.Value{tmp}},
			)
			return ri.v, nil
		}
	}

	// parenthesized call: (*fnptr)(args) or (fnptr)(args)
	if strings.HasPrefix(expr, "(") {
		if close1, err := matchParen(expr, 0); err == nil && close1 < len(expr)-1 {
			head := strings.TrimSpace(expr[:close1+1])
			rest := strings.TrimSpace(expr[close1+1:])
			if strings.HasPrefix(rest, "(") && strings.HasSuffix(rest, ")") {
				if close2, err2 := matchParen(rest, 0); err2 == nil && close2 == len(rest)-1 {
					headClean := strings.TrimPrefix(stripOuterParens(head), "*")
					headClean = strings.TrimSpace(headClean)
					if identOnly(headClean) {
						fname := headClean
						argsRaw := rest[1 : len(rest)-1]
						var argVals []ir.Value
						if strings.TrimSpace(argsRaw) != "" {
							for _, a := range splitCSV(argsRaw) {
								v, err := parseExpr(f, regs, a)
								if err != nil {
									return ir.NoVal, err
								}
								argVals = append(argVals, v)
							}
						}
						dst := f.Alloc()
						ensureScratch(f)
						f.Body = append(f.Body, ir.Instr{Op: ir.OpCall, Dst: dst, Args: argVals, Sym: fname})
						return dst, nil
					}
				}
			}
		}
	}

	if i := strings.IndexByte(expr, '('); i > 0 && strings.HasSuffix(expr, ")") {
		head := strings.TrimSpace(expr[:i])
		headClean := strings.TrimPrefix(stripOuterParens(head), "*")
		headClean = strings.TrimSpace(headClean)
		isPureCall := identOnly(head) || identOnly(headClean)
		if isPureCall && !identOnly(head) {
			head = headClean
		}
		isMethodCall := false
		var mBase, mField string
		if !isPureCall {
			if isFieldChain(head) {
				isMethodCall = true
				mBase = head
			} else if arrow := indexArrow(head); arrow > 0 && identOnly(strings.TrimSpace(head[:arrow])) && identOnly(strings.TrimSpace(head[arrow+2:])) {
				isMethodCall = true
				mBase = strings.TrimSpace(head[:arrow])
				mField = strings.TrimSpace(head[arrow+2:])
			} else if dot := indexByteAtDepth(head, '.', 0); dot > 0 && identOnly(strings.TrimSpace(head[:dot])) && identOnly(strings.TrimSpace(head[dot+1:])) {
				isMethodCall = true
				mBase = strings.TrimSpace(head[:dot])
				mField = strings.TrimSpace(head[dot+1:])
			}
		}
		if isPureCall || isMethodCall {
			close, err := matchParen(expr, i)
			if err != nil || close != len(expr)-1 {
				// not a pure call (e.g. f(x) | g(y)) — fall through to binops
				goto afterCall
			}
			fname := head
			if isMethodCall && mField != "" {
				fname = mBase + "->" + mField
			}
			argsRaw := expr[i+1 : close]
			var argVals []ir.Value
			if strings.TrimSpace(argsRaw) != "" {
				for _, a := range splitCSV(argsRaw) {
					v, err := parseExpr(f, regs, a)
					if err != nil {
						return ir.NoVal, err
					}
					// materialize ptr+off as reslice for call args (load32_le(src+i*4))
					if pm, ok := ptrMeta[v]; ok && pm.ptr && pm.hasBase && pm.structName == "" && pm.base != pm.v && pm.base != ir.NoVal {
						nv := f.Alloc()
						ensureScratch(f)
						f.Body = append(f.Body, ir.Instr{Op: ir.OpAdd, Dst: nv, Args: []ir.Value{pm.v, pm.base}, Sym: "ptr_add"})
						ptrMeta[nv] = regInfo{v: nv, typ: pm.typ, ptr: true, scale: pm.scale, elemIndex: pm.elemIndex}
						v = nv
					}
					argVals = append(argVals, v)
				}
			}
			dst := f.Alloc()
			ensureScratch(f)
			f.Body = append(f.Body, ir.Instr{Op: ir.OpCall, Dst: dst, Args: argVals, Sym: fname})
			return dst, nil
		}
	}

afterCall:
	// a.field or a.field[i] or arr[i].field (struct value, monocypher blocks[i].a / tmp_c.Yp)
	if dot := indexByteAtDepth(expr, '.', 0); dot > 0 && (dot > lastIndexArrow(expr) || !strings.Contains(expr, "->")) {
		base := strings.TrimSpace(expr[:dot])
		rest := strings.TrimSpace(expr[dot+1:])
		fname := rest
		idxe := ""
		if b := strings.IndexByte(rest, '['); b >= 0 && strings.HasSuffix(rest, "]") {
			fname = strings.TrimSpace(rest[:b])
			idxe = rest[b+1 : len(rest)-1]
		}
		if identOnly(fname) {
			var baseVal ir.Value
			var sname string
			if identOnly(base) {
				if ri, ok := regs[base]; ok && ri.structName != "" {
					baseVal = ri.v
					sname = ri.structName
				}
			}
			if baseVal == ir.NoVal || sname == "" {
				bv, err := parseExpr(f, regs, base)
				if err != nil {
					return ir.NoVal, err
				}
				baseVal = bv
				if pm, ok := ptrMeta[bv]; ok {
					sname = pm.structName
				}
				if sname == "" {
					for _, ri := range regs {
						if ri.v == bv && ri.structName != "" {
							sname = ri.structName
							break
						}
					}
				}
				if sname == "" {
					root := base
					if lb := strings.IndexByte(base, '['); lb > 0 {
						root = strings.TrimSpace(base[:lb])
					}
					if ri, ok := regs[root]; ok {
						sname = ri.structName
					}
					for _, p := range f.Params {
						if p.Name == root && p.StructName != "" {
							sname = p.StructName
							break
						}
					}
				}
				if sname == "" {
					for i := range structEnv {
						if fieldOf(&structEnv[i], fname) != nil {
							sname = structEnv[i].Name
							break
						}
					}
				}
				if sname == "" {
					return ir.NoVal, &Error{Code: ErrParse, Message: "dot field base: " + base}
				}
			}
			st := findStruct(sname, structEnv)
			sf := fieldOf(st, fname)
			if sf == nil {
				return ir.NoVal, &Error{Code: ErrParse, Message: "dot field: " + fname}
			}
			dst := f.Alloc()
			ensureScratch(f)
			f.Body = append(f.Body, ir.Instr{Op: ir.OpField, Dst: dst, Args: []ir.Value{baseVal}, Sym: fname, Elem: sf.Type, Imm: int64(sf.ArrayLen)})
			if isStructType(string(sf.Type), structEnv) {
				ptrMeta[dst] = regInfo{v: dst, typ: sf.Type, ptr: sf.ArrayLen < 0 || sf.ArrayLen > 0, structName: string(sf.Type), localArr: sf.ArrayLen, elemIndex: sf.ArrayLen > 0}
			} else if sf.ArrayLen != 0 {
				// array or pointer field decays to pointer/slice
				ptrMeta[dst] = regInfo{v: dst, typ: sf.Type, ptr: true, scale: 1, elemIndex: true, localArr: sf.ArrayLen}
				notePtr(dst, sf.Type, 1, ir.NoVal)
			}
			if idxe != "" {
				iv, err := parseExpr(f, regs, idxe)
				if err != nil {
					return ir.NoVal, err
				}
				ld := f.Alloc()
				f.Body = append(f.Body, ir.Instr{Op: ir.OpLoad, Dst: ld, Args: []ir.Value{dst, iv}, Elem: sf.Type})
				if isStructType(string(sf.Type), structEnv) {
					ptrMeta[ld] = regInfo{v: ld, typ: sf.Type, structName: string(sf.Type)}
				}
				return ld, nil
			}
			return dst, nil
		}
	}

	// p->field or p->field[i]  (after binops so ctx->x + 1 splits correctly)
	if arrow := lastIndexArrow(expr); arrow > 0 {
		base := strings.TrimSpace(expr[:arrow])
		rest := strings.TrimSpace(expr[arrow+2:])
		fname := rest
		idxe := ""
		if b := strings.IndexByte(rest, '['); b >= 0 && strings.HasSuffix(rest, "]") {
			fname = strings.TrimSpace(rest[:b])
			idxe = rest[b+1 : len(rest)-1]
		}
		// only pure field form; otherwise already handled by binops
		if identOnly(fname) {
			bv, err := parseExpr(f, regs, base)
			if err != nil {
				return ir.NoVal, err
			}
			sname := ""
			if pm, ok := ptrMeta[bv]; ok {
				sname = pm.structName
				if sname == "" && pm.typ != "" && isStructType(string(pm.typ), structEnv) {
					sname = string(pm.typ)
				}
			}
			if sname == "" {
				if ri, ok := regs[base]; ok {
					sname = ri.structName
					if sname == "" && ri.typ != "" && isStructType(string(ri.typ), structEnv) {
						sname = string(ri.typ)
					}
				}
			}
			// also try regs by value match
			if sname == "" {
				for _, ri := range regs {
					if ri.v == bv {
						sname = ri.structName
						if sname == "" && ri.typ != "" && isStructType(string(ri.typ), structEnv) {
							sname = string(ri.typ)
						}
						if sname != "" {
							break
						}
					}
				}
			}
			if sname == "" {
				for _, p := range f.Params {
					if p.Name == base {
						sname = p.StructName
						break
					}
				}
			}
			if sname == "" {
				for i := range structEnv {
					if fieldOf(&structEnv[i], fname) != nil {
						sname = structEnv[i].Name
						break
					}
				}
			}
			st := findStruct(sname, structEnv)
			sf := fieldOf(st, fname)
			if sf == nil {
				return ir.NoVal, &Error{Code: ErrParse, Message: "field: " + fname + " sname=" + sname}
			}
			dst := f.Alloc()
			ensureScratch(f)
			fi := ir.Instr{Op: ir.OpField, Dst: dst, Args: []ir.Value{bv}, Sym: fname, Elem: sf.Type, Imm: int64(sf.ArrayLen)}
			f.Body = append(f.Body, fi)
			if idxe != "" {
				iv, err := parseExpr(f, regs, idxe)
				if err != nil {
					return ir.NoVal, err
				}
				ld := f.Alloc()
				f.Body = append(f.Body, ir.Instr{Op: ir.OpLoad, Dst: ld, Args: []ir.Value{dst, iv}, Elem: sf.Type})
				if isStructType(string(sf.Type), structEnv) {
					ptrMeta[ld] = regInfo{v: ld, typ: sf.Type, structName: string(sf.Type)}
				}
				return ld, nil
			}
			if isStructType(string(sf.Type), structEnv) {
				ptrMeta[dst] = regInfo{v: dst, typ: sf.Type, ptr: sf.ArrayLen < 0, structName: string(sf.Type)}
			} else if sf.ArrayLen != 0 {
				ptrMeta[dst] = regInfo{v: dst, typ: sf.Type, ptr: true, scale: 1, elemIndex: true, localArr: sf.ArrayLen}
				notePtr(dst, sf.Type, 1, ir.NoVal)
			}
			return dst, nil
		}
	}

	if expr == "NULL" || expr == "null" || expr == "nil" {
		dst := f.Alloc()
		ensureScratch(f)
		f.Body = append(f.Body, ir.Instr{Op: ir.OpConst, Dst: dst, Imm: 0})
		return dst, nil
	}
	if ri, ok := regs[expr]; ok {
		// Keep root register; call-site / index path apply offSlot (avoid unused temps).
		return ri.v, nil
	}
	if n, err := parseIntLit(expr); err == nil {
		dst := f.Alloc()
		ensureScratch(f)
		f.Body = append(f.Body, ir.Instr{Op: ir.OpConst, Dst: dst, Imm: n})
		return dst, nil
	}
	cleanedExpr := strings.TrimSuffix(strings.TrimSuffix(expr, "f"), "F")
	if flt, err := strconv.ParseFloat(cleanedExpr, 64); err == nil && (strings.ContainsAny(cleanedExpr, ".eE") || strings.HasSuffix(expr, "f") || strings.HasSuffix(expr, "F")) {
		dst := f.Alloc()
		ensureScratch(f)
		elemTyp := ir.TypFloat64
		var imm int64
		if strings.HasSuffix(expr, "f") || strings.HasSuffix(expr, "F") {
			elemTyp = ir.TypFloat32
			imm = int64(math.Float32bits(float32(flt)))
		} else {
			imm = int64(math.Float64bits(flt))
		}
		f.Body = append(f.Body, ir.Instr{Op: ir.OpConst, Dst: dst, Imm: imm, Elem: elemTyp, Sym: expr})
		return dst, nil
	}
	// char literal 'x' or '\n' or '\0'
	if len(expr) >= 3 && expr[0] == '\'' && expr[len(expr)-1] == '\'' {
		inner := expr[1 : len(expr)-1]
		var n int64
		switch {
		case inner == `\0`:
			n = 0
		case inner == `\n`:
			n = '\n'
		case inner == `\t`:
			n = '\t'
		case inner == `\r`:
			n = '\r'
		case inner == `\\`:
			n = '\\'
		case inner == `\'`:
			n = '\''
		case len(inner) == 1:
			n = int64(inner[0])
		default:
			return ir.NoVal, &Error{Code: ErrParse, Message: "char: " + expr}
		}
		dst := f.Alloc()
		ensureScratch(f)
		f.Body = append(f.Body, ir.Instr{Op: ir.OpConst, Dst: dst, Imm: n})
		return dst, nil
	}
	if identOnly(expr) {
		dst := f.Alloc()
		ensureScratch(f)
		f.Body = append(f.Body, ir.Instr{Op: ir.OpMov, Dst: dst, Sym: "fn:" + expr})
		return dst, nil
	}
	return ir.NoVal, &Error{Code: ErrParse, Message: "expr: " + expr}
}

// ensureScratch uses Body as expression scratch; drainScratch moves to parent stmt list.
func ensureScratch(f *ir.Func) {}

// drainExpr pulls scratch Body instrs generated during parseExpr into a stmt slice.
func drainExpr(f *ir.Func) []ir.Instr {
	out := f.Body
	f.Body = nil
	return out
}

// wrap parseBlock to drain expr scratch into instructions before each stmt
// Actually parseExpr appends to f.Body — parseSimpleStmt and friends must prepend drain.

func indexCaseColon(s string) int {
	inChar := false
	inString := false
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && (inChar || inString) {
			i++
			continue
		}
		if s[i] == '\'' && !inString {
			inChar = !inChar
			continue
		}
		if s[i] == '"' && !inChar {
			inString = !inString
			continue
		}
		if s[i] == ':' && !inChar && !inString {
			return i
		}
	}
	return -1
}

func parseIntLit(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'") && len(s) >= 2 {
		inner := s[1 : len(s)-1]
		if inner == `\"` || inner == `"` {
			return int64('"'), nil
		}
		if inner == `\'` || inner == `'` {
			return int64('\''), nil
		}
		if inner == `\\` {
			return int64('\\'), nil
		}
		if inner == `\n` {
			return int64('\n'), nil
		}
		if inner == `\r` {
			return int64('\r'), nil
		}
		if inner == `\t` {
			return int64('\t'), nil
		}
		if inner == `\b` {
			return int64('\b'), nil
		}
		if inner == `\f` {
			return int64('\f'), nil
		}
		if inner == `\0` {
			return 0, nil
		}
		if len(inner) == 1 {
			return int64(inner[0]), nil
		}
		r, _, _, err := strconv.UnquoteChar(inner, '\'')
		if err == nil {
			return int64(r), nil
		}
	}
	s = strings.TrimRight(s, "uUlL")
	s = strings.TrimRight(s, "uUlL")
	// uint64 bit-pattern (FNV offset etc. exceed int64 max)
	if u, err := strconv.ParseUint(s, 0, 64); err == nil {
		return int64(u), nil
	}
	if n, err := strconv.ParseInt(s, 0, 64); err == nil {
		return n, nil
	}
	if val, ok := evalIntExpr(s); ok {
		return val, nil
	}
	return 0, fmt.Errorf("invalid int lit: %s", s)
}

func binOp(op string) ir.Op {
	switch op {
	case "|":
		return ir.OpOr
	case "^":
		return ir.OpXor
	case "&":
		return ir.OpAnd
	case "<<":
		return ir.OpShl
	case ">>":
		return ir.OpShr
	case "+":
		return ir.OpAdd
	case "-":
		return ir.OpSub
	case "*":
		return ir.OpMul
	case "/":
		return ir.OpDiv
	case "%":
		return ir.OpMod
	default:
		return ir.OpNop
	}
}

func findOp(expr string, ops []string) (int, string) {
	depth := 0
	for i := len(expr) - 1; i >= 0; i-- {
		c := expr[i]
		if c == '\'' {
			sl := 0
			for k := i - 1; k >= 0 && expr[k] == '\\'; k-- {
				sl++
			}
			if sl%2 == 1 {
				continue
			}
			j := i - 1
			for j >= 0 {
				if expr[j] == '\'' {
					jsl := 0
					for k := j - 1; k >= 0 && expr[k] == '\\'; k-- {
						jsl++
					}
					if jsl%2 == 0 {
						break
					}
				}
				j--
			}
			if j >= 0 {
				i = j
				continue
			}
		}
		if c == '"' {
			sl := 0
			for k := i - 1; k >= 0 && expr[k] == '\\'; k-- {
				sl++
			}
			if sl%2 == 1 {
				continue
			}
			j := i - 1
			for j >= 0 {
				if expr[j] == '"' {
					jsl := 0
					for k := j - 1; k >= 0 && expr[k] == '\\'; k-- {
						jsl++
					}
					if jsl%2 == 0 {
						break
					}
				}
				j--
			}
			if j >= 0 {
				i = j
				continue
			}
		}
		if c == ')' || c == ']' {
			depth++
			continue
		}
		if c == '(' || c == '[' {
			depth--
			continue
		}
		if depth != 0 {
			continue
		}
		for _, op := range ops {
			lo := len(op)
			if i+1 >= lo && expr[i-lo+1:i+1] == op {
				start := i - lo + 1
				if start == 0 {
					continue
				}
				// Unary +/- preceded by an operator or open bracket/paren (e.g. + -x or * -x or (-x))
				if (op == "-" || op == "+") && start > 0 {
					if expr[start-1] == 'e' || expr[start-1] == 'E' {
						if start > 1 && ((expr[start-2] >= '0' && expr[start-2] <= '9') || expr[start-2] == '.') {
							continue
						}
					}
					j := start - 1
					for j >= 0 && (expr[j] == ' ' || expr[j] == '\t') {
						j--
					}
					if j >= 0 && (expr[j] == '+' || expr[j] == '-' || expr[j] == '*' || expr[j] == '/' || expr[j] == '%' ||
						expr[j] == '&' || expr[j] == '|' || expr[j] == '^' || expr[j] == '<' || expr[j] == '>' ||
						expr[j] == '=' || expr[j] == ',' || expr[j] == '?' || expr[j] == ':' || expr[j] == '(' || expr[j] == '[') {
						continue
					}
					if j >= 0 && expr[j] == ')' {
						depthP := 0
						k := j
						for ; k >= 0; k-- {
							if expr[k] == ')' {
								depthP++
							} else if expr[k] == '(' {
								depthP--
								if depthP == 0 {
									break
								}
							}
						}
						if k >= 0 {
							inner := strings.TrimSpace(expr[k+1 : j])
							inner = strings.TrimSpace(strings.TrimSuffix(inner, "*"))
							if isTypeToken(inner) {
								continue
							}
						}
					}
				}
				// don't match - or > inside ->
				if op == "-" && start+1 < len(expr) && expr[start+1] == '>' {
					continue
				}
				if op == ">" && start > 0 && expr[start-1] == '-' {
					continue
				}
				// don't match < inside << or <= ; > inside >> or >=
				if op == "<" && start+1 < len(expr) && (expr[start+1] == '<' || expr[start+1] == '=') {
					continue
				}
				if op == ">" && start+1 < len(expr) && (expr[start+1] == '>' || expr[start+1] == '=') {
					continue
				}
				if op == "<" && start > 0 && expr[start-1] == '<' {
					continue
				}
				if op == ">" && start > 0 && expr[start-1] == '>' {
					continue
				}
				// don't match | inside || or |=
				if op == "|" && start+1 < len(expr) && (expr[start+1] == '|' || expr[start+1] == '=') {
					continue
				}
				if op == "|" && start > 0 && expr[start-1] == '|' {
					continue
				}
				if op == "&" && start+1 < len(expr) && (expr[start+1] == '&' || expr[start+1] == '=') {
					continue
				}
				if op == "&" && start > 0 && expr[start-1] == '&' {
					continue
				}
				// don't match + inside ++ or += ; - inside -- or -=
				if op == "+" && start+1 < len(expr) && (expr[start+1] == '+' || expr[start+1] == '=') {
					continue
				}
				if op == "+" && start > 0 && expr[start-1] == '+' {
					continue
				}
				if op == "-" && start+1 < len(expr) && (expr[start+1] == '-' || expr[start+1] == '=') {
					continue
				}
				if op == "-" && start > 0 && expr[start-1] == '-' {
					continue
				}
				return start, op
			}
		}
	}
	return -1, ""
}

func balanced(s string) bool {
	d := 0
	for _, c := range s {
		if c == '(' {
			d++
		} else if c == ')' {
			d--
			if d < 0 {
				return false
			}
		}
	}
	return d == 0
}

func isPtrOffsetBase(s string) bool {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		sub := strings.TrimSpace(s[1 : len(s)-1])
		if balanced(sub) {
			s = sub
		}
	}
	idx := strings.IndexAny(s, "+-")
	if idx <= 0 {
		return false
	}
	left := strings.TrimSpace(s[:idx])
	return identOnly(left)
}

func splitCSV(s string) []string {
	var out []string
	var cur strings.Builder
	depth := 0
	inChar := false
	inString := false
	for i := 0; i < len(s); i++ {
		r := s[i]
		if r == '\\' && (inChar || inString) {
			cur.WriteByte(r)
			if i+1 < len(s) {
				i++
				cur.WriteByte(s[i])
			}
			continue
		}
		if r == '\'' && !inString {
			inChar = !inChar
			cur.WriteByte(r)
			continue
		}
		if r == '"' && !inChar {
			inString = !inString
			cur.WriteByte(r)
			continue
		}
		if inChar || inString {
			cur.WriteByte(r)
			continue
		}
		switch r {
		case '(', '[', '{':
			depth++
			cur.WriteByte(r)
		case ')', ']', '}':
			depth--
			cur.WriteByte(r)
		case ',':
			if depth == 0 {
				out = append(out, cur.String())
				cur.Reset()
			} else {
				cur.WriteByte(r)
			}
		default:
			cur.WriteByte(r)
		}
	}
	if t := strings.TrimSpace(cur.String()); t != "" {
		out = append(out, t)
	}
	return out
}

// bodyBumpsPtr reports pointer-cursor advances in body: name +=/-=, name++/--, ++/--name.
func bodyBumpsPtr(body, name string) bool {
	n := regexp.QuoteMeta(name)
	re := regexp.MustCompile(`\b` + n + `\s*(\+\=|\-\=|\+\+|\-\-)`)
	if re.MatchString(body) {
		return true
	}
	rePref := regexp.MustCompile(`(\+\+|\-\-)\s*` + n + `\b`)
	return rePref.MatchString(body)
}

// splitTrailingIndex splits expr of form base[idx] → (base, idx, true).
func splitTrailingIndex(expr string) (base, idx string, ok bool) {
	expr = strings.TrimSpace(expr)
	if len(expr) == 0 || expr[len(expr)-1] != ']' {
		return "", "", false
	}
	depth := 0
	for k := len(expr) - 1; k >= 0; k-- {
		if expr[k] == ']' {
			depth++
		} else if expr[k] == '[' {
			depth--
			if depth == 0 {
				b := strings.TrimSpace(expr[:k])
				if !identOnly(b) {
					return "", "", false
				}
				return b, expr[k+1 : len(expr)-1], true
			}
		}
	}
	return "", "", false
}

func isSimpleBase(s string) bool {
	s = strings.TrimSpace(s)
	// ((TYPE*)p) or (TYPE*)p — indexable cast base
	if isPtrCastBase(s) {
		return true
	}
	s = strings.Trim(s, "()")
	if identOnly(s) {
		return true
	}
	parts := strings.Split(s, "->")
	if len(parts) == 2 && identOnly(strings.TrimSpace(parts[0])) && identOnly(strings.TrimSpace(parts[1])) {
		return true
	}
	partsDot := strings.Split(s, ".")
	if len(partsDot) == 2 && identOnly(strings.TrimSpace(partsDot[0])) && identOnly(strings.TrimSpace(partsDot[1])) {
		return true
	}
	return false
}

// isPtrCastBase reports (TYPE*)ident or ((TYPE*)ident).
func isPtrCastBase(s string) bool {
	s = strings.TrimSpace(s)
	for strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") && balanced(s[1:len(s)-1]) {
		inner := strings.TrimSpace(s[1 : len(s)-1])
		// (TYPE*)ident
		re := regexp.MustCompile(`^\(\s*([A-Za-z_][A-Za-z0-9_]*)\s*\*\s*\)\s*([A-Za-z_][A-Za-z0-9_]*)$`)
		if m := re.FindStringSubmatch(inner); m != nil && isTypeToken(m[1]) {
			return true
		}
		if re2 := regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*\*\s*\)\s*([A-Za-z_][A-Za-z0-9_]*)$`); false {
			_ = re2
		}
		// TYPE*)ident without extra wrap already handled; try full s
		s = inner
		if re.MatchString(s) {
			return true
		}
		reBare := regexp.MustCompile(`^\(\s*([A-Za-z_][A-Za-z0-9_]*)\s*\*\s*\)\s*([A-Za-z_][A-Za-z0-9_]*)$`)
		if m := reBare.FindStringSubmatch(s); m != nil && isTypeToken(m[1]) {
			return true
		}
		break
	}
	reBare := regexp.MustCompile(`^\(\s*([A-Za-z_][A-Za-z0-9_]*)\s*\*\s*\)\s*([A-Za-z_][A-Za-z0-9_]*)$`)
	if m := reBare.FindStringSubmatch(s); m != nil && isTypeToken(m[1]) {
		return true
	}
	return false
}

func identOnly(s string) bool {
	return regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`).MatchString(strings.TrimSpace(s))
}

func isFieldChain(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	s = strings.ReplaceAll(s, "->", ".")
	parts := strings.Split(s, ".")
	if len(parts) < 2 {
		return false
	}
	for _, p := range parts {
		if !identOnly(strings.TrimSpace(p)) {
			return false
		}
	}
	return true
}

// tryArrayInitDecl consumes brace-initialized arrays. Returns bytes consumed + stmts.
func tryArrayInitDecl(f *ir.Func, regs map[string]regInfo, rest string) (int, []ir.Stmt, error) {
	// Map all indices into `rest` (not a stripped copy) so end consumption is exact.
	lead := len(rest) - len(strings.TrimLeft(rest, " \t\n\r"))
	s := rest[lead:]
	wasStaticConst := false
	pos := 0 // offset within s after keyword stripping
	for {
		ss := s[pos:]
		if strings.HasPrefix(ss, "static") && (len(ss) == 6 || isSpace(ss[6])) {
			wasStaticConst = true
			pos += 6
			for pos < len(s) && isSpace(s[pos]) {
				pos++
			}
			continue
		}
		if strings.HasPrefix(ss, "const") && (len(ss) == 5 || isSpace(ss[5])) {
			wasStaticConst = true
			pos += 5
			for pos < len(s) && isSpace(s[pos]) {
				pos++
			}
			continue
		}
		break
	}
	body := s[pos:]
	reHeadStruct := regexp.MustCompile(`^(?:struct\s+)?([A-Za-z_][A-Za-z0-9_]*)\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*\{`)
	if mStruct := reHeadStruct.FindStringSubmatch(body); mStruct != nil && isStructType(mStruct[1], structEnv) {
		sname := mStruct[1]
		name := mStruct[2]
		openRel := strings.Index(body, "{")
		closeRel, err := matchBraceFrom(body, openRel)
		if err != nil {
			return 0, nil, &Error{Code: ErrParse, Message: "struct init brace: " + name}
		}
		end := lead + pos + closeRel + 1
		for end < len(rest) && rest[end] != ';' {
			end++
		}
		if end >= len(rest) || rest[end] != ';' {
			return 0, nil, &Error{Code: ErrParse, Message: "struct init ;"}
		}
		end++
		initBody := body[openRel+1 : closeRel]
		initCSV := flattenBraceInit(initBody)
		r := f.Alloc()
		sym := "struct:" + sname
		if initCSV != "" {
			sym = "struct_init:" + sname + ":" + name + ":" + initCSV
		}
		regs[name] = regInfo{v: r, typ: ir.TypeName(sname), ptr: true, scale: 1, elemIndex: false, structName: sname}
		notePtr(r, ir.TypeName(sname), 1, ir.NoVal)
		pm := ptrMeta[r]
		pm.structName = sname
		ptrMeta[r] = pm
		st := ir.Stmt{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpAlloca, Dst: r, Imm: 1, Elem: ir.TypeName(sname), Sym: sym}}
		return end, []ir.Stmt{st}, nil
	}

	reHead := regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s+([A-Za-z_][A-Za-z0-9_]*)\s*((?:\[\s*\d+\s*\])+)\s*=\s*\{`)
	m := reHead.FindStringSubmatch(body)
	if m == nil || !isTypeToken(m[1]) {
		return 0, nil, nil
	}
	typ := mapType(m[1])
	name := m[2]
	dims := regexp.MustCompile(`\[\s*(\d+)\s*\]`).FindAllStringSubmatch(m[3], -1)
	if len(dims) == 0 {
		return 0, nil, nil
	}
	openRel := strings.Index(body, "{")
	if openRel < 0 {
		return 0, nil, nil
	}
	closeRel, err := matchBraceFrom(body, openRel)
	if err != nil {
		return 0, nil, &Error{Code: ErrParse, Message: "array init brace: " + name}
	}
	if !strings.HasPrefix(strings.TrimSpace(body[closeRel+1:]), ";") {
		return 0, nil, nil
	}
	// Absolute end in rest: lead + pos + closeRel + 1, then skip to and past ';'
	end := lead + pos + closeRel + 1
	for end < len(rest) && rest[end] != ';' {
		end++
	}
	if end >= len(rest) || rest[end] != ';' {
		return 0, nil, &Error{Code: ErrParse, Message: "array init ;"}
	}
	end++

	initBody := body[openRel+1 : closeRel]
	total := 1
	var rows, cols int
	for _, d := range dims {
		n, _ := strconv.Atoi(d[1])
		total *= n
	}
	if len(dims) == 2 {
		rows, _ = strconv.Atoi(dims[0][1])
		cols, _ = strconv.Atoi(dims[1][1])
	} else {
		cols, _ = strconv.Atoi(dims[0][1])
		rows = 1
	}

	// Hoist multi-dim / large const tables as module globals (blake2b sigma).
	// Also hoist ALL function-static const with non-zero init (Barrett r[9] is small but must keep values).
	// JAMAIS un local mutable zéro-initialisé : en C il est re-zéroté à CHAQUE
	// entrée ; promu en globale, il garde les résidus de l'appel précédent
	// (bug de condensat keyed Blake2b démontré par oracle le 2026-08-15 :
	// uint8_t key_block[128] = {0} hoisté → clé courte polluée par la clé
	// longue antérieure) — en plus de la non-réentrance.
	hoist := len(dims) >= 2 ||
		(looksLikeConstInit(initBody) && !isZeroInit(initBody)) ||
		wasStaticConst
	if hoist {
		gname := f.Name + "_" + name
		// Prefer bare name when it matches a package-level harvest (mod_l's r).
		if name != "" {
			gname = name
		}
		g := ir.Global{Name: gname, Type: typ, Cols: cols, Rows: rows}
		if isZeroInit(initBody) {
			g.ZeroLen = total
		} else {
			csv := flattenBraceInit(initBody)
			var parts []string
			for _, p := range strings.Split(csv, ",") {
				if s := strings.TrimSpace(p); s != "" {
					parts = append(parts, s)
				}
			}
			if total > 0 && len(parts) < total {
				for len(parts) < total {
					parts = append(parts, "0")
				}
			}
			g.InitCSV = strings.Join(parts, ", ")
		}
		pendingGlobals = append(pendingGlobals, g)
		r := f.Alloc()
		regs[name] = regInfo{v: r, typ: typ, ptr: true, scale: 1, elemIndex: true, localArr: total, cols: cols}
		hoistedLocalGlobal[r] = gname
		notePtr(r, typ, 1, ir.NoVal)
		pm := ptrMeta[r]
		pm.elemIndex = true
		pm.localArr = total
		ptrMeta[r] = pm
		// Address of global: OpGlobal if available — emit via Alloca placeholder + Sym
		st := ir.Stmt{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpMov, Dst: r, Sym: "global:" + gname}}
		return end, []ir.Stmt{st}, nil
	}

	// 1D small local (mutable / non-zero or zero-init)
	sym := ""
	if !isZeroInit(initBody) {
		sym = "init:" + name + ":" + flattenBraceInit(initBody)
	}
	r := f.Alloc()
	regs[name] = regInfo{v: r, typ: typ, ptr: true, scale: 1, elemIndex: true, localArr: total}
	notePtr(r, typ, 1, ir.NoVal)
	pm := ptrMeta[r]
	pm.elemIndex = true
	pm.localArr = total
	ptrMeta[r] = pm
	st := ir.Stmt{Kind: ir.SKInstr, Ins: ir.Instr{Op: ir.OpAlloca, Dst: r, Imm: int64(total), Elem: typ, Sym: sym}}
	return end, []ir.Stmt{st}, nil
}

func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

func isIdentChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

func matchBraceFrom(s string, open int) (int, error) {
	if open >= len(s) || s[open] != '{' {
		return -1, fmt.Errorf("not brace")
	}
	depth := 0
	inChar := false
	inString := false
	for i := open; i < len(s); i++ {
		if s[i] == '\\' && (inChar || inString) {
			i++
			continue
		}
		if s[i] == '\'' && !inString {
			inChar = !inChar
			continue
		}
		if s[i] == '"' && !inChar {
			inString = !inString
			continue
		}
		if inChar || inString {
			continue
		}
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i, nil
			}
		}
	}
	return -1, fmt.Errorf("unbalanced")
}

func looksLikeConstInit(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || s == "0" {
		return true
	}
	for _, r := range s {
		if (r >= '0' && r <= '9') || r == 'x' || r == 'X' || r == 'a' || r == 'b' || r == 'c' || r == 'd' || r == 'e' || r == 'f' ||
			r == 'A' || r == 'B' || r == 'C' || r == 'D' || r == 'E' || r == 'F' ||
			r == ',' || r == '{' || r == '}' || r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == 'u' || r == 'U' || r == 'l' || r == 'L' {
			continue
		}
		return false
	}
	return true
}

func isZeroInit(s string) bool {
	s = strings.TrimSpace(s)
	return s == "0" || s == ""
}

func flattenBraceInit(s string) string {
	parts := splitCSV(strings.TrimSpace(s))
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.TrimPrefix(p, "{")
		p = strings.TrimSuffix(p, "}")
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if strings.HasPrefix(p, "\"") && strings.HasSuffix(p, "\"") {
			str, err := strconv.Unquote(p)
			if err == nil {
				for i := 0; i < len(str); i++ {
					out = append(out, fmt.Sprintf("0x%02x", str[i]))
				}
				out = append(out, "0x00")
				continue
			}
		}
		if strings.HasPrefix(p, "'") && strings.HasSuffix(p, "'") {
			out = append(out, p)
			continue
		}
		out = append(out, p)
	}
	return strings.Join(out, ", ")
}

// pending side channels for array-init decls (cleared per function)
var pendingGlobals []ir.Global
var pendingAlloca []pendingAllocaItem
var hoistedLocalGlobal = map[ir.Value]string{}

type pendingAllocaItem struct {
	reg  ir.Value
	n    int
	typ  ir.TypeName
	name string
	init string
}

func normalizeDeclStmt(stClean string) string {
	stClean = strings.TrimSpace(stClean)
	for {
		if strings.HasPrefix(stClean, "static ") {
			stClean = strings.TrimSpace(stClean[7:])
		} else if strings.HasPrefix(stClean, "const ") {
			stClean = strings.TrimSpace(stClean[6:])
		} else if strings.HasPrefix(stClean, "volatile ") {
			stClean = strings.TrimSpace(stClean[9:])
		} else if strings.HasPrefix(stClean, "register ") {
			stClean = strings.TrimSpace(stClean[9:])
		} else if strings.HasPrefix(stClean, "inline ") {
			stClean = strings.TrimSpace(stClean[7:])
		} else if strings.HasPrefix(stClean, "unsigned char ") {
			stClean = "uint8_t " + strings.TrimSpace(stClean[14:])
		} else if strings.HasPrefix(stClean, "unsigned short ") {
			stClean = "uint16_t " + strings.TrimSpace(stClean[15:])
		} else if strings.HasPrefix(stClean, "unsigned int ") {
			stClean = "uint32_t " + strings.TrimSpace(stClean[13:])
		} else if strings.HasPrefix(stClean, "unsigned long long ") {
			stClean = "uint64_t " + strings.TrimSpace(stClean[19:])
		} else if strings.HasPrefix(stClean, "unsigned long ") {
			stClean = "uint64_t " + strings.TrimSpace(stClean[14:])
		} else if strings.HasPrefix(stClean, "signed char ") {
			stClean = "int8_t " + strings.TrimSpace(stClean[12:])
		} else if strings.HasPrefix(stClean, "signed short ") {
			stClean = "int16_t " + strings.TrimSpace(stClean[13:])
		} else if strings.HasPrefix(stClean, "signed int ") {
			stClean = "int32_t " + strings.TrimSpace(stClean[11:])
		} else if strings.HasPrefix(stClean, "long long ") {
			stClean = "int64_t " + strings.TrimSpace(stClean[10:])
		} else if strings.HasPrefix(stClean, "unsigned __int64 ") || strings.HasPrefix(stClean, "unsigned __int64\t") || stClean == "unsigned __int64" {
			stClean = "uint64_t " + strings.TrimSpace(stClean[16:])
		} else if strings.HasPrefix(stClean, "__uint64_t ") || stClean == "__uint64_t" {
			stClean = "uint64_t " + strings.TrimSpace(stClean[11:])
		} else if strings.HasPrefix(stClean, "__int64 ") || strings.HasPrefix(stClean, "__int64\t") || stClean == "__int64" {
			stClean = "int64_t " + strings.TrimSpace(stClean[8:])
		} else if strings.HasPrefix(stClean, "unsigned __int128 ") || strings.HasPrefix(stClean, "unsigned __int128\t") || stClean == "unsigned __int128" {
			stClean = "uint128_t " + strings.TrimSpace(stClean[17:])
		} else if strings.HasPrefix(stClean, "__uint128_t ") || stClean == "__uint128_t" {
			stClean = "uint128_t " + strings.TrimSpace(stClean[11:])
		} else if strings.HasPrefix(stClean, "__int128 ") || strings.HasPrefix(stClean, "__int128\t") || stClean == "__int128" {
			stClean = "int128_t " + strings.TrimSpace(stClean[8:])
		} else if strings.HasPrefix(stClean, "unsigned ") {
			stClean = "uint32_t " + strings.TrimSpace(stClean[9:])
		} else if strings.HasPrefix(stClean, "signed ") {
			stClean = "int32_t " + strings.TrimSpace(stClean[7:])
		} else {
			break
		}
	}
	// Separate attached pointer asterisk on type: "uint8_t* p" -> "uint8_t * p"
	if idx := strings.IndexByte(stClean, '*'); idx > 0 {
		before := strings.TrimSpace(stClean[:idx])
		if isTypeToken(before) {
			rest := strings.TrimSpace(stClean[idx+1:])
			if strings.HasPrefix(rest, "const ") {
				rest = strings.TrimSpace(rest[6:])
			}
			stClean = before + " * " + rest
		}
	}
	return stClean
}

func isTypeToken(s string) bool {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "const")
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "struct")
	s = strings.TrimSpace(s)
	switch s {
	case "void", "int", "char", "short", "long", "fe", "bool", "_Bool",
		"int8_t", "int16_t", "int32_t", "int64_t",
		"uint8_t", "uint16_t", "uint32_t", "uint64_t",
		"uint32", "uint64", "size_t", "uintptr_t", "u8", "u16", "u32", "u64",
		"i8", "i16", "i32", "i64",
		"uint128_t", "int128_t", "u128", "i128", "uint128", "int128", "__uint128_t", "__int128_t", "__int128",
		"nk_f64_t", "nk_f32_t", "nk_i32_t", "nk_u32_t", "nk_i64_t", "nk_u64_t", "nk_size_t",
		"nk_f16_t", "nk_bf16_t", "nk_i8_t", "nk_u8_t",
		"unsigned", "signed",
		"float", "double", "float32", "float64",
		"pt2Function", "ptr_lookup_fn", "stoken_t", "sfilter":
		return true
	default:
		if isStructType(s, structEnv) {
			return true
		}
		if typedefEnv != nil {
			if _, ok := typedefEnv[s]; ok {
				return true
			}
		}
		return false
	}
}

func intScalarReg(r regInfo) bool {
	if !r.elemIndex && (r.typ == ir.TypInt || r.typ == ir.TypInt32 || r.typ == ir.TypInt64) {
		return true
	}
	return false
}

func mapType(s string) ir.TypeName {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "static ")
	s = strings.TrimPrefix(s, "inline ")
	s = strings.TrimPrefix(s, "__inline__ ")
	s = strings.TrimPrefix(s, "__inline ")
	if !strings.Contains(s, "*") {
		s = strings.TrimPrefix(s, "const ")
		s = strings.TrimPrefix(s, "volatile ")
	}
	s = strings.TrimSpace(s)
	switch s {
	case "bool", "_Bool":
		return ir.TypBool
	case "const char *", "const char*", "const char * const",
		"char *", "char*", "unsigned char *", "unsigned char*", "const unsigned char *", "const unsigned char*", "uint8_t *", "uint8_t*", "u8 *", "u8*", "const u8 *", "const u8*":
		return ir.TypeName("[]byte")
	case "void *", "void*", "const void *", "const void*":
		return ir.TypeName("any")
	case "int", "int32_t", "i32", "nk_i32_t", "signed int", "cJSON_bool":
		return ir.TypInt
	case "int8_t", "i8", "nk_i8_t", "signed char":
		return ir.TypInt8
	case "char", "uint8_t", "u8", "nk_u8_t", "unsigned char":
		return ir.TypUint8
	case "int16_t", "i16", "short", "signed short":
		return ir.TypInt16
	case "uint16_t", "u16", "unsigned short":
		return ir.TypUint16
	case "uint32_t", "uint32", "u32", "nk_u32_t", "unsigned int", "unsigned":
		return ir.TypUint32
	case "uint64_t", "uint64", "u64", "size_t", "usize", "uintptr_t", "nk_u64_t", "nk_size_t", "unsigned long", "unsigned long long":
		return ir.TypUint64
	case "int64_t", "i64", "nk_i64_t", "long long", "long":
		return ir.TypInt64
	case "uint128_t", "uint128", "u128", "unsigned __int128", "__uint128_t", "int128_t", "int128", "i128", "__int128", "__int128_t":
		return ir.TypUint128
	case "float", "float32", "nk_f32_t", "nk_f16_t", "nk_bf16_t":
		return ir.TypFloat32
	case "double", "float64", "nk_f64_t":
		return ir.TypFloat64
	case "utf8proc_property_t", "utf8proc_property_t *", "utf8proc_property_t*", "const utf8proc_property_t*", "const utf8proc_property_t *",
		"utf8proc_property_struct", "utf8proc_property_struct *", "utf8proc_property_struct*":
		return ir.TypeName("*Utf8proc_property_t")
	case "utf8proc_bool":
		return ir.TypBool
	case "utf8proc_int8_t":
		return ir.TypInt8
	case "utf8proc_uint8_t":
		return ir.TypUint8
	case "utf8proc_int16_t", "utf8proc_propval_t":
		return ir.TypInt16
	case "utf8proc_uint16_t":
		return ir.TypUint32
	case "utf8proc_int32_t":
		return ir.TypInt32
	case "utf8proc_uint32_t":
		return ir.TypUint32
	case "utf8proc_ssize_t", "utf8proc_size_t":
		return ir.TypUint64
	case "utf8proc_category_t", "utf8proc_bidi_class_t", "utf8proc_decomp_type_t", "utf8proc_boundclass_t":
		return ir.TypInt
	case "void":
		return ir.TypVoid
	case "pt2Function", "ptr_lookup_fn":
		return ir.TypeName(s)
	default:
		if isStructType(s, structEnv) {
			return ir.TypeName(s)
		}
		return ir.TypInt
	}
}

func baseName(path string) string {
	i := strings.LastIndexAny(path, `/\`)
	if i >= 0 {
		path = path[i+1:]
	}
	if j := strings.LastIndexByte(path, '.'); j > 0 {
		path = path[:j]
	}
	return path
}
