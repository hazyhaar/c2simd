package findings

// Finding F-20260820-simsimd-l2sq-vector-kernel
// Intégration et validation bit-exacte du calcul de distance euclidienne carrée float32
// face à gcc -O2 dans tribench.

"F-20260820-simsimd-l2sq-vector-kernel": #Finding & {
	id:       "F-20260820-simsimd-l2sq-vector-kernel"
	kernel:   "simsimd"
	stage:    "ast_opt"
	symptom:  "Validation de conformité bit-exacte face à l'oracle C gcc -O2 pour le calcul de distance L2 carrée"
	evidence: {
		file_line: "tribench/catalog.go:88"
		kat:       "pass"
	}
	lever:   "ast_rule"
	action:  "Abaissement vers tranche Go native float32 et calcul d'accumulateur double précision"
	status:  "landed"
	notes:   "Valide la conformité bit-exacte 30/30 de tribench"
}
