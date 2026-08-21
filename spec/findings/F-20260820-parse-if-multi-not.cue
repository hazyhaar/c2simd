package findings

// Finding F-20260820-parse-if-multi-not
// Les conditions if comportant des négations imbriquées ou parenthésées (issues de
// macros de garde de bornes comme FASTLZ_BOUND_CHECK) provoquaient un échec de parsing.
// Résolu par une boucle de dépouillement alterné des '!' et stripOuterParens dans parseIf.

"F-20260820-parse-if-multi-not": #Finding & {
	id:       "F-20260820-parse-if-multi-not"
	kernel:   "fastlz"
	stage:    "ast_opt"
	symptom:  "err_parse: expr: (ip lors du parsing d'un if avec macro de garde FASTLZ_BOUND_CHECK"
	evidence: {
		file_line: "front/front.go:929"
		kat:       "pass"
	}
	lever:   "ast_rule"
	action:  "Déplier et inverser la négation logique à travers les niveaux de parenthèses de la condition"
	status:  "landed"
	notes:   "Permet la transpilation complète du décompresseur fastlz1_decompress"
}
