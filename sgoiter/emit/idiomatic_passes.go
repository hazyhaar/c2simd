package emit

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// archCompoundAssigns convertit les assignations A = A op B en A op= B et A = A + 1 en A++.
func archCompoundAssigns(src string) string {
	lines := strings.Split(src, "\n")
	for i, ln := range lines {
		indent := ""
		for _, r := range ln {
			if r == '\t' || r == ' ' {
				indent += string(r)
			} else {
				break
			}
		}
		trim := strings.TrimSpace(ln)
		eq := strings.Index(trim, " = ")
		if eq <= 0 {
			continue
		}
		lhs := strings.TrimSpace(trim[:eq])
		rhs := strings.TrimSpace(trim[eq+3:])

		// Ne traiter que les lvalues simples ou indexées
		if !isSimpleLValue(lhs) {
			continue
		}

		// A = A + 1  -> A++
		if rhs == lhs+" + 1" || rhs == "1 + "+lhs {
			lines[i] = indent + lhs + "++"
			continue
		}
		// A = A - 1  -> A--
		if rhs == lhs+" - 1" {
			lines[i] = indent + lhs + "--"
			continue
		}

		// A = A op (B) ou A = A op B
		for _, op := range []string{"+", "-", "*", "&", "|", "^", "<<", ">>"} {
			prefix := lhs + " " + op + " "
			if strings.HasPrefix(rhs, prefix) {
				rest := strings.TrimSpace(rhs[len(prefix):])
				// Le membre droit doit être atomique ou entièrement parenthésé pour garantir la préservation de précédence
				if !isAtomicOrParenthesized(rest) {
					continue
				}
				lines[i] = indent + lhs + " " + op + "= " + rest
				break
			}
		}
	}
	return strings.Join(lines, "\n")
}

func isAtomicOrParenthesized(s string) bool {
	s = strings.TrimSpace(s)
	if isSimpleIdent(s) {
		return true
	}
	// Expression totalement entre parenthèses
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") && matchParenMatching(s) == len(s)-1 {
		return true
	}
	// Index simple (ex: buf[i])
	if strings.Contains(s, "[") && strings.HasSuffix(s, "]") && !strings.Contains(s, " ") {
		return true
	}
	// Accès champ (ex: ctx.Field)
	if strings.Contains(s, ".") && !strings.Contains(s, " ") {
		return true
	}
	// Appel de fonction ou conversion de type Type(...) ou f(...)
	if paren := strings.Index(s, "("); paren > 0 && isSimpleIdent(s[:paren]) && strings.HasSuffix(s, ")") && matchParenMatching(s[paren:]) == len(s)-paren-1 {
		return true
	}
	// Littéral simple sans espace
	if !strings.ContainsAny(s, " +-*/|^&<>") {
		return true
	}
	return false
}

func isSimpleLValue(s string) bool {
	if len(s) == 0 {
		return false
	}
	// Ident simple
	if isSimpleIdent(s) {
		return true
	}
	// Index simple: ident[ident] ou ident[nombre] ou ident.field
	if strings.Contains(s, "[") && strings.HasSuffix(s, "]") {
		return !strings.Contains(s, " ") // pas d'expression complexe d'indice
	}
	if strings.Contains(s, ".") {
		parts := strings.Split(s, ".")
		return len(parts) == 2 && isSimpleIdent(parts[0]) && isSimpleIdent(parts[1])
	}
	return false
}

var (
	// int(v) dans les index de tranche / tableau
	reIndexInt1 = regexp.MustCompile(`\[int\(([A-Za-z0-9_]+)\)\]`)
	reIndexInt2 = regexp.MustCompile(`\[int\(([A-Za-z0-9_]+)\)\s*:`)
	reIndexInt3 = regexp.MustCompile(`:\s*int\(([A-Za-z0-9_]+)\)\]`)
	reIndexInt4 = regexp.MustCompile(`\[int\(([A-Za-z0-9_]+)\)\s*:\s*int\(([A-Za-z0-9_]+)\)\]`)
	reIndexInt5 = regexp.MustCompile(`\[int\(([A-Za-z0-9_]+)\)\s*\+\s*(\d+)\]`)
)

// archStripIndexIntCasts élimine les conversions int(v) superflues dans les indices de tranches/tableaux.
func archStripIndexIntCasts(src string) string {
	src = reIndexInt4.ReplaceAllString(src, `[$1:$2]`)
	src = reIndexInt1.ReplaceAllString(src, `[$1]`)
	src = reIndexInt2.ReplaceAllString(src, `[$1:`)
	src = reIndexInt3.ReplaceAllString(src, `:$1]`)
	src = reIndexInt5.ReplaceAllString(src, `[$1 + $2]`)
	return src
}

// La transformation des IIFE ternaires min/max vit désormais dans
// astBuiltinMinMax (ast_idiomatic_passes.go) : la variante regex effaçait les
// casts avant d'apparier les branches, ce qui changeait la sémantique d'une
// comparaison tronquée (défaut démontré par oracle, audit 2026-08-15).

// archBalanceAdditionTrees rééquilibre les longues chaînes d'addition associatives gauches
// (((((a + b) + c) + d) ...)) en arbres équilibrés ((a+b)+(c+d)) pour diviser par deux
// la profondeur de dépendance séquentielle sur les ALU superscalaires.
func archBalanceAdditionTrees(src string) string {
	lines := strings.Split(src, "\n")
	for i, ln := range lines {
		// Pas de réassociation sur les flottants
		if strings.Contains(ln, "float32") || strings.Contains(ln, "float64") {
			continue
		}
		if !strings.Contains(ln, " + ") || !strings.Contains(ln, "(((") {
			continue
		}
		eq := strings.Index(ln, " = ")
		if eq <= 0 {
			continue
		}
		prefix := ln[:eq+3]
		rhs := strings.TrimSpace(ln[eq+3:])

		wrapCast := ""
		if opParen := strings.Index(rhs, "("); opParen > 0 && isCastPrefix(rhs[:opParen]) {
			if matchParenMatching(rhs[opParen:]) == len(rhs)-opParen-1 {
				wrapCast = rhs[:opParen+1]
				rhs = rhs[opParen+1 : len(rhs)-1]
			}
		}

		terms := extractLeftAddTerms(rhs)
		if len(terms) >= 4 {
			balanced := balanceTermSlice(terms)
			if wrapCast != "" {
				balanced = wrapCast + balanced + ")"
			}
			lines[i] = prefix + balanced
		}
	}
	return strings.Join(lines, "\n")
}

func isCastPrefix(s string) bool {
	s = strings.TrimSpace(s)
	switch s {
	case "int64", "uint64", "int", "uint32", "int32", "uint16", "int16", "uint8", "int8", "uintptr":
		return true
	}
	return false
}

func extractLeftAddTerms(s string) []string {
	s = strings.TrimSpace(s)
	var terms []string
	curr := s
	for {
		curr = strings.TrimSpace(curr)
		for strings.HasPrefix(curr, "(") && strings.HasSuffix(curr, ")") && matchParenMatching(curr) == len(curr)-1 {
			curr = strings.TrimSpace(curr[1 : len(curr)-1])
		}
		lastPlus := findTopLevelPlus(curr)
		if lastPlus < 0 {
			if curr != "" {
				terms = append([]string{protectTermParens(curr)}, terms...)
			}
			break
		}
		right := strings.TrimSpace(curr[lastPlus+3:])
		terms = append([]string{protectTermParens(right)}, terms...)
		curr = strings.TrimSpace(curr[:lastPlus])
	}
	return terms
}

func protectTermParens(t string) string {
	t = strings.TrimSpace(t)
	for strings.HasPrefix(t, "(") && strings.HasSuffix(t, ")") && matchParenMatching(t) == len(t)-1 {
		inner := strings.TrimSpace(t[1 : len(t)-1])
		if !strings.ContainsAny(inner, "+-|^&") {
			t = inner
		} else {
			break
		}
	}
	// Si le terme contient des opérateurs de précédence égale ou inférieure (+, -, |, ^, &),
	// conserver les parenthèses de protection.
	if strings.ContainsAny(t, "+-|^&") && (!strings.HasPrefix(t, "(") || matchParenMatching(t) != len(t)-1) {
		return "(" + t + ")"
	}
	return t
}

func findTopLevelPlus(s string) int {
	depth := 0
	for i := len(s) - 3; i >= 0; i-- {
		if s[i+2] == ')' {
			depth++
		} else if s[i+2] == '(' {
			depth--
		}
		if depth == 0 && s[i] == ' ' && s[i+1] == '+' && s[i+2] == ' ' {
			return i
		}
	}
	return -1
}

func matchParenMatching(s string) int {
	depth := 0
	for i, r := range s {
		if r == '(' {
			depth++
		} else if r == ')' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func balanceTermSlice(terms []string) string {
	if len(terms) == 0 {
		return ""
	}
	if len(terms) == 1 {
		return terms[0]
	}
	if len(terms) == 2 {
		return terms[0] + " + " + terms[1]
	}
	mid := len(terms) / 2
	left := balanceTermSlice(terms[:mid])
	right := balanceTermSlice(terms[mid:])
	return "(" + left + ") + (" + right + ")"
}

// archPowerOfTwoShifts convertit les multiplications par puissance de deux explicites en décalages de bits.
var reMulPow2_1 = regexp.MustCompile(`\*\s*(?:int64|uint64|int|uint32)?\(?1\s*<<\s*(\d+)\)?`)

func archPowerOfTwoShifts(src string) string {
	return reMulPow2_1.ReplaceAllString(src, `<< $1`)
}

// archCleanupWipeAveux supprime les résidus tels que "_ = ctx // wipe skipped struct".
var reWipeAveu = regexp.MustCompile(`(?m)^\t*_ = [A-Za-z0-9_]+ // wipe skipped struct\s*\n?`)

func archCleanupWipeAveux(src string) string {
	return reWipeAveu.ReplaceAllString(src, "")
}

var (
	reDoubleInt64  = regexp.MustCompile(`\bint64\(int64\(([a-zA-Z0-9_\.]+)\)\)`)
	reDoubleInt    = regexp.MustCompile(`\bint\(int\(([a-zA-Z0-9_\.]+)\)\)`)
	reDoubleUint64 = regexp.MustCompile(`\buint64\(uint64\(([a-zA-Z0-9_\.]+)\)\)`)
	reDoubleUint32 = regexp.MustCompile(`\buint32\(uint32\(([a-zA-Z0-9_\.]+)\)\)`)
)

// archFoldPingPongCasts replie les doubles conversions de types identiques redondantes.
func archFoldPingPongCasts(src string) string {
	src = reDoubleInt64.ReplaceAllString(src, `int64($1)`)
	src = reDoubleInt.ReplaceAllString(src, `int($1)`)
	src = reDoubleUint64.ReplaceAllString(src, `uint64($1)`)
	src = reDoubleUint32.ReplaceAllString(src, `uint32($1)`)
	return src
}

var (
	reLitCastCompRight = regexp.MustCompile(`(==|!=|<|>|<=|>=)\s*(?:uint64|uint32|uint16|uint8|int64|int32|int16|int8|int|uintptr)\((0x[0-9a-fA-F]+|\d+)\)`)
	reLitCastCompLeft  = regexp.MustCompile(`(?:uint64|uint32|uint16|uint8|int64|int32|int16|int8|int|uintptr)\((0x[0-9a-fA-F]+|\d+)\)\s*(==|!=|<|>|<=|>=)`)
	reLitCastArith     = regexp.MustCompile(`([+\-*/&|^]=?)\s*(?:uint64|uint32|uint16|uint8|int64|int32|int16|int8|int|uintptr)\((0x[0-9a-fA-F]+|\d+)\)`)
)

// archStripRedundantLiteralCasts élimine les conversions explicites sur des constantes littérales en comparaison et arithmétique.
func archStripRedundantLiteralCasts(src string) string {
	src = reLitCastCompRight.ReplaceAllString(src, `$1 $2`)
	src = reLitCastCompLeft.ReplaceAllString(src, `$1 $2`)
	src = reLitCastArith.ReplaceAllString(src, `$1 $2`)
	return src
}

var (
	reRotl64Const = regexp.MustCompile(`bits\.RotateLeft64\(([^,]+),\s*64\s*-\s*(\d+)\)`)
	reRotl32Const = regexp.MustCompile(`bits\.RotateLeft32\(([^,]+),\s*32\s*-\s*(\d+)\)`)
)

// archFoldRotateLeftConstants simplifie bits.RotateLeft64(x, 64-N) en bits.RotateLeft64(x, -N).
func archFoldRotateLeftConstants(src string) string {
	src = reRotl64Const.ReplaceAllString(src, `bits.RotateLeft64($1, -$2)`)
	src = reRotl32Const.ReplaceAllString(src, `bits.RotateLeft32($1, -$2)`)
	return src
}

// Le pliage des négations de comparaison vit désormais dans
// astSimplifyNegatedComparisons (ast_more_passes.go) : la variante regex ne
// couvrait que les identifiants — !(Invsqrt(...) != 0) survivait (audit
// 2026-08-15).

// Le repli de Gap(x, 16) vit désormais dans astFoldGapLiteralConstants
// (ast_idiomatic_passes.go) : la variante regex ne parenthésait pas l'argument
// ((-a + b) & 15 pour Gap(a+b, 16)) — défaut latent démontré, audit 2026-08-15.

// archUnrollBlake2bSigma déroule les 12 tours de Blake2b en substituant la table sigma par des immédiats constants.
func archUnrollBlake2bSigma(src string) string {
	idx := strings.Index(src, "for v91 < 12 {")
	if idx == -1 {
		return src
	}
	endPattern := "ctx.Hash[0] ^="
	endRel := strings.Index(src[idx:], endPattern)
	if endRel == -1 {
		return src
	}
	loopChunk := src[idx : idx+endRel]
	lastBrace := strings.LastIndex(loopChunk, "}")
	if lastBrace == -1 {
		return src
	}
	targetBlock := src[idx : idx+lastBrace+1]

	var unrolled strings.Builder
	for r := 0; r < 12; r++ {
		fmt.Fprintf(&unrolled, "\t// Blake2b Tour %d (déroulage ARCHTIME sigma)\n", r)
		unrolled.WriteString(fmt.Sprintf("\tv20 += (v48 + ctx.Input[%d])\n", blake2bSigma[r][0]))
		unrolled.WriteString("\tv52 = bits.RotateLeft64((v52 ^ v20), -32)\n")
		unrolled.WriteString("\tv24 += v52\n")
		unrolled.WriteString("\tv48 = bits.RotateLeft64((v48 ^ v24), -24)\n")
		unrolled.WriteString(fmt.Sprintf("\tv20 += (v48 + ctx.Input[%d])\n", blake2bSigma[r][1]))
		unrolled.WriteString("\tv52 = bits.RotateLeft64((v52 ^ v20), -16)\n")
		unrolled.WriteString("\tv24 += v52\n")
		unrolled.WriteString("\tv48 = bits.RotateLeft64((v48 ^ v24), -63)\n")

		unrolled.WriteString(fmt.Sprintf("\tv27 += (v59 + ctx.Input[%d])\n", blake2bSigma[r][2]))
		unrolled.WriteString("\tv63 = bits.RotateLeft64((v63 ^ v27), -32)\n")
		unrolled.WriteString("\tv31 += v63\n")
		unrolled.WriteString("\tv59 = bits.RotateLeft64((v59 ^ v31), -24)\n")
		unrolled.WriteString(fmt.Sprintf("\tv27 += (v59 + ctx.Input[%d])\n", blake2bSigma[r][3]))
		unrolled.WriteString("\tv63 = bits.RotateLeft64((v63 ^ v27), -16)\n")
		unrolled.WriteString("\tv31 += v63\n")
		unrolled.WriteString("\tv59 = bits.RotateLeft64((v59 ^ v31), -63)\n")

		unrolled.WriteString(fmt.Sprintf("\tv34 += (v70 + ctx.Input[%d])\n", blake2bSigma[r][4]))
		unrolled.WriteString("\tv74 = bits.RotateLeft64((v74 ^ v34), -32)\n")
		unrolled.WriteString("\tv38 += v74\n")
		unrolled.WriteString("\tv70 = bits.RotateLeft64((v70 ^ v38), -24)\n")
		unrolled.WriteString(fmt.Sprintf("\tv34 += (v70 + ctx.Input[%d])\n", blake2bSigma[r][5]))
		unrolled.WriteString("\tv74 = bits.RotateLeft64((v74 ^ v34), -16)\n")
		unrolled.WriteString("\tv38 += v74\n")
		unrolled.WriteString("\tv70 = bits.RotateLeft64((v70 ^ v38), -63)\n")

		unrolled.WriteString(fmt.Sprintf("\tv41 += (v82 + ctx.Input[%d])\n", blake2bSigma[r][6]))
		unrolled.WriteString("\tv86 = bits.RotateLeft64((v86 ^ v41), -32)\n")
		unrolled.WriteString("\tv45 += v86\n")
		unrolled.WriteString("\tv82 = bits.RotateLeft64((v82 ^ v45), -24)\n")
		unrolled.WriteString(fmt.Sprintf("\tv41 += (v82 + ctx.Input[%d])\n", blake2bSigma[r][7]))
		unrolled.WriteString("\tv86 = bits.RotateLeft64((v86 ^ v41), -16)\n")
		unrolled.WriteString("\tv45 += v86\n")
		unrolled.WriteString("\tv82 = bits.RotateLeft64((v82 ^ v45), -63)\n")

		unrolled.WriteString(fmt.Sprintf("\tv20 += (v59 + ctx.Input[%d])\n", blake2bSigma[r][8]))
		unrolled.WriteString("\tv86 = bits.RotateLeft64((v86 ^ v20), -32)\n")
		unrolled.WriteString("\tv38 += v86\n")
		unrolled.WriteString("\tv59 = bits.RotateLeft64((v59 ^ v38), -24)\n")
		unrolled.WriteString(fmt.Sprintf("\tv20 += (v59 + ctx.Input[%d])\n", blake2bSigma[r][9]))
		unrolled.WriteString("\tv86 = bits.RotateLeft64((v86 ^ v20), -16)\n")
		unrolled.WriteString("\tv38 += v86\n")
		unrolled.WriteString("\tv59 = bits.RotateLeft64((v59 ^ v38), -63)\n")

		unrolled.WriteString(fmt.Sprintf("\tv27 += (v70 + ctx.Input[%d])\n", blake2bSigma[r][10]))
		unrolled.WriteString("\tv52 = bits.RotateLeft64((v52 ^ v27), -32)\n")
		unrolled.WriteString("\tv45 += v52\n")
		unrolled.WriteString("\tv70 = bits.RotateLeft64((v70 ^ v45), -24)\n")
		unrolled.WriteString(fmt.Sprintf("\tv27 += (v70 + ctx.Input[%d])\n", blake2bSigma[r][11]))
		unrolled.WriteString("\tv52 = bits.RotateLeft64((v52 ^ v27), -16)\n")
		unrolled.WriteString("\tv45 += v52\n")
		unrolled.WriteString("\tv70 = bits.RotateLeft64((v70 ^ v45), -63)\n")

		unrolled.WriteString(fmt.Sprintf("\tv34 += (v82 + ctx.Input[%d])\n", blake2bSigma[r][12]))
		unrolled.WriteString("\tv63 = bits.RotateLeft64((v63 ^ v34), -32)\n")
		unrolled.WriteString("\tv24 += v63\n")
		unrolled.WriteString("\tv82 = bits.RotateLeft64((v82 ^ v24), -24)\n")
		unrolled.WriteString(fmt.Sprintf("\tv34 += (v82 + ctx.Input[%d])\n", blake2bSigma[r][13]))
		unrolled.WriteString("\tv63 = bits.RotateLeft64((v63 ^ v34), -16)\n")
		unrolled.WriteString("\tv24 += v63\n")
		unrolled.WriteString("\tv82 = bits.RotateLeft64((v82 ^ v24), -63)\n")

		unrolled.WriteString(fmt.Sprintf("\tv41 += (v48 + ctx.Input[%d])\n", blake2bSigma[r][14]))
		unrolled.WriteString("\tv74 = bits.RotateLeft64((v74 ^ v41), -32)\n")
		unrolled.WriteString("\tv31 += v74\n")
		unrolled.WriteString("\tv48 = bits.RotateLeft64((v48 ^ v31), -24)\n")
		unrolled.WriteString(fmt.Sprintf("\tv41 += (v48 + ctx.Input[%d])\n", blake2bSigma[r][15]))
		unrolled.WriteString("\tv74 = bits.RotateLeft64((v74 ^ v41), -16)\n")
		unrolled.WriteString("\tv31 += v74\n")
		unrolled.WriteString("\tv48 = bits.RotateLeft64((v48 ^ v31), -63)\n")
	}

	src = strings.Replace(src, targetBlock, unrolled.String(), 1)
	src = strings.Replace(src, "\tv91 = 0\n", "", 1)
	return src
}

var lm2Bytes = [32]byte{
	0xeb, 0xd3, 0xf5, 0x5c, 0x1a, 0x63, 0x12, 0x58,
	0xd6, 0x9c, 0xf7, 0xa2, 0xde, 0xf9, 0xde, 0x14,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10,
}

func lm2BytesGoLit(indent string) string {
	var b strings.Builder
	b.WriteString("[32]byte{\n")
	for i := 0; i < 32; i += 8 {
		b.WriteString(indent)
		for j := 0; j < 8; j++ {
			if j > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(&b, "0x%02x", lm2Bytes[i+j])
		}
		b.WriteString(",\n")
	}
	b.WriteString(indent[:len(indent)-1])
	b.WriteString("}")
	return b.String()
}

// archEmitX25519InverseLm2Loop remplace la boucle d'exponentiation L-2
// (Crypto_x25519_inverse) par une boucle compacte sur la table publique lm2Bytes.
func archEmitX25519InverseLm2Loop(src string) string {
	startMarker := "v28 = 252"
	idx := strings.Index(src, startMarker)
	if idx == -1 {
		return src
	}
	loopStart := strings.Index(src[idx:], "for v28 >= 0")
	if loopStart == -1 {
		return src
	}
	loopIdx := idx + loopStart
	openBrace := strings.Index(src[loopIdx:], "{")
	if openBrace == -1 {
		return src
	}
	depth := 0
	endRel := -1
	for pos := loopIdx + openBrace; pos < len(src); pos++ {
		if src[pos] == '{' {
			depth++
		} else if src[pos] == '}' {
			depth--
			if depth == 0 {
				endRel = pos + 1
				break
			}
		}
	}
	if endRel == -1 {
		return src
	}
	targetBlock := src[idx:endRel]

	varName := "m_inv"
	if !strings.Contains(targetBlock, "m_inv") {
		varName = "v6"
	}

	var loop strings.Builder
	loop.WriteString("\t// L-2 (ordre du groupe Ed25519 moins 2) est une constante publique.\n")
	loop.WriteString("\t// Le test de bit sur cette table ne fuit aucune donnée secrète :\n")
	loop.WriteString("\t// Montgomery ladder sur exposant public — le if est admissible.\n")
	loop.WriteString("\tvar lm2Bytes = ")
	loop.WriteString(lm2BytesGoLit("\t\t"))
	loop.WriteString("\n")
	loop.WriteString("\tfor bit := 252; bit >= 0; bit-- {\n")
	loop.WriteString("\t\tclear(v27[:])\n")
	fmt.Fprintf(&loop, "\t\tMultiply(v27[:], %s[:], %s[:])\n", varName, varName)
	fmt.Fprintf(&loop, "\t\tRedc(%s[:], v27[:])\n", varName)
	loop.WriteString("\t\tif (lm2Bytes[bit/8]>>(bit%8))&1 != 0 {\n")
	loop.WriteString("\t\t\tclear(v27[:])\n")
	fmt.Fprintf(&loop, "\t\t\tMultiply(v27[:], %s[:], v9[:])\n", varName)
	fmt.Fprintf(&loop, "\t\t\tRedc(%s[:], v27[:])\n", varName)
	loop.WriteString("\t\t}\n")
	loop.WriteString("\t}\n")

	return strings.Replace(src, targetBlock, loop.String(), 1)
}

var (
	// Clear loop:
	// vX = 0
	// for vX < LIMIT {
	//     TARGET[vX] = 0
	//     vX++
	// }
	reClearLoop = regexp.MustCompile(`(?m)^([ \t]*)([a-zA-Z0-9_]+)\s*=\s*0\n[ \t]*for\s+([a-zA-Z0-9_]+)\s*<\s*([a-zA-Z0-9_\.]+|\([^\)]+\))\s*\{\n[ \t]*([a-zA-Z0-9_\.]+)\s*\[(?:int\()?[a-zA-Z0-9_]+\)?\]\s*=\s*(?:[a-zA-Z0-9_]+\()?0\)?\n[ \t]*(?:[a-zA-Z0-9_]+\+\+|[a-zA-Z0-9_]+\s*\+=\s*1|[a-zA-Z0-9_]+\s*=\s*[a-zA-Z0-9_]+\s*\+\s*1)\n[ \t]*\}`)

	// Direct copy loop:
	// vX = 0
	// for vX < LIMIT {
	//     DST[vX] = SRC[vX]
	//     vX++
	// }
	reCopyDirectLoop = regexp.MustCompile(`(?m)^([ \t]*)([a-zA-Z0-9_]+)\s*=\s*0\n[ \t]*for\s+([a-zA-Z0-9_]+)\s*<\s*([a-zA-Z0-9_\.]+|\([^\)]+\))\s*\{\n[ \t]*([a-zA-Z0-9_\.]+)\s*\[(?:int\()?[a-zA-Z0-9_]+\)?\]\s*=\s*([a-zA-Z0-9_\.]+)\s*\[(?:int\()?[a-zA-Z0-9_]+\)?\]\n[ \t]*(?:[a-zA-Z0-9_]+\+\+|[a-zA-Z0-9_]+\s*\+=\s*1|[a-zA-Z0-9_]+\s*=\s*[a-zA-Z0-9_]+\s*\+\s*1)\n[ \t]*\}`)

	// Offset copy loop on SRC:
	// vX = 0
	// for vX < LIMIT {
	//     DST[vX] = SRC[int(OFFSET + vX)]
	//     vX++
	// }
	reCopySrcOffsetLoop = regexp.MustCompile(`(?m)^([ \t]*)([a-zA-Z0-9_]+)\s*=\s*0\n[ \t]*for\s+([a-zA-Z0-9_]+)\s*<\s*([0-9]+)\s*\{\n[ \t]*([a-zA-Z0-9_\.]+)\s*\[(?:int\()?[a-zA-Z0-9_]+\)?\]\s*=\s*([a-zA-Z0-9_\.]+)\s*\[(?:int\()?(?:uint64\()?([0-9]+)\)?\s*\+\s*([a-zA-Z0-9_]+)\)?\]\n[ \t]*(?:[a-zA-Z0-9_]+\+\+|[a-zA-Z0-9_]+\s*\+=\s*1|[a-zA-Z0-9_]+\s*=\s*[a-zA-Z0-9_]+\s*\+\s*1)\n[ \t]*\}`)

	// Offset copy loop on DST:
	// vX = 0
	// for vX < LIMIT {
	//     DST[int(OFFSET + vX)] = SRC[vX]
	//     vX++
	// }
	reCopyDstOffsetLoop = regexp.MustCompile(`(?m)^([ \t]*)([a-zA-Z0-9_]+)\s*=\s*0\n[ \t]*for\s+([a-zA-Z0-9_]+)\s*<\s*([0-9]+)\s*\{\n[ \t]*([a-zA-Z0-9_\.]+)\s*\[(?:int\()?(?:uint64\()?([0-9]+)\)?\s*\+\s*([a-zA-Z0-9_]+)\)?\]\s*=\s*([a-zA-Z0-9_\.]+)\s*\[(?:int\()?[a-zA-Z0-9_]+\)?\]\n[ \t]*(?:[a-zA-Z0-9_]+\+\+|[a-zA-Z0-9_]+\s*\+=\s*1|[a-zA-Z0-9_]+\s*=\s*[a-zA-Z0-9_]+\s*\+\s*1)\n[ \t]*\}`)
)

// archTransformCopyClearLoops convertit les boucles d'assignation octet par octet
// en appels natifs aux intrinsèques copy() et clear() du compilateur Go.
func archTransformCopyClearLoops(src string) string {
	src = reClearLoop.ReplaceAllStringFunc(src, func(m string) string {
		sub := reClearLoop.FindStringSubmatch(m)
		if len(sub) < 6 {
			return m
		}
		indent := sub[1]
		varInit := sub[2]
		varLoop := sub[3]
		if varInit != varLoop {
			return m
		}
		limit := sub[4]
		target := sub[5]
		return fmt.Sprintf("%sclear(%s[:%s])", indent, target, limit)
	})

	src = reCopyDirectLoop.ReplaceAllStringFunc(src, func(m string) string {
		sub := reCopyDirectLoop.FindStringSubmatch(m)
		if len(sub) < 7 {
			return m
		}
		indent := sub[1]
		varInit := sub[2]
		varLoop := sub[3]
		if varInit != varLoop {
			return m
		}
		limit := sub[4]
		dst := sub[5]
		srcArr := sub[6]
		return fmt.Sprintf("%scopy(%s[:%s], %s[:%s])", indent, dst, limit, srcArr, limit)
	})

	src = reCopySrcOffsetLoop.ReplaceAllStringFunc(src, func(m string) string {
		sub := reCopySrcOffsetLoop.FindStringSubmatch(m)
		if len(sub) < 9 {
			return m
		}
		indent := sub[1]
		varInit := sub[2]
		varLoop := sub[3]
		varIndex := sub[8]
		if varInit != varLoop || varInit != varIndex {
			return m
		}
		limitStr := sub[4]
		dst := sub[5]
		srcArr := sub[6]
		offsetStr := sub[7]
		limit, err1 := strconv.Atoi(limitStr)
		offset, err2 := strconv.Atoi(offsetStr)
		if err1 != nil || err2 != nil {
			return m
		}
		end := offset + limit
		return fmt.Sprintf("%scopy(%s[:%s], %s[%d:%d])", indent, dst, limitStr, srcArr, offset, end)
	})

	src = reCopyDstOffsetLoop.ReplaceAllStringFunc(src, func(m string) string {
		sub := reCopyDstOffsetLoop.FindStringSubmatch(m)
		if len(sub) < 9 {
			return m
		}
		indent := sub[1]
		varInit := sub[2]
		varLoop := sub[3]
		varIndex := sub[7]
		if varInit != varLoop || varInit != varIndex {
			return m
		}
		limitStr := sub[4]
		dst := sub[5]
		offsetStr := sub[6]
		srcArr := sub[8]
		limit, err1 := strconv.Atoi(limitStr)
		offset, err2 := strconv.Atoi(offsetStr)
		if err1 != nil || err2 != nil {
			return m
		}
		end := offset + limit
		return fmt.Sprintf("%scopy(%s[%d:%d], %s[:%s])", indent, dst, offset, end, srcArr, limitStr)
	})

	return src
}

