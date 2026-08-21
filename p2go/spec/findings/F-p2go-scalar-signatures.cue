package findings

F_p2go_scalar_signatures: #Finding & {
	id:      "F-p2go-scalar-signatures"
	kernel:  "function greet(string $qui): string { … }"
	stage:   "types"
	symptom: "Corpus functions_scalar.php refusé — params et retours restaient int-only, les fonctions string étaient inutilisables."
	evidence: {
		file_line: "front/front.go parseFunc (hints) ; types/types.go checkUserCall/ReturnKind"
		fixture:   "testdata/phpt/functions_scalar.phpt"
		kat:       "pass"
	}
	lever:  "types"
	action: "Kinds de signature par HINTS PHP explicites (int/string/array en param, « : type » en retour) — contrat déclaré, pas d'inférence interprocédurale. Arguments vérifiés kind par kind contre la signature ; sans hint, int par défaut (rétro-compatible)."
	status: "landed"
	notes:  "float/bool/callable/object/mixed/iterable → err_parse explicite."
}
