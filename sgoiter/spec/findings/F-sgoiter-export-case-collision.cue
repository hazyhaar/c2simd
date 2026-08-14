package findings

F_sgoiter_export_case_collision: #Finding & {
	id:     "F-sgoiter-export-case-collision"
	kernel: "tweetnacl Sigma0/sigma0"
	stage:  "emit"
	symptom: "exportName lower→Upper collision; goName lower map écrasait Sigma0."
	evidence: {
		file_line: "emit uniqueExportName + goName exact C only; resolveGoName"
		kat:       "pass"
	}
	lever:  "emit"
	action: "uniqueExportName; map exact names only."
	status: "landed"
}
