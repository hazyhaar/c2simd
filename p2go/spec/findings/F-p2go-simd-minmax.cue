package findings

F_p2go_simd_minmax: #Finding & {
	id:      "F-p2go-simd-minmax"
	kernel:  "for (…) { if ($a[$i] > $m) { $m = $a[$i]; } }"
	stage:   "rules"
	symptom: "La recherche d'extremum restait scalaire ; VPMAXSQ/VPMINSQ (min/max int64) exigent AVX-512."
	evidence: {
		file_line: "rules/simd_minmax.go ; emit/simd.go (helpers max/min)"
		fixture:   "testdata/phpt/simd_minmax.phpt"
		kat:       "pass"
	}
	lever:   "ir_rule"
	action:  "Nœud MinMaxLoop (sens > ou <) + helpers duals : max/min émulés AVX2 par Greater (VPCMPGTQ, comparaison SIGNÉE native) + IfElse (blend) ; amorce vectorielle sur les 4 premiers éléments, réduction des lanes et queue en scalaire ; m = min/max(m, éléments) préserve la sémantique quand le tableau est vide."
	status:  "landed"
	rule_id: "simd_minmax"
}
