package findings

// Finding F-20260820-preprocessor-branch-hints-token-pasting
// Les macros d'indices de branchement (likely, unlikely, yyjson_likely, XXH_LIKELY)
// et l'opérateur de collage préprocesseur '##' (ex: 0x7FF00000##UL) provoquaient des erreurs
// de parsing d'expressions dans yyjson et xxhash.
// Résolu par le dépliage direct des branch hints prédéfinis et l'élision de '##' dans foldDefines.

"F-20260820-preprocessor-branch-hints-token-pasting": #Finding & {
	id:       "F-20260820-preprocessor-branch-hints-token-pasting"
	kernel:   "yyjson"
	stage:    "ast_opt"
	symptom:  "err_parse: expr: 0x7FF00000##UL et err_parse: expr: yyjson_likely(...)"
	evidence: {
		file_line: "front/preprocess.go:225"
		kat:       "pass"
	}
	lever:   "ast_rule"
	action:  "Définition d'un dictionnaire universel de branch hints et stripping des opérateurs ## de collage de jetons"
	status:  "landed"
	notes:   "Permet la transpilation sans erreur des calculs arithmétiques 64 bits et flottants de yyjson (bigint, diy_fp)"
}
