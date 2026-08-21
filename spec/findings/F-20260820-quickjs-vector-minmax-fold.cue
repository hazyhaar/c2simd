package findings

// Finding F-20260820-quickjs-vector-minmax-fold
// Transpilation et conformité bit-exacte du repliement min/max conjoint 64-bit de QuickJS
// face à gcc -O2.

"F-20260820-quickjs-vector-minmax-fold": #Finding & {
	id:       "F-20260820-quickjs-vector-minmax-fold"
	kernel:   "quickjs"
	stage:    "ast_opt"
	symptom:  "Validation différentielle de la double réduction min/max conjointe dans un mot de 64 bits"
	evidence: {
		file_line: "tribench/catalog.go:87"
		kat:       "pass"
	}
	lever:   "ast_rule"
	action:  "Transpilation de l'analyseur min/max vectoriel et encodage compact haut/bas 32-bit"
	status:  "landed"
	notes:   "Valide le score 30/30 de tribench"
}
