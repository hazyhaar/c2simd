package emit

import (
	"fmt"
	"strings"
)

// maybePostUnrollHashes applies N7 FNV-style unroll on IR-emitted bodies.
// Looks for a simple counted loop with xor-byte + mul-const + inc.
func maybePostUnrollHashes(body string) string {
	out, ok := postUnrollFnvBody(body)
	if ok {
		return out
	}
	return body
}

// postUnrollFnvBody finds:
//
//	for IDX < LIM {
//		H = H ^ uint64(DATA[int(IDX)])
//		H = H * PRIME
//		IDX = IDX + 1
//	}
//
// and replaces with int index + ×8 subslice BCE + scalar tail (Gemini rec 3–4).
func postUnrollFnvBody(body string) (string, bool) {
	lines := strings.Split(body, "\n")
	for i := 0; i < len(lines); i++ {
		ln := strings.TrimSpace(lines[i])
		var idx, lim string
		if !strings.HasPrefix(ln, "for ") || !strings.Contains(ln, " < ") || !strings.HasSuffix(ln, "{") {
			continue
		}
		rest := strings.TrimPrefix(ln, "for ")
		rest = strings.TrimSuffix(rest, "{")
		rest = strings.TrimSpace(rest)
		parts := strings.Split(rest, " < ")
		if len(parts) != 2 {
			continue
		}
		idx = strings.TrimSpace(parts[0])
		lim = strings.TrimSpace(parts[1])
		if idx == "" || lim == "" {
			continue
		}
		if i+3 >= len(lines) {
			continue
		}
		l1 := strings.TrimSpace(lines[i+1])
		l2 := strings.TrimSpace(lines[i+2])
		l3 := strings.TrimSpace(lines[i+3])
		closeI := i + 4
		if closeI < len(lines) && strings.TrimSpace(lines[closeI]) != "}" {
			for closeI < len(lines) && strings.TrimSpace(lines[closeI]) == "" {
				closeI++
			}
			if closeI >= len(lines) || strings.TrimSpace(lines[closeI]) != "}" {
				continue
			}
		}
		var h, data, prime string
		if strings.Contains(l1, " ^= uint64(") {
			eq := strings.Index(l1, " ^= ")
			if eq >= 0 {
				h = strings.TrimSpace(l1[:eq])
				u := strings.Index(l1, "uint64(")
				r := l1[u+len("uint64("):]
				if br := strings.Index(r, "["); br >= 0 {
					data = strings.TrimSpace(r[:br])
				}
			}
		} else if strings.Contains(l1, "^ uint64(") {
			eq := strings.Index(l1, " = ")
			if eq >= 0 {
				h = strings.TrimSpace(l1[:eq])
				u := strings.Index(l1, "uint64(")
				r := l1[u+len("uint64("):]
				if br := strings.Index(r, "["); br >= 0 {
					data = strings.TrimSpace(r[:br])
				}
			}
		}
		if h == "" || data == "" {
			continue
		}
		if strings.HasPrefix(l2, h+" *= ") {
			prime = strings.TrimSpace(strings.TrimPrefix(l2, h+" *= "))
		} else if star := strings.LastIndex(l2, "* "); star >= 0 && (strings.Contains(l2, h+" * ") || strings.HasPrefix(l2, h+" = "+h+" * ")) {
			prime = strings.TrimSpace(l2[star+2:])
		}
		if prime == "" {
			continue
		}
		if l3 != idx+" = "+idx+" + 1" && l3 != idx+"++" && l3 != idx+" += 1" {
			continue
		}
		indent := lines[i][:len(lines[i])-len(strings.TrimLeft(lines[i], " \t"))]
		var rep strings.Builder
		// int index + subslice BCE (Gemini: no per-byte int(idx))
		fmt.Fprintf(&rep, "%sn := int(%s)\n", indent, lim)
		fmt.Fprintf(&rep, "%si := 0\n", indent)
		fmt.Fprintf(&rep, "%sfor ; i+8 <= n; i += 8 {\n", indent)
		fmt.Fprintf(&rep, "%s\tb := %s[i : i+8]\n", indent, data)
		for k := 0; k < 8; k++ {
			fmt.Fprintf(&rep, "%s\t%s = (%s ^ uint64(b[%d])) * %s\n", indent, h, h, k, prime)
		}
		fmt.Fprintf(&rep, "%s}\n", indent)
		fmt.Fprintf(&rep, "%sfor ; i < n; i++ {\n", indent)
		fmt.Fprintf(&rep, "%s\t%s = (%s ^ uint64(%s[i])) * %s\n", indent, h, h, data, prime)
		fmt.Fprintf(&rep, "%s}\n", indent)
		// drop prior "idx = 0" if present immediately above
		start := i
		if i > 0 {
			prev := strings.TrimSpace(lines[i-1])
			if prev == idx+" = 0" || prev == idx+" := 0" {
				start = i - 1
			}
		}
		// drop unused var idx declaration if it becomes dead (stripDeadVars later)
		var out []string
		out = append(out, lines[:start]...)
		out = append(out, strings.Split(strings.TrimSuffix(rep.String(), "\n"), "\n")...)
		out = append(out, lines[closeI+1:]...)
		return strings.Join(out, "\n"), true
	}
	return body, false
}
