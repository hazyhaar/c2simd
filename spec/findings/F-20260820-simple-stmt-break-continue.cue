package findings

// Finding F-20260820-simple-stmt-break-continue
// Les instructions simples "break;" et "continue;" dans les branches d'instructions
// directes (ex: if (*p++ != *q++) break;) étaient rejetées par parseSimpleStmt
// avec err_parse: stmt: break.
// Résolu par l'ajout des opcodes ir.OpBreak / ir.OpContinue et leur émission Go native.

"F-20260820-simple-stmt-break-continue": #Finding & {
	id:       "F-20260820-simple-stmt-break-continue"
	kernel:   "fastlz"
	stage:    "ast_opt"
	symptom:  "err_parse: stmt: break lors du parsing d'une instruction break/continue sans bloc"
	evidence: {
		file_line: "front/front.go:1382"
		kat:       "pass"
	}
	lever:   "ast_rule"
	action:  "Support de break et continue dans parseSimpleStmt et abaissement vers ir.OpBreak / ir.OpContinue"
	status:  "landed"
	notes:   "Permet la transpilation sans stub de la boucle de comparaison de motifs flz_cmp de FastLZ"
}
