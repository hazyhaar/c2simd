package emit

import (
	"fmt"
	"strings"
)

// maybeRewriteMurmurLoop rewrites the classic
//
//	vN = -vBlocks
//	for vN != 0 {
//	  ... key[int((vBlocks * 4) + (vN * 4))] ...
//	  vN = vN + 1
//	}
//
// into a forward 0..vBlocks-1 loop with key[int(j*4)].
func maybeRewriteMurmurLoop(body string) string {
	// Find "vXX = -vYY"
	lines := strings.Split(body, "\n")
	for i := 0; i < len(lines)-2; i++ {
		ln := strings.TrimSpace(lines[i])
		var ind, blocks string
		if _, err := fmt.Sscanf(ln, "%s = -%s", &ind, &blocks); err != nil {
			continue
		}
		// strip trailing junk
		ind = strings.TrimSuffix(ind, "")
		if !strings.HasPrefix(ind, "v") || !strings.HasPrefix(blocks, "v") {
			continue
		}
		// next non-empty should be for ind != 0 {
		j := i + 1
		for j < len(lines) && strings.TrimSpace(lines[j]) == "" {
			j++
		}
		if j >= len(lines) {
			continue
		}
		fl := strings.TrimSpace(lines[j])
		want := "for " + ind + " != 0 {"
		if fl != want {
			continue
		}
		// find matching close brace at same indent as for
		forIndent := lines[j][:len(lines[j])-len(strings.TrimLeft(lines[j], " \t"))]
		k := j + 1
		depth := 1
		for k < len(lines) && depth > 0 {
			t := strings.TrimSpace(lines[k])
			if strings.HasSuffix(t, "{") {
				depth++
			}
			if t == "}" {
				depth--
			}
			k++
		}
		if depth != 0 {
			continue
		}
		// body is lines[j+1 : k-1]
		bodyLines := lines[j+1 : k-1]
		// replace index pattern (blocks * 4) + (ind * 4) → (j * 4)
		// and ind = ind + 1 at end → j = j + 1
		jname := "j"
		var nb []string
		nb = append(nb, forIndent+jname+" := 0")
		nb = append(nb, forIndent+"for "+jname+" < "+blocks+" {")
		inner := forIndent + "\t"
		for _, bl := range bodyLines {
			s := bl
			// replace ind with j in index expressions carefully
			s = strings.ReplaceAll(s, "("+blocks+" * 4) + ("+ind+" * 4)", "("+jname+" * 4)")
			s = strings.ReplaceAll(s, "("+ind+" * 4) + ("+blocks+" * 4)", "("+jname+" * 4)")
			// end increment
			if strings.TrimSpace(s) == ind+" = "+ind+" + 1" {
				s = inner + jname + " = " + jname + " + 1"
				nb = append(nb, s)
				continue
			}
			// reindent body lines: keep relative
			trim := strings.TrimLeft(s, " \t")
			s = inner + trim
			// replace remaining bare ind token in array indices
			s = replaceWord(s, ind, jname)
			nb = append(nb, s)
		}
		nb = append(nb, forIndent+"}")
		// drop the "ind = -blocks" line and unused "var ind ..."
		var out []string
		for _, pl := range lines[:i] {
			if strings.TrimSpace(pl) == "var "+ind+" int" || strings.TrimSpace(pl) == "var "+ind+" int32" {
				continue
			}
			out = append(out, pl)
		}
		out = append(out, nb...)
		out = append(out, lines[k:]...)
		return strings.Join(out, "\n")
	}
	return body
}

func replaceWord(s, old, new string) string {
	// naive token replace
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_')
	})
	// Use strings.Replace with word boundaries via manual scan
	var b strings.Builder
	i := 0
	for i < len(s) {
		if i+len(old) <= len(s) && s[i:i+len(old)] == old {
			// word boundary
			beforeOK := i == 0 || !identByte(s[i-1])
			afterOK := i+len(old) == len(s) || !identByte(s[i+len(old)])
			if beforeOK && afterOK {
				b.WriteString(new)
				i += len(old)
				continue
			}
		}
		b.WriteByte(s[i])
		i++
	}
	_ = parts
	return b.String()
}

func identByte(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}
