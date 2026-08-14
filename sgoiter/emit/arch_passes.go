package emit

import (
	"regexp"
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

// archInlineRotlWrappers removes tiny Rotl32 helpers and inlines bits.RotateLeft32.
var reRotlDef = regexp.MustCompile(`(?s)func Rotl32\(x uint32, r uint8\) uint32 \{\s*return uint32\(bits\.RotateLeft32\(x, int\(r\)\)\)\s*\}\n*`)
var reRotlCall = regexp.MustCompile(`Rotl32\(([^,]+),\s*uint8\((\d+)\)\)`)
var reRotlCall2 = regexp.MustCompile(`Rotl32\(([^,]+),\s*(\d+)\)`)

func archInlineRotlWrappers(src string) string {
	if !strings.Contains(src, "Rotl32") {
		return src
	}
	src = reRotlDef.ReplaceAllString(src, "")
	src = reRotlCall.ReplaceAllString(src, `bits.RotateLeft32($1, $2)`)
	src = reRotlCall2.ReplaceAllString(src, `bits.RotateLeft32($1, $2)`)
	return src
}

var reRotr64Def = regexp.MustCompile(`(?s)func Rotr64\(x uint64, n uint64\) uint64 \{\s*return uint64\(bits\.RotateLeft64\(x, 64-int\(n\)\)\)\s*\}\n*`)
var reRotr64Call = regexp.MustCompile(`Rotr64\(([^,]+),\s*uint64\((\d+)\)\)`)
var reRotr64Call2 = regexp.MustCompile(`Rotr64\(([^,]+),\s*(\d+)\)`)

func archInlineRotrWrappers(src string) string {
	if !strings.Contains(src, "Rotr64") {
		return src
	}
	src = reRotr64Def.ReplaceAllString(src, "")
	src = reRotr64Call.ReplaceAllString(src, `bits.RotateLeft64($1, 64-$2)`)
	src = reRotr64Call2.ReplaceAllString(src, `bits.RotateLeft64($1, 64-$2)`)
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

func archWipeToClear(src string) string {
	lines := strings.Split(src, "\n")
	out := make([]string, 0, len(lines))
	for _, ln := range lines {
		if m := reRangeZero.FindStringSubmatch(ln); m != nil && m[2] == m[3] {
			out = append(out, m[1]+"clear("+m[2]+")")
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
