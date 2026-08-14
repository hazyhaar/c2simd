package findings

F_sgoiter_offslot_body_init: #Finding & {
	id:     "F-sgoiter-offslot-body-init"
	kernel: "siphash24 for in!=end"
	stage:  "front"
	symptom: "ensureOffSlot écrivait f.Stmts (perdu); ForInit vide → undefined vreg offSlot."
	evidence: {
		file_line: "front ensureOffSlot → f.Body; parseFor merge ForInit"
		kat:       "pass"
	}
	lever:  "front"
	action: "Toujours init offSlot sur f.Body."
	status: "landed"
}
