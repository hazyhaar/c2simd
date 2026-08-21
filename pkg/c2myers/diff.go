package c2myers

import (
	"fmt"
	"strings"
)

// DiffLines calcule la séquence d'opérations d'édition minimale (Myers) entre deux ensembles de lignes.
func DiffLines(a, b []string) []EditOp {
	n := len(a)
	m := len(b)
	maxD := n + m

	if n == 0 && m == 0 {
		return nil
	}
	if n == 0 {
		ops := make([]EditOp, m)
		for i, line := range b {
			ops[i] = EditOp{Type: OpInsert, Line: line}
		}
		return ops
	}
	if m == 0 {
		ops := make([]EditOp, n)
		for i, line := range a {
			ops[i] = EditOp{Type: OpDelete, Line: line}
		}
		return ops
	}

	if maxD > 50000 {
		return nil
	}

	offset := maxD
	v := make([]int, 2*maxD+1)
	trace := make([][]int, 0, maxD+1)

	for d := 0; d <= maxD; d++ {
		vCopy := make([]int, len(v))
		copy(vCopy, v)
		trace = append(trace, vCopy)

		for k := -d; k <= d; k += 2 {
			var x int
			if k == -d || (k != d && v[k-1+offset] < v[k+1+offset]) {
				x = v[k+1+offset] // Déplacement vers le bas (insertion)
			} else {
				x = v[k-1+offset] + 1 // Déplacement vers la droite (suppression)
			}
			y := x - k

			// Suivre la diagonale (lignes égales)
			for x < n && y < m && a[x] == b[y] {
				x++
				y++
			}
			v[k+offset] = x

			if x >= n && y >= m {
				return backtrackMyers(a, b, trace, d, maxD)
			}
		}
	}

	return nil
}

func backtrackMyers(a, b []string, trace [][]int, d, maxD int) []EditOp {
	x := len(a)
	y := len(b)
	offset := maxD

	var ops []EditOp

	for step := d; step > 0; step-- {
		v := trace[step]
		k := x - y

		var prevK int
		if k == -step || (k != step && v[k-1+offset] < v[k+1+offset]) {
			prevK = k + 1
		} else {
			prevK = k - 1
		}

		prevX := v[prevK+offset]
		prevY := prevX - prevK

		for x > prevX && y > prevY {
			ops = append(ops, EditOp{Type: OpEqual, Line: a[x-1]})
			x--
			y--
		}

		if step > 0 {
			if x == prevX {
				ops = append(ops, EditOp{Type: OpInsert, Line: b[prevY]})
				y--
			} else if y == prevY {
				ops = append(ops, EditOp{Type: OpDelete, Line: a[prevX]})
				x--
			}
		}
	}

	for x > 0 && y > 0 {
		ops = append(ops, EditOp{Type: OpEqual, Line: a[x-1]})
		x--
		y--
	}

	// Inverser pour obtenir l'ordre chronologique
	for i, j := 0, len(ops)-1; i < j; i, j = i+1, j-1 {
		ops[i], ops[j] = ops[j], ops[i]
	}

	return ops
}

// GenerateUnifiedDiff produit une représentation unifiée standard (équivalente à git diff -p1).
func GenerateUnifiedDiff(oldPath, newPath string, oldText, newText string, contextLines int) string {
	if contextLines <= 0 {
		contextLines = 3
	}

	aLines := splitLines(oldText)
	bLines := splitLines(newText)
	ops := DiffLines(aLines, bLines)

	hunks := buildHunksFromOps(ops, contextLines)
	if len(hunks) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("--- a/%s\n", oldPath))
	sb.WriteString(fmt.Sprintf("+++ b/%s\n", newPath))

	for _, h := range hunks {
		sb.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", h.OldStart, h.OldLines, h.NewStart, h.NewLines))
		for _, l := range h.Lines {
			sb.WriteString(l)
			sb.WriteByte('\n')
		}
	}

	return sb.String()
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.Split(s, "\n")
}

func buildHunksFromOps(ops []EditOp, contextLines int) []Hunk {
	if contextLines <= 0 {
		contextLines = 3
	}

	// 1. Repérer les index d'opérations modifiées
	type changeRange struct {
		start int // Index dans ops
		end   int
	}
	var ranges []changeRange
	inChange := false
	startIdx := 0

	for i, op := range ops {
		if op.Type != OpEqual {
			if !inChange {
				inChange = true
				startIdx = i
			}
		} else {
			if inChange {
				inChange = false
				ranges = append(ranges, changeRange{start: startIdx, end: i})
			}
		}
	}
	if inChange {
		ranges = append(ranges, changeRange{start: startIdx, end: len(ops)})
	}

	if len(ranges) == 0 {
		return nil
	}

	// 2. Fusionner les plages proches (séparées par <= 2 * contextLines lignes égales)
	var merged []changeRange
	cur := ranges[0]
	for i := 1; i < len(ranges); i++ {
		gap := ranges[i].start - cur.end
		if gap <= 2*contextLines {
			cur.end = ranges[i].end // Fusion
		} else {
			merged = append(merged, cur)
			cur = ranges[i]
		}
	}
	merged = append(merged, cur)

	// 3. Calculer les numéros de lignes pour chaque op
	oldLineNum := make([]int, len(ops))
	newLineNum := make([]int, len(ops))
	oldL := 1
	newL := 1
	for i, op := range ops {
		oldLineNum[i] = oldL
		newLineNum[i] = newL
		switch op.Type {
		case OpEqual:
			oldL++
			newL++
		case OpDelete:
			oldL++
		case OpInsert:
			newL++
		}
	}

	// 4. Construire les Hunks avec contexte amont/aval
	var hunks []Hunk
	for _, r := range merged {
		hStart := r.start - contextLines
		if hStart < 0 {
			hStart = 0
		}
		hEnd := r.end + contextLines
		if hEnd > len(ops) {
			hEnd = len(ops)
		}

		hunkOldStart := oldLineNum[hStart]
		hunkNewStart := newLineNum[hStart]
		hunkOldCount := 0
		hunkNewCount := 0
		var hunkLines []string

		for i := hStart; i < hEnd; i++ {
			op := ops[i]
			switch op.Type {
			case OpEqual:
				hunkLines = append(hunkLines, " "+op.Line)
				hunkOldCount++
				hunkNewCount++
			case OpDelete:
				hunkLines = append(hunkLines, "-"+op.Line)
				hunkOldCount++
			case OpInsert:
				hunkLines = append(hunkLines, "+"+op.Line)
				hunkNewCount++
			}
		}

		hunks = append(hunks, Hunk{
			OldStart: hunkOldStart,
			OldLines: hunkOldCount,
			NewStart: hunkNewStart,
			NewLines: hunkNewCount,
			Lines:    hunkLines,
		})
	}

	return hunks
}
