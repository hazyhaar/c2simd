package findings

F_sgoiter_hoist_types_priority: #Finding & {
	id:     "F-sgoiter-hoist-types-priority"
	kernel: "emitFunc var declarations"
	stage:  "emit"
	symptom: "var v int émis au lieu de var v bool car e.regType dirty de passe 1 primait sur e.hoistTypes."
	evidence: {
		file_line: "sgoiter/emit/emit.go:1215"
		kat:       "pass"
	}
	lever:  "emit"
	action: "Donner la priorité absolue à e.hoistTypes sur e.regType lors de l'émission des déclarations var."
	status: "landed"
}
