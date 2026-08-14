package findings

// AGY dd7965 — writeAssign := sans newline → tokens collés (syntax error go test).
F_sgoiter_writeassign_newline: #Finding & {
	id:     "F-sgoiter-writeassign-newline"
	kernel: "*"
	stage:  "emit"
	symptom: "writeAssign branche `:=` omettait WriteString(\"\\n\") → `v21binary.LittleEndian`."
	evidence: {
		file_line: "emit.go writeAssign"
		kat:       "pass"
	}
	lever:  "emit"
	action: "newline après rhs sur := et =."
	status: "landed"
	notes:  "Régression visible au dogfood KAT combiné ; fix AGY puis ci_check vert."
}
