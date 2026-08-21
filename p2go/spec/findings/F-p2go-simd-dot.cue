package findings

F_p2go_simd_dot: #Finding & {
	id:      "F-p2go-simd-dot"
	kernel:  "for ($i=0; $i<count($a); $i++) { $s += $a[$i] * $b[$i]; }"
	stage:   "rules"
	symptom: "Le produit scalaire — réduction vectorisable canonique — s'émettait en boucle scalaire ; VPMULLQ (multiply 64-bit) exige AVX-512, absent du poste."
	evidence: {
		file_line: "rules/simd_dot.go ; emit/simd.go (helper dot)"
		fixture:   "testdata/phpt/simd_dot.phpt"
		kat:       "pass"
	}
	lever:   "ir_rule"
	action:  "Nœud DotLoop + helper dual : le multiply 64-bit bas est ÉMULÉ AVX2 par 3 VPMULUDQ (MulWidenEven) — lo64(x·y) = uduq(lo,lo) + ((uduq(hi,lo)+uduq(lo,hi))<<32), wraparound mod 2⁶⁴ identique au scalaire int64. Garde : compteur non lu ailleurs ; b plus court que a = panic Go (PHP émettrait un warning null→0, divergence assumée fail-loud)."
	status:  "landed"
	rule_id: "simd_dot"
}
