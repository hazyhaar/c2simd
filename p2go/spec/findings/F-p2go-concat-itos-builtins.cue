package findings

F_p2go_concat_itos_builtins: #Finding & {
	id:      "F-p2go-concat-itos-builtins"
	kernel:  "$out . substr($alpha, $i, 1)"
	stage:   "ir"
	symptom: "base64_core : les appels substr/chr en opérande de concaténation étaient enveloppés d'un ItoS (strconv.FormatInt sur une string, build_fail) — isStrExpr ignorait les kinds de retour des appels."
	evidence: {
		file_line: "ir/ir.go isStrExpr (cas *front.Call) + curInfo"
		fixture:   "testdata/phpt/algo_base64_core.phpt"
		kat:       "pass"
	}
	lever:  "ir_rule"
	action: "isStrExpr étendu aux appels : builtin retournant string (table types.BuiltinReturnsString) et fonction utilisateur à ReturnKind string (Info du programme exposée au lowering via curInfo)."
	status: "landed"
	notes:  "Capturé PAR le corpus algorithmique — la boucle de dogfooding a fonctionné comme prévu."
}
