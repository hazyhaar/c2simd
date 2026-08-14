package findings

F_sgoiter_call_rettype_callee: #Finding & {
	id:     "F-sgoiter-call-rettype-callee"
	kernel: "Load32_le|any call"
	stage:  "emit"
	symptom: "OpCall typait dst avec e.dom → uint64 slot pour Load32_le uint32."
	evidence: {
		file_line: "emit.go OpCall retType=callees[sym].Result; AGY L207"
		kat:       "n/a"
		source_doc: "FIXLOG_agy_dd7965_20260810.md"
	}
	lever:  "emit"
	action: "retType depuis table callees (+ ToLower fallback) avant writeAssign."
	status: "landed"
	notes:  "landed-wip. FIXLOG A6-A7."
}
