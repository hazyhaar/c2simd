package findings

F_sgoiter_writeassign_non_lvalue: #Finding & {
	id:     "F-sgoiter-writeassign-non-lvalue"
	kernel: "poly_blocks"
	stage:  "emit"
	symptom: "ctx.R[:] := … — writeAssign utilise := car declared[dst]==false sur expression field."
	evidence: {
		file_line: "emit writeAssign; AGY L352"
		kat:       "fail"
	}
	lever:  "emit"
	action: "Vérification des expressions d'accès aux champs avec !strings.ContainsAny(nm, '.[]&') pour empêcher l'opérateur := sur les champs de structures."
	status: "landed"
	notes:  "Résolu 2026-08-10 : forçage de l'opérateur = pour les accès aux membres de structures."
}
