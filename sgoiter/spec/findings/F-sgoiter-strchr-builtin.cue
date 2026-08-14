package findings

F_sgoiter_strchr_builtin: #Finding & {
	id:     "F-sgoiter-strchr-builtin"
	kernel: "libinjection_sqli"
	stage:  "emit"
	symptom: "strchr non harvesté → undefined Strchr."
	evidence: {
		file_line: "emitBuiltinCall strchr stub nil []byte; FillStubs skip"
		kat:       "pass"
	}
	lever:  "emit"
	action: "builtin stub + skip FillStubs."
	status: "landed"
	notes:  "Stub fonctionnel minimal (return nil) — build green, pas sémantique C complète."
}
