package findings

// Finding F-20260820-parse-cond-strip-parens
// Les expressions de conditions entourées de parenthèses (issues d'expansions de macros
// telles que FASTLZ_LIKELY ou d'expressions imbriquées) provoquaient un échec de
// décomposition binaire avec err_parse: expr: (ip.
// Résolu par l'application de stripOuterParens en tête de parseCond.

"F-20260820-parse-cond-strip-parens": #Finding & {
	id:       "F-20260820-parse-cond-strip-parens"
	kernel:   "fastlz"
	stage:    "ast_opt"
	symptom:  "err_parse: expr: (ip lors du parsing d'une condition parenthésée dans while/if"
	evidence: {
		file_line: "front/front.go:1113"
		kat:       "pass"
	}
	lever:   "ast_rule"
	action:  "Application systématique de stripOuterParens au début de parseCond"
	status:  "landed"
	notes:   "Permet la transpilation complète des compresseurs fastlz1_compress et fastlz2_compress"
}
