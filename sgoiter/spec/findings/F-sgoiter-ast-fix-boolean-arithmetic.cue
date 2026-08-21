package findings

F_sgoiter_ast_fix_boolean_arithmetic: #Finding & {
	id:     "F-sgoiter-ast-fix-boolean-arithmetic"
	kernel: "astFixBooleanArithmetic"
	stage:  "emit"
	symptom: "(v != 0) << 1 invalide en Go car (v != 0) est un booléen non typé."
	evidence: {
		file_line: "sgoiter/emit/ast_more_passes.go: astFixBooleanArithmetic"
		kat:       "pass"
	}
	lever:  "emit"
	action: "Encapsulation automatique des expressions de comparaison utilisées en opérande arithmétique/décalage sous forme func() int { if cond { return 1 }; return 0 }()."
	status: "landed"
}
