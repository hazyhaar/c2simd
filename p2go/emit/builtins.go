// Emit des builtins PHP (F-p2go-stdlib-*) : inline quand Go le permet
// (min/max natifs, chr), helper p2go* sinon — helpers ajoutés à main.go à la
// demande, sémantique PHP documentée par helper.
package emit

import (
	"strings"

	"code.hazyhaar.fr/devhoros/c2simd/p2go/ir"
)

// builtin émet l'expression d'un builtin et enregistre son helper éventuel.
func (e *emitter) builtin(b *ir.Builtin) string {
	arg := func(i int) string { return e.expr(b.Args[i], 0) }
	switch b.Name {
	case "min", "max": // builtins Go natifs, variadiques
		var args []string
		for i := range b.Args {
			args = append(args, arg(i))
		}
		return b.Name + "(" + strings.Join(args, ", ") + ")"
	case "chr": // PHP : octet n mod 256
		return "string([]byte{byte(" + arg(0) + ")})"
	case "str_replace": // PHP (search, replace, subject) → Go (subject, search, replace)
		e.needStrings = true
		return "strings.ReplaceAll(" + arg(2) + ", " + arg(0) + ", " + arg(1) + ")"
	case "trim": // charlist PHP par défaut, PAS unicode
		e.needStrings = true
		return "strings.Trim(" + arg(0) + ", \" \\t\\n\\r\\x00\\x0B\")"
	case "strpos": // SENTINELLE -1 (strings.Index), jamais false — écart documenté
		e.needStrings = true
		return "int64(strings.Index(" + arg(0) + ", " + arg(1) + "))"
	case "strtoupper": // helper dual scalaire/SIMD (F-p2go-simd-ascii-case)
		e.simd["upper"] = true
		return "p2goToUpper(" + arg(0) + ")"
	case "strtolower":
		e.simd["lower"] = true
		return "p2goToLower(" + arg(0) + ")"
	case "abs", "pow", "ord", "substr",
		"array_reverse", "array_slice", "array_fill", "in_array":
		h := builtinHelper[b.Name]
		e.helpers[h] = true
		var args []string
		for i := range b.Args {
			args = append(args, arg(i))
		}
		return h + "(" + strings.Join(args, ", ") + ")"
	}
	panic("emit: builtin inconnu " + b.Name)
}

var builtinHelper = map[string]string{
	"abs": "p2goAbs", "pow": "p2goPow", "ord": "p2goOrd", "substr": "p2goSubstr",
	"array_reverse": "p2goArrRev", "array_slice": "p2goArrSlice",
	"array_fill": "p2goArrFill", "in_array": "p2goInArray",
}

// helperSrc — code source des helpers, clef = nom Go.
var helperSrc = map[string]string{
	"p2goAbs": `func p2goAbs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
`,
	"p2goPow": `// pow int par carrés successifs ; exposant négatif = domaine float PHP, refusé.
func p2goPow(base, exp int64) int64 {
	if exp < 0 {
		panic("p2go: pow à exposant négatif hors subset int")
	}
	r := int64(1)
	for exp > 0 {
		if exp&1 == 1 {
			r *= base
		}
		base *= base
		exp >>= 1
	}
	return r
}
`,
	"p2goOrd": `func p2goOrd(s string) int64 {
	if len(s) == 0 {
		return 0
	}
	return int64(s[0])
}
`,
	"p2goSubstr": `// substr PHP 8 : offsets négatifs depuis la fin, bornes clampées.
func p2goSubstr(s string, start, length int64) string {
	n := int64(len(s))
	if start < 0 {
		start = n + start
		if start < 0 {
			start = 0
		}
	}
	if start >= n {
		return ""
	}
	end := start + length
	if length < 0 {
		end = n + length
	}
	if end > n {
		end = n
	}
	if end <= start {
		return ""
	}
	return s[start:end]
}
`,
	"p2goArrRev": `func p2goArrRev(a []int64) []int64 {
	out := make([]int64, len(a))
	for i, v := range a {
		out[len(a)-1-i] = v
	}
	return out
}
`,
	"p2goArrSlice": `// array_slice PHP : offset négatif depuis la fin, length négative = borne depuis la fin ; copie.
func p2goArrSlice(a []int64, off, length int64) []int64 {
	n := int64(len(a))
	if off < 0 {
		off = n + off
		if off < 0 {
			off = 0
		}
	}
	if off >= n {
		return []int64{}
	}
	end := off + length
	if length < 0 {
		end = n + length
	}
	if end > n {
		end = n
	}
	if end <= off {
		return []int64{}
	}
	return append([]int64(nil), a[off:end]...)
}
`,
	"p2goArrFill": `func p2goArrFill(start, count, val int64) []int64 {
	_ = start // 0 imposé par types/ (clefs denses)
	out := make([]int64, count)
	for i := range out {
		out[i] = val
	}
	return out
}
`,
	"p2goInArray": `func p2goInArray(v int64, a []int64) bool {
	for _, x := range a {
		if x == v {
			return true
		}
	}
	return false
}
`,
}
