package findings

F_sgoiter_dogfood_xor_self: #Finding & {
	id:     "F-sgoiter-dogfood-xor-self"
	kernel: "xor_clear"
	stage:  "dogfood"
	symptom: "x^x en C subset reste Xor IR sans simplification."
	evidence: {
		file_line: "sgoiter/rules/rules.go xorSelf ; testdata/c/xor_clear.c"
		kat:       "pass"
	}
	lever:   "ir_rule"
	action:  "RuleDef xor_self : Xor same reg → Const 0."
	status:  "landed"
	rule_id: "xor_self"
}
