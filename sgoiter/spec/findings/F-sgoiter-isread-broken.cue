package findings

F_sgoiter_isread_broken: #Finding & {
	id:     "F-sgoiter-isread-broken"
	kernel: "emit hoist prune"
	stage:  "emit"
	symptom: "isRead utilise s.Instr / s.IfThen / s.IfElse inexistants sur ir.Stmt → sgoiter ne compile plus."
	evidence: {
		file_line: "emit.go isRead AGY L637-641; go build error Instr/IfThen"
		kat:       "fail"
		source_doc: "FIXLOG_agy_dd7965_20260810.md"
	}
	lever:  "emit"
	action: "isRead réécrit sur l'ensemble des formes d'instructions ir.Stmt (SKInstr/SKFor/SKIf/SKSwitch)."
	status: "landed"
	notes:  "Résolu 2026-08-10 : isRead parcourt récursivement les arbres d'instructions et cas de contrôle sans erreur de compilation."
}
