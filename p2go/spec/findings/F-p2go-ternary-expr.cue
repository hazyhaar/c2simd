package findings

F_p2go_ternary_expr: #Finding & {
	id:      "F-p2go-ternary-expr"
	kernel:  "$y = cond ? a : b;"
	stage:   "dogfood"
	symptom: "Corpus ternary.php refusé err_parse — le ternaire exige une expression conditionnelle en contexte valeur, absente de l'IR v0.1."
	evidence: {
		file_line: "testdata/dogfood/ternary.php"
		kat:       "n/a"
	}
	lever:  "front"
	action: "Désucrage AU PARSER en positions statement — $v = c ? a : b, return c ? a : b, echo c ? a : b (argument unique), forme courte ?: incluse — vers un If à deux branches ; évaluation paresseuse préservée sans slot temporaire ni helper eager. Refusé en clauses de for et en sous-expression générale (v0.4, exigerait des temporaires IR)."
	status: "landed"
	notes:  "Fixture testdata/phpt/ternary.phpt ; corpus ternary.php refused → ok. Nesting non parenthésé refusé (déprécié PHP 8)."
}
