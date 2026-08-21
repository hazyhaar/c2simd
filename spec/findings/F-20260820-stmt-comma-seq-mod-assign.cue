package findings

// Finding F-20260820-stmt-comma-seq-mod-assign
// Les séquences d'instructions C séparées par des virgules (ex: s1 += ptr[0], s2 += s1;)
// et les opérateurs d'assignation composée avec division/modulo (/=, %=) provoquaient
// des échecs de parsing (err_parse: lhs: s1 % et err_parse: expr: 8, ptr += 8).
// Résolu par le découpage splitCSV au niveau 0 dans parseSimpleStmt et l'intégration de /= et %= dans les tables binop.

"F-20260820-stmt-comma-seq-mod-assign": #Finding & {
	id:       "F-20260820-stmt-comma-seq-mod-assign"
	kernel:   "miniz"
	stage:    "ast_opt"
	symptom:  "err_parse: lhs: s1 % lors du parsing d'une instruction d'assignation modulo %= ou d'une séquence à virgules"
	evidence: {
		file_line: "front/front.go:1408"
		kat:       "pass"
	}
	lever:   "ast_rule"
	action:  "Découpage des séquences d'instructions à virgules et support de /= et %= dans les tables binop"
	status:  "landed"
	notes:   "Permet la transpilation intégrale de mz_adler32 et mz_crc32 de la bibliothèque miniz"
}
