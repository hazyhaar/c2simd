package findings

F_p2go_ternary_subexpr: #Finding & {
	id:      "F-p2go-ternary-subexpr"
	kernel:  "$y = 10 + ($x > 5 ? 1 : 2) * 100; f($c ? a : b);"
	stage:   "front"
	symptom: "Le ternaire n'était accepté qu'en position statement (F-p2go-ternary-expr) — refusé en sous-expression arithmétique et en argument d'appel."
	evidence: {
		file_line: "front/desugar.go (passe de hoisting post-parse)"
		fixture:   "testdata/phpt/ternary_subexpr.phpt"
		kat:       "pass"
	}
	lever:  "front"
	action: "Désucreur post-parse : chaque Ternary/Match d'une expression est hoisté en temporaire p2go_tN affecté dans un If/Switch posé AVANT le statement porteur — paresse préservée (les branches vivent dans les corps), forme courte ?: évaluée une seule fois via second temporaire. Interdits gardés : sucre en condition de boucle (réévaluation par itération) et en valeur de case."
	status: "landed"
	notes:  "Remplace le désucrage statement-level de F-p2go-ternary-expr (généralisation)."
}
