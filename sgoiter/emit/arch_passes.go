package emit

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// archArrayNotSlice: `var _arr_vN [K]T` + `vN := _arr_vN[:]` → use array directly as vN.
// Pattern from emit of stack arrays lowered to slice headers.
var (
	reArrDecl  = regexp.MustCompile(`(?m)^(\t*)var (_arr_[A-Za-z0-9_]+) \[(\d+)\](\w+)\s*$`)
	reArrSlice = regexp.MustCompile(`(?m)^(\t*)([A-Za-z_][A-Za-z0-9_]*) := (_arr_[A-Za-z0-9_]+)\[:\]\s*$`)
)

func archArrayNotSlice(src string) string {
	// Map _arr_X -> user name when pair found; rewrite decl to user name array; drop slice line; replace _arr_ refs.
	lines := strings.Split(src, "\n")
	type pair struct{ arr, user, typ, n, pad string }
	var pairs []pair
	sliceAt := map[int]bool{}
	for i, ln := range lines {
		if m := reArrSlice.FindStringSubmatch(ln); m != nil {
			pairs = append(pairs, pair{arr: m[3], user: m[2], pad: m[1]})
			sliceAt[i] = true
		}
	}
	if len(pairs) == 0 {
		return src
	}
	byArr := map[string]pair{}
	for _, p := range pairs {
		byArr[p.arr] = p
	}
	var out []string
	for i, ln := range lines {
		if sliceAt[i] {
			continue
		}
		if m := reArrDecl.FindStringSubmatch(ln); m != nil {
			arr, n, typ := m[2], m[3], m[4]
			if p, ok := byArr[arr]; ok {
				out = append(out, m[1]+"var "+p.user+" ["+n+"]"+typ)
				continue
			}
		}
		// replace _arr_foo with user name in remaining lines
		for arr, p := range byArr {
			ln = strings.ReplaceAll(ln, arr, p.user)
		}
		out = append(out, ln)
	}
	return strings.Join(out, "\n")
}

// archInlineRotlWrappers removes tiny Rotl32/Rotl64 helpers and inlines bits.RotateLeft32/64.
var reRotlDef = regexp.MustCompile(`(?s)func Rotl32\(x uint32, r (?:uint8|int8|int32|uint32|int)\) uint32 \{\s*return (?:uint32\()?bits\.RotateLeft32\(x, int\(r\)\)\)?\s*\}\n*`)
var reRotlCall = regexp.MustCompile(`Rotl32\(([^,]+),\s*(?:u?int(?:8|16|32|64)?\((\d+)\)|(\d+))\)`)

var reRotl64Def = regexp.MustCompile(`(?s)func Rotl64\(x uint64, r (?:uint8|int8|int32|uint32|int|int8)\) uint64 \{\s*return (?:uint64\()?bits\.RotateLeft64\(x, int\(r\)\)\)?\s*\}\n*`)
var reRotl64Call = regexp.MustCompile(`Rotl64\(([^,]+),\s*(?:u?int(?:8|16|32|64)?\((\d+)\)|(\d+))\)`)

func archInlineRotlWrappers(src string) string {
	if strings.Contains(src, "Rotl32") {
		src = reRotlDef.ReplaceAllString(src, "")
		src = reRotlCall.ReplaceAllStringFunc(src, func(m string) string {
			sub := reRotlCall.FindStringSubmatch(m)
			if len(sub) > 0 {
				arg1 := sub[1]
				shift := sub[2]
				if shift == "" {
					shift = sub[3]
				}
				return fmt.Sprintf("bits.RotateLeft32(%s, %s)", arg1, shift)
			}
			return m
		})
	}
	if strings.Contains(src, "Rotl64") {
		src = reRotl64Def.ReplaceAllString(src, "")
		src = reRotl64Call.ReplaceAllStringFunc(src, func(m string) string {
			sub := reRotl64Call.FindStringSubmatch(m)
			if len(sub) > 0 {
				arg1 := sub[1]
				shift := sub[2]
				if shift == "" {
					shift = sub[3]
				}
				return fmt.Sprintf("bits.RotateLeft64(%s, %s)", arg1, shift)
			}
			return m
		})
	}
	return src
}

var (
	rePack8Off = regexp.MustCompile(`\({7}uint64\(([A-Za-z0-9_]+)\[([^\]\+]+)\]\)\s*\|\s*\(uint64\([A-Za-z0-9_]+\[[^\]\+]+\s*\+\s*1\]\)\s*<<\s*8\)\)\s*\|\s*\(uint64\([A-Za-z0-9_]+\[[^\]\+]+\s*\+\s*2\]\)\s*<<\s*16\)\)\s*\|\s*\(uint64\([A-Za-z0-9_]+\[[^\]\+]+\s*\+\s*3\]\)\s*<<\s*24\)\)\s*\|\s*\(uint64\([A-Za-z0-9_]+\[[^\]\+]+\s*\+\s*4\]\)\s*<<\s*32\)\)\s*\|\s*\(uint64\([A-Za-z0-9_]+\[[^\]\+]+\s*\+\s*5\]\)\s*<<\s*40\)\)\s*\|\s*\(uint64\([A-Za-z0-9_]+\[[^\]\+]+\s*\+\s*6\]\)\s*<<\s*48\)\)\s*\|\s*\(uint64\([A-Za-z0-9_]+\[[^\]\+]+\s*\+\s*7\]\)\s*<<\s*56\)\)`)
	rePack8_0  = regexp.MustCompile(`\({7}uint64\(([A-Za-z0-9_]+)\[0\]\)\s*\|\s*\(uint64\([A-Za-z0-9_]+\[1\]\)\s*<<\s*8\)\)\s*\|\s*\(uint64\([A-Za-z0-9_]+\[2\]\)\s*<<\s*16\)\)\s*\|\s*\(uint64\([A-Za-z0-9_]+\[3\]\)\s*<<\s*24\)\)\s*\|\s*\(uint64\([A-Za-z0-9_]+\[4\]\)\s*<<\s*32\)\)\s*\|\s*\(uint64\([A-Za-z0-9_]+\[5\]\)\s*<<\s*40\)\)\s*\|\s*\(uint64\([A-Za-z0-9_]+\[6\]\)\s*<<\s*48\)\)\s*\|\s*\(uint64\([A-Za-z0-9_]+\[7\]\)\s*<<\s*56\)\)`)

	rePack8OffNum = regexp.MustCompile(`\({7}uint64\(([A-Za-z0-9_]+)\[([^\]\+]+)\s*\+\s*(\d+)\]\)\s*\|\s*\(uint64\([A-Za-z0-9_]+\[[^\]\+]+\s*\+\s*(\d+)\]\)\s*<<\s*8\)\)\s*\|\s*\(uint64\([A-Za-z0-9_]+\[[^\]\+]+\s*\+\s*(\d+)\]\)\s*<<\s*16\)\)\s*\|\s*\(uint64\([A-Za-z0-9_]+\[[^\]\+]+\s*\+\s*(\d+)\]\)\s*<<\s*24\)\)\s*\|\s*\(uint64\([A-Za-z0-9_]+\[[^\]\+]+\s*\+\s*(\d+)\]\)\s*<<\s*32\)\)\s*\|\s*\(uint64\([A-Za-z0-9_]+\[[^\]\+]+\s*\+\s*(\d+)\]\)\s*<<\s*40\)\)\s*\|\s*\(uint64\([A-Za-z0-9_]+\[[^\]\+]+\s*\+\s*(\d+)\]\)\s*<<\s*48\)\)\s*\|\s*\(uint64\([A-Za-z0-9_]+\[[^\]\+]+\s*\+\s*(\d+)\]\)\s*<<\s*56\)\)`)
	rePack4OffNum = regexp.MustCompile(`\({3}uint32\(([A-Za-z0-9_]+)\[([^\]\+]+)\s*\+\s*(\d+)\]\)\s*\|\s*\(uint32\([A-Za-z0-9_]+\[[^\]\+]+\s*\+\s*(\d+)\]\)\s*<<\s*8\)\)\s*\|\s*\(uint32\([A-Za-z0-9_]+\[[^\]\+]+\s*\+\s*(\d+)\]\)\s*<<\s*16\)\)\s*\|\s*\(uint32\([A-Za-z0-9_]+\[[^\]\+]+\s*\+\s*(\d+)\]\)\s*<<\s*24\)\)`)

	rePack4Off = regexp.MustCompile(`\({3}uint32\(([A-Za-z0-9_]+)\[([^\]\+]+)\]\)\s*\|\s*\(uint32\([A-Za-z0-9_]+\[[^\]\+]+\s*\+\s*1\]\)\s*<<\s*8\)\)\s*\|\s*\(uint32\([A-Za-z0-9_]+\[[^\]\+]+\s*\+\s*2\]\)\s*<<\s*16\)\)\s*\|\s*\(uint32\([A-Za-z0-9_]+\[[^\]\+]+\s*\+\s*3\]\)\s*<<\s*24\)\)`)
	rePack4_0  = regexp.MustCompile(`\({3}uint32\(([A-Za-z0-9_]+)\[0\]\)\s*\|\s*\(uint32\([A-Za-z0-9_]+\[1\]\)\s*<<\s*8\)\)\s*\|\s*\(uint32\([A-Za-z0-9_]+\[2\]\)\s*<<\s*16\)\)\s*\|\s*\(uint32\([A-Za-z0-9_]+\[3\]\)\s*<<\s*24\)\)`)

	reRead32LeDef = regexp.MustCompile(`(?s)func (?:Read32_le|load32_le)\(p \[\]byte\) uint32 \{\s*return [^\n]+\n\}`)
	reRead64LeDef = regexp.MustCompile(`(?s)func (?:Read64_le|load64_le)\(p \[\]byte\) uint64 \{\s*(?:_\s*=\s*p\[7\]\s*)?return [^\n]+\n\}`)
)

func archFoldBytePacksToLittleEndian(src string) string {
	src = reRead32LeDef.ReplaceAllString(src, "func Read32_le(p []byte) uint32 {\n\treturn binary.LittleEndian.Uint32(p)\n}")
	src = reRead64LeDef.ReplaceAllString(src, "func Read64_le(p []byte) uint64 {\n\treturn binary.LittleEndian.Uint64(p)\n}")
	src = rePack8OffNum.ReplaceAllStringFunc(src, func(m string) string {
		sub := rePack8OffNum.FindStringSubmatch(m)
		if len(sub) >= 11 {
			startNum, _ := strconv.Atoi(sub[3])
			match := true
			for idx := 4; idx < 11; idx++ {
				n, _ := strconv.Atoi(sub[idx])
				if n != startNum+(idx-3) {
					match = false
					break
				}
			}
			if match {
				return fmt.Sprintf("binary.LittleEndian.Uint64(%s[%s+%d:])", sub[1], sub[2], startNum)
			}
		}
		return m
	})
	src = rePack8Off.ReplaceAllString(src, `binary.LittleEndian.Uint64($1[$2:])`)
	src = rePack8_0.ReplaceAllString(src, `binary.LittleEndian.Uint64($1)`)
	src = rePack4OffNum.ReplaceAllStringFunc(src, func(m string) string {
		sub := rePack4OffNum.FindStringSubmatch(m)
		if len(sub) >= 7 {
			startNum, _ := strconv.Atoi(sub[3])
			match := true
			for idx := 4; idx < 7; idx++ {
				n, _ := strconv.Atoi(sub[idx])
				if n != startNum+(idx-3) {
					match = false
					break
				}
			}
			if match {
				return fmt.Sprintf("binary.LittleEndian.Uint32(%s[%s+%d:])", sub[1], sub[2], startNum)
			}
		}
		return m
	})
	src = rePack4Off.ReplaceAllString(src, `binary.LittleEndian.Uint32($1[$2:])`)
	src = rePack4_0.ReplaceAllString(src, `binary.LittleEndian.Uint32($1)`)
	src = regexp.MustCompile(`(?:uint64|uint32)\(binary\.LittleEndian\.Uint(64|32)\(([^)]+)\)\)`).ReplaceAllString(src, `binary.LittleEndian.Uint$1($2)`)
	src = regexp.MustCompile(`(?:uint64|uint32)binary\.LittleEndian\.Uint(64|32)\(([^)]+)\)`).ReplaceAllString(src, `binary.LittleEndian.Uint$1($2)`)
	src = strings.ReplaceAll(src, "uint64binary.LittleEndian", "binary.LittleEndian")
	src = strings.ReplaceAll(src, "uint32binary.LittleEndian", "binary.LittleEndian")
	return src
}

var reRotr64Def = regexp.MustCompile(`(?s)func Rotr64\(x uint64, n (?:uint8|int8|int32|uint32|uint64|int)\) uint64 \{\s*return (?:uint64\()?bits\.RotateLeft64\(x, 64-int\(n\)\)\)?\s*\}\n*`)
var reRotr64Call = regexp.MustCompile(`Rotr64\(([^,]+),\s*(?:u?int(?:8|16|32|64)?\((\d+)\)|(\d+))\)`)

func archInlineRotrWrappers(src string) string {
	if !strings.Contains(src, "Rotr64") {
		return src
	}
	src = reRotr64Def.ReplaceAllString(src, "")
	src = reRotr64Call.ReplaceAllStringFunc(src, func(m string) string {
		sub := reRotr64Call.FindStringSubmatch(m)
		if len(sub) > 0 {
			arg1 := sub[1]
			shift := sub[2]
			if shift == "" {
				shift = sub[3]
			}
			return fmt.Sprintf("bits.RotateLeft64(%s, 64-%s)", arg1, shift)
		}
		return m
	})
	return src
}

func archDeduplicateStores(src string) string {
	lines := strings.Split(src, "\n")
	out := make([]string, 0, len(lines))
	var lastLine string
	for _, ln := range lines {
		trimmed := strings.TrimSpace(ln)
		if trimmed != "" && strings.Contains(trimmed, " = ") && !strings.Contains(trimmed, " + ") && !strings.Contains(trimmed, " - ") {
			if ln == lastLine {
				continue
			}
		}
		out = append(out, ln)
		lastLine = ln
	}
	return strings.Join(out, "\n")
}

// archWipeToClear converts range-zero wipe loops into clear(v) builtins.
var reRangeZero = regexp.MustCompile(`^(\t*)for _i := range ([A-Za-z0-9_.]+) \{\s*([A-Za-z0-9_.]+)\[_i\] = 0\s*\}$`)

var constGlobals = map[string]bool{
	"A": true, "A2": true, "fe_one": true, "sqrtm1": true, "d": true, "D2": true,
	"lop_x": true, "lop_y": true, "ufactor": true, "zero": true, "r": true,
	"base_point": true, "dirty_base_point": true, "Lm2": true, "m_inv": true,
	"k": true, "s": true, "t": true, "half_mod_L": true, "half_ones": true,
	"blake2b_compress_sigma": true, "chacha20_constant": true,
}

func archWipeToClear(src string) string {
	lines := strings.Split(src, "\n")
	out := make([]string, 0, len(lines))
	for _, ln := range lines {
		if m := reRangeZero.FindStringSubmatch(ln); m != nil && m[2] == m[3] {
			if constGlobals[m[2]] {
				continue
			}
			out = append(out, m[1]+"clear("+m[2]+"[:])")
			continue
		}
		out = append(out, ln)
	}
	return strings.Join(out, "\n")
}

// archOptimizeBlockCopies optimizes fixed loops for struct/array copy and wipe.
var reWipeLoop = regexp.MustCompile(`(?s)func Crypto_wipe\(secret \[\]byte, size uint64\) \{\s*var v4 uint64\s*v4 = 0\s*for v4 < size \{\s*secret\[(?:int\()?v4\)?\] = uint8\(0\)\s*(?:v4 = v4 \+ 1|v4\+\+|v4 \+= 1)\s*\}\s*\}`)
var reCopyBlock = regexp.MustCompile(`(?s)func Copy_block\(o \*Blk, in \*Blk\) \{\s*var v2 uint64\s*v2 = 0\s*for v2 < 0x80 \{\s*o\.A\[(?:int\()?v2\)?\] = in\.A\[(?:int\()?v2\)?\]\s*(?:v2 = v2 \+ 1|v2\+\+|v2 \+= 1)\s*\}\s*\}`)

func archOptimizeBlockCopies(src string) string {
	src = reWipeLoop.ReplaceAllString(src, "func Crypto_wipe(secret []byte, size uint64) {\n\tclear(secret[:size])\n}")
	src = reCopyBlock.ReplaceAllString(src, "func Copy_block(o *Blk, in *Blk) {\n\to.A = in.A\n}")
	return src
}

var (
	reLoad32Le  = regexp.MustCompile(`(?s)func Load32_le\(s \[\]byte\) uint32 \{\s*return [^\n]+\n\}`)
	reLoad64Le  = regexp.MustCompile(`(?s)func Load64_le\(s \[\]byte\) uint64 \{\s*return [^\n]+\n\}`)
	reStore32Le = regexp.MustCompile(`(?s)func Store32_le\(out \[\]byte, in uint32\) \{\s*out\[0\] = [^\n]+\s*out\[1\] = [^\n]+\s*out\[2\] = [^\n]+\s*out\[3\] = [^\n]+\s*\}`)
	reStore64Le = regexp.MustCompile(`(?s)func Store64_le\(out \[\]byte, in uint64\) \{\s*Store32_le\(out, uint32\(in\)\)\s*Store32_le\(out\[4:\], uint32\(in >> 32\)\)\s*\}`)
)

// archOptimizeEndianHelpers projette les helpers manuels vers les intrinsèques binary.LittleEndian.
func archOptimizeEndianHelpers(src string) string {
	src = reLoad32Le.ReplaceAllString(src, "func Load32_le(s []byte) uint32 {\n\treturn binary.LittleEndian.Uint32(s)\n}")
	src = reLoad64Le.ReplaceAllString(src, "func Load64_le(s []byte) uint64 {\n\treturn binary.LittleEndian.Uint64(s)\n}")
	src = reStore32Le.ReplaceAllString(src, "func Store32_le(out []byte, in uint32) {\n\tbinary.LittleEndian.PutUint32(out, in)\n}")
	src = reStore64Le.ReplaceAllString(src, "func Store64_le(out []byte, in uint64) {\n\tbinary.LittleEndian.PutUint64(out, in)\n}")
	return src
}

// archNormalizeNegInduction: for i := -n; i != 0; i++ with blocks[i] style — hard;
// light touch: document via comment only if pattern not safe to rewrite.
// Soft rewrite of `v18 = -v7` + `for v18 != 0` is too risky without IR.
func archNormalizeNegInduction(src string) string {
	return src
}

// archNormalize3ClauseLoops restaure la syntaxe Go 3-clauses `for i = 0; i < N; i++ { ... }`.
func archNormalize3ClauseLoops(src string) string {
	lines := strings.Split(src, "\n")
	var out []string
	i := 0
	for i < len(lines) {
		ln := lines[i]
		trim := strings.TrimSpace(ln)
		if (strings.Contains(trim, " = ") || strings.Contains(trim, " := ")) && !strings.HasPrefix(trim, "//") && !strings.HasPrefix(trim, "var ") {
			sep := " = "
			if strings.Contains(trim, " := ") {
				sep = " := "
			}
			parts := strings.SplitN(trim, sep, 2)
			varName := strings.TrimSpace(parts[0])
			initVal := strings.TrimSpace(parts[1])
			if isSimpleIdent(varName) && i+1 < len(lines) {
				nextLn := lines[i+1]
				nextTrim := strings.TrimSpace(nextLn)
				if strings.HasPrefix(nextTrim, "for "+varName+" ") && strings.HasSuffix(nextTrim, "{") {
					cond := strings.TrimSpace(nextTrim[len("for ") : len(nextTrim)-1])
					depth := 1
					j := i + 2
					var bodyLines []string
					for j < len(lines) && depth > 0 {
						l := lines[j]
						t := strings.TrimSpace(l)
						if strings.HasSuffix(t, "{") {
							depth++
						}
						if t == "}" || strings.HasPrefix(t, "}") {
							depth--
						}
						if depth > 0 {
							bodyLines = append(bodyLines, l)
						}
						j++
					}
					if depth == 0 && len(bodyLines) > 0 {
						lastBody := strings.TrimSpace(bodyLines[len(bodyLines)-1])
						post := ""
						if lastBody == varName+"++" {
							post = varName + "++"
						} else if strings.HasPrefix(lastBody, varName+" += ") {
							post = lastBody
						} else if strings.HasPrefix(lastBody, varName+" = "+varName+" + ") {
							step := lastBody[len(varName+" = "+varName+" + "):]
							post = varName + " += " + step
						}
						hasContinue := false
						for _, bl := range bodyLines {
							if strings.Contains(bl, "continue") {
								hasContinue = true
								break
							}
						}
						if post != "" && !hasContinue {
							pad := ln[:len(ln)-len(strings.TrimLeft(ln, " \t"))]
							inner := strings.Join(bodyLines[:len(bodyLines)-1], "\n")
							newLoop := fmt.Sprintf("%sfor %s %s %s; %s; %s {\n%s\n%s}", pad, varName, strings.TrimSpace(sep), initVal, cond, post, inner, pad)
							out = append(out, newLoop)
							i = j
							continue
						}
					}
				}
			}
		}
		out = append(out, ln)
		i++
	}
	return strings.Join(out, "\n")
}

