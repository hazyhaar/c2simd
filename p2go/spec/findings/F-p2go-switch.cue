package findings

F_p2go_switch: #Finding & {
	id:      "F-p2go-switch"
	kernel:  "switch ($v) { case a: … break; case b: case c: … return; default: … }"
	stage:   "front"
	symptom: "switch refusé err_switch — structure de contrôle centrale des ports d'algorithmes (dispatch d'opcodes, machines à états)."
	evidence: {
		file_line: "front/front.go parseSwitch/parseCaseBody ; ir/ir.go Switch ; emit switch Go natif"
		fixture:   "testdata/phpt/switch.phpt"
		kat:       "pass"
	}
	lever:  "front"
	action: "Abaissement vers switch Go natif. Fail-loud : un case non vide se termine par break; (consommé) ou return — le fallthrough IMPLICITE PHP n'est pas imité ; les cases vides s'empilent en case multi-valeurs Go. Sujet int ou string, kinds des cases homogènes."
	status: "landed"
	notes:  "break de boucle hors switch : err_parse explicite (v0.5)."
}
