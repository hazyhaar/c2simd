package findings

F_sgoiter_field_regname_alias: #Finding & {
	id:     "F-sgoiter-field-regname-alias"
	kernel: "ctx.H|ctx.R"
	stage:  "emit"
	symptom: "OpField sans regName → hoist var v191 + assign ctx.H impossible; index sur scalaire."
	evidence: {
		file_line: "emit OpField regName=rhs; hoist skip alias; AGY L347 L255"
		kat:       "n/a"
		source_doc: "FIXLOG_agy_dd7965_20260810.md"
	}
	lever:  "emit"
	action: "regName[dst]=base.Field; hoist ignore si name≠vN."
	status: "landed"
	notes:  "landed-wip. FIXLOG A15 A23."
}
