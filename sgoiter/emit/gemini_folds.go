package emit

import (
	"fmt"
	"regexp"
	"strings"
)

// foldLiveRangeCopies: vA = vB (pure ident copy) → replace uses of vA with vB
// until the next assignment to vA. Batch non-overlapping copies per pass.
func foldLiveRangeCopies(block string) string {
	for round := 0; round < 32; round++ {
		next, n := foldLiveRangeCopiesBatch(block)
		block = next
		if n == 0 {
			break
		}
	}
	return block
}

func foldLiveRangeCopiesBatch(block string) (string, int) {
	lines := strings.Split(block, "\n")
	type asg struct {
		idx, nextDef int
		lhs, rhs      string
	}
	var copies []asg
	for i, ln := range lines {
		trim := strings.TrimSpace(ln)
		eq := strings.Index(trim, " = ")
		if eq <= 0 {
			continue
		}
		lhs := strings.TrimSpace(trim[:eq])
		rhs := strings.TrimSpace(trim[eq+3:])
		if !isSimpleIdent(lhs) || !strings.HasPrefix(lhs, "v") {
			continue
		}
		if !isSimpleIdent(rhs) || !strings.HasPrefix(rhs, "v") || lhs == rhs {
			continue
		}
		nextDef := len(lines)
		for j := i + 1; j < len(lines); j++ {
			t2 := strings.TrimSpace(lines[j])
			if e2 := strings.Index(t2, " = "); e2 > 0 && strings.TrimSpace(t2[:e2]) == lhs {
				nextDef = j
				break
			}
			if e2 := strings.Index(t2, " := "); e2 > 0 && strings.TrimSpace(t2[:e2]) == lhs {
				nextDef = j
				break
			}
		}
		rhsKilled := false
		for j := i + 1; j < nextDef; j++ {
			t2 := strings.TrimSpace(lines[j])
			if e2 := strings.Index(t2, " = "); e2 > 0 && strings.TrimSpace(t2[:e2]) == rhs {
				rhsKilled = true
				break
			}
		}
		if rhsKilled {
			continue
		}
		// only fold when lhs is read exactly once in the live range (safe)
		reads := 0
		for j := i + 1; j < nextDef; j++ {
			if identContains(lines[j], lhs) {
				// assignment TO lhs already ends range; bare "lhs = lhs + 1" counts
				t2 := strings.TrimSpace(lines[j])
				if e2 := strings.Index(t2, " = "); e2 > 0 && strings.TrimSpace(t2[:e2]) == lhs {
					if identContains(t2[e2+3:], lhs) {
						reads++
					}
					continue
				}
				reads++
			}
		}
		if reads != 1 {
			continue
		}
		// refuse control-flow headers
		safe := true
		for j := i + 1; j < nextDef; j++ {
			t2 := strings.TrimSpace(lines[j])
			if strings.HasPrefix(t2, "for ") || strings.HasPrefix(t2, "if ") || strings.HasPrefix(t2, "switch ") {
				if identContains(lines[j], lhs) {
					safe = false
					break
				}
			}
		}
		if !safe {
			continue
		}
		copies = append(copies, asg{i, nextDef, lhs, rhs})
	}
	if len(copies) == 0 {
		return block, 0
	}
	// greedy non-overlapping (by source line)
	drop := map[int]bool{}
	type repl struct{ from, to int; old, neu string }
	var repls []repl
	lastEnd := -1
	for _, c := range copies {
		if c.idx <= lastEnd {
			continue
		}
		drop[c.idx] = true
		repls = append(repls, repl{c.idx + 1, c.nextDef, c.lhs, c.rhs})
		lastEnd = c.nextDef - 1
	}
	out := make([]string, 0, len(lines))
	for j, ln := range lines {
		if drop[j] {
			continue
		}
		for _, r := range repls {
			if j >= r.from && j < r.to && identContains(ln, r.old) {
				ln = replaceIdent(ln, r.old, r.neu)
			}
		}
		out = append(out, ln)
	}
	return strings.Join(out, "\n"), len(drop)
}

// foldCrcBitSteps collapses the C bit-serial CRC mask expansion into one line:
//	crc = (crc >> 1) ^ (POLY & -(crc & 1))
// Go RE2 has no backrefs — match with a small line state machine.
func foldCrcBitSteps(block string) string {
	lines := strings.Split(block, "\n")
	out := make([]string, 0, len(lines))
	i := 0
	for i < len(lines) {
		if n, repl := matchCrcBitStep(lines, i); n > 0 {
			out = append(out, repl)
			i += n
			continue
		}
		out = append(out, lines[i])
		i++
	}
	return strings.Join(out, "\n")
}

// matchCrcBitStep tries to match the 8–9 line CRC bit step starting at i.
// Returns (linesConsumed, replacement) or (0, "").
func matchCrcBitStep(lines []string, i int) (int, string) {
	// need at least 8 lines
	if i+7 >= len(lines) {
		return 0, ""
	}
	parse := func(ln string) (pad, lhs, rhs string, ok bool) {
		trim := strings.TrimSpace(ln)
		eq := strings.Index(trim, " = ")
		if eq <= 0 {
			return "", "", "", false
		}
		pad = ln[:len(ln)-len(strings.TrimLeft(ln, "\t "))]
		return pad, strings.TrimSpace(trim[:eq]), strings.TrimSpace(trim[eq+3:]), true
	}
	// L0: vA = crc & 1
	pad, vA, rhs0, ok := parse(lines[i])
	if !ok || !strings.HasSuffix(rhs0, " & 1") {
		return 0, ""
	}
	crc := strings.TrimSpace(strings.TrimSuffix(rhs0, " & 1"))
	if !isSimpleIdent(crc) || !isSimpleIdent(vA) {
		return 0, ""
	}
	// L1: vB = int(vA)
	_, vB, rhs1, ok := parse(lines[i+1])
	if !ok || rhs1 != "int("+vA+")" {
		return 0, ""
	}
	// L2: vC = -vB
	_, vC, rhs2, ok := parse(lines[i+2])
	if !ok || rhs2 != "-"+vB {
		return 0, ""
	}
	// L3: vD = uint32(vC)
	_, vD, rhs3, ok := parse(lines[i+3])
	if !ok || rhs3 != "uint32("+vC+")" {
		return 0, ""
	}
	j := i + 4
	mask := vD
	// optional L4: vE = vD
	if _, vE, rhs4, ok := parse(lines[j]); ok && rhs4 == vD && isSimpleIdent(vE) {
		mask = vE
		j++
	}
	if j+3 >= len(lines) {
		return 0, ""
	}
	// Lshr: vF = crc >> 1
	_, vF, rhsF, ok := parse(lines[j])
	if !ok || rhsF != crc+" >> 1" {
		return 0, ""
	}
	// Land: vG = POLY & mask
	_, vG, rhsG, ok := parse(lines[j+1])
	if !ok || !strings.HasSuffix(rhsG, " & "+mask) {
		return 0, ""
	}
	poly := strings.TrimSpace(strings.TrimSuffix(rhsG, " & "+mask))
	// Lxor: vH = vF ^ vG
	_, vH, rhsH, ok := parse(lines[j+2])
	if !ok || rhsH != vF+" ^ "+vG {
		return 0, ""
	}
	// Lcrc: crc = vH
	_, lhsC, rhsC, ok := parse(lines[j+3])
	if !ok || lhsC != crc || rhsC != vH {
		return 0, ""
	}
	n := (j + 4) - i
	repl := fmt.Sprintf("%s%s = (%s >> 1) ^ (%s & -(%s & 1))", pad, crc, crc, poly, crc)
	return n, repl
}

// foldManualRotHelpers rewrites tweetnacl-style L32/R helpers to bits.RotateLeft*.
var (
	reL32Def = regexp.MustCompile(`(?s)\nfunc L32\(x uint64, c int\) uint64 \{[^}]*\}\n*`)
	reL32Call = regexp.MustCompile(`\bL32\(([^,]+),\s*([^)]+)\)`)
	reRDef = regexp.MustCompile(`(?s)\nfunc R\(x uint64, c int\) uint64 \{[^}]*\}\n*`)
	reRCall = regexp.MustCompile(`\bR\(([^,]+),\s*([^)]+)\)`)
)

func foldManualRotHelpers(block string) string {
	// Drop defs FIRST — \bL32( also matches "func L32(".
	if strings.Contains(block, "func L32(") {
		block = reL32Def.ReplaceAllString(block, "\n")
		block = reL32Call.ReplaceAllString(block, `uint64(bits.RotateLeft32(uint32($1), $2))`)
	}
	if strings.Contains(block, "func R(") {
		block = reRDef.ReplaceAllString(block, "\n")
		block = reRCall.ReplaceAllString(block, `bits.RotateLeft64($1, 64-($2))`)
	}
	return block
}

// foldIdentityReturnCast: return uint32(Fmix32(x)) → return Fmix32(x) when types match.
var reRetSameCast = regexp.MustCompile(`return (u?int(?:8|16|32|64)?|byte)\(([A-Za-z_][A-Za-z0-9_]*\([^)]*\))\)`)

func foldIdentityCallCast(block string) string {
	// only when outer cast type appears in callee name heuristics is unsafe;
	// drop uint32(Fmix32(...)) via known helpers
	block = strings.ReplaceAll(block, "return uint32(Fmix32(", "return Fmix32(")
	// fix extra paren: return Fmix32(v8)) → return Fmix32(v8)
	block = regexp.MustCompile(`return Fmix32\(([^)]+)\)\)`).ReplaceAllString(block, `return Fmix32($1)`)
	return block
}
