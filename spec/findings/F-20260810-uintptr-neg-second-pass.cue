package findings

F_20260810_uintptr_neg_second_pass: #Finding & {
	id:      "F-20260810-uintptr-neg-second-pass"
	kernel:  "tiny_regex|lz4"
	stage:   "ast_opt"
	symptom: "Après rewrite __ccgo_up, x+uintptr(-N) capturé dans unsafe.Pointer(E) restait non réécrit (constant -1 overflows uintptr) malgré la passe 4b du 1er Apply."
	evidence: {
		file_line: "internal/astmatch/astmatch.go 2e tour BinaryExpr ADD ; tests uintptr_neg_* ; cycles/20260810h tiny_regex+lz4 build OK"
		kat:       "pass"
	}
	lever:   "ast_rule"
	action:  "Second astutil.Apply dédié ADD+matchUintptrNegConst après passes __ccgo_up / index."
	status:  "landed"
	rule_id: "uintptr_neg_second_pass"
}
