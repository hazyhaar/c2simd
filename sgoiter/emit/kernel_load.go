package emit

import (
	"fmt"
	"strings"
)

// rewriteKernelLELoads rewrites common binary.LittleEndian.Uint64/32 patterns
// to unsafe LE loads when mode=kernel (host LE dogfood).
// Safe patterns only: Uint64(x[i:]) / Uint64(x[i+K:]) and PutUint64 same.
func rewriteKernelLELoads(src string) string {
	if !strings.Contains(src, "binary.LittleEndian") {
		return src
	}
	// Uint64(slice[idx:]) 
	// binary.LittleEndian.Uint64(foo[bar:])
	reU64 := strings.NewReplacer() // manual via regex in arch
	_ = reU64
	out := src
	// Simple stable replaces for override-generated and IR-emitted forms:
	// binary.LittleEndian.Uint64(NAME[OFF:])
	// Use iterative scan
	out = replaceAllLE(out, "Uint64", 8)
	out = replaceAllLE(out, "Uint32", 4)
	out = replaceAllPut(out, "PutUint64", 8)
	out = replaceAllPut(out, "PutUint32", 4)
	return out
}

func replaceAllLE(src, fn string, _ int) string {
	// binary.LittleEndian.Uint64(expr)
	needle := "binary.LittleEndian." + fn + "("
	var b strings.Builder
	i := 0
	for {
		j := strings.Index(src[i:], needle)
		if j < 0 {
			b.WriteString(src[i:])
			break
		}
		j += i
		b.WriteString(src[i:j])
		// parse balanced parens
		start := j + len(needle)
		depth := 1
		k := start
		for k < len(src) && depth > 0 {
			switch src[k] {
			case '(':
				depth++
			case ')':
				depth--
			}
			k++
		}
		if depth != 0 {
			b.WriteString(src[j:])
			break
		}
		arg := src[start : k-1]
		// only transform simple slice index forms: ident[…]
		if isSimpleSliceArg(arg) {
			if fn == "Uint64" {
				fmt.Fprintf(&b, "*(*uint64)(unsafe.Pointer(&%s))", stripSliceEnd(arg))
			} else {
				fmt.Fprintf(&b, "*(*uint32)(unsafe.Pointer(&%s))", stripSliceEnd(arg))
			}
		} else {
			b.WriteString(src[j:k])
		}
		i = k
	}
	return b.String()
}

func replaceAllPut(src, fn string, _ int) string {
	needle := "binary.LittleEndian." + fn + "("
	var b strings.Builder
	i := 0
	for {
		j := strings.Index(src[i:], needle)
		if j < 0 {
			b.WriteString(src[i:])
			break
		}
		j += i
		b.WriteString(src[i:j])
		start := j + len(needle)
		depth := 1
		k := start
		for k < len(src) && depth > 0 {
			switch src[k] {
			case '(':
				depth++
			case ')':
				depth--
			}
			k++
		}
		if depth != 0 {
			b.WriteString(src[j:])
			break
		}
		args := src[start : k-1]
		// dst, val
		comma := splitTopComma(args)
		if len(comma) == 2 && isSimpleSliceArg(comma[0]) {
			dst := stripSliceEnd(strings.TrimSpace(comma[0]))
			val := strings.TrimSpace(comma[1])
			if fn == "PutUint64" {
				fmt.Fprintf(&b, "*(*uint64)(unsafe.Pointer(&%s)) = %s", dst, val)
			} else {
				fmt.Fprintf(&b, "*(*uint32)(unsafe.Pointer(&%s)) = %s", dst, val)
			}
		} else {
			b.WriteString(src[j:k])
		}
		i = k
	}
	return b.String()
}

func isSimpleSliceArg(s string) bool {
	s = strings.TrimSpace(s)
	// name[expr:] or name[expr]
	lb := strings.IndexByte(s, '[')
	if lb <= 0 {
		return false
	}
	if s[len(s)-1] != ']' {
		return false
	}
	ident := s[:lb]
	for _, c := range ident {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}

// stripSliceEnd: foo[i:] or foo[i] → foo[i] (element address)
func stripSliceEnd(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, ":]") {
		// foo[i:] → foo[i]
		return s[:len(s)-2] + "]"
	}
	return s
}

func splitTopComma(s string) []string {
	depth := 0
	for i, c := range s {
		switch c {
		case '(', '[':
			depth++
		case ')', ']':
			depth--
		case ',':
			if depth == 0 {
				return []string{s[:i], s[i+1:]}
			}
		}
	}
	return []string{s}
}
