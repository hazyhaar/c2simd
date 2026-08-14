package findings

F_sgoiter_arg_field_slice: #Finding & {
	id:     "F-sgoiter-arg-field-slice"
	kernel: "Load32_le_buf(ctx.R)"
	stage:  "emit"
	symptom: "Array field passé sans [:] à param []T."
	evidence: {
		file_line: "emit arg() elemIdx/regPtr + dot → [:]; AGY L361-453"
		kat:       "n/a"
		source_doc: "FIXLOG_agy_dd7965_20260810.md"
	}
	lever:  "emit"
	action: "arg() ajoute [:] sur field array; OpField n'ajoute plus [:] sur rhs (reste indexable)."
	status: "landed"
	notes:  "landed-wip partial selon ctx. Supersède partiellement F-field-array-slice-arg."
}
