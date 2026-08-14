package findings

F_sgoiter_agy_session_dd7965: #Finding & {
	id:     "F-sgoiter-agy-session-dd7965"
	kernel: "monocypher_aead"
	stage:  "emit"
	symptom: "Session AGY Gemini 3.6 High dd7965 : ≥62 patches atomiques emit/front monocypher; build AEAD encore FAIL; isRead final casse compile outil."
	evidence: {
		file_line: "spec/findings/FIXLOG_agy_dd7965_20260810.md; brain dd7965da transcript 646 events"
		kat:       "fail"
		source_doc: "FIXLOG_agy_dd7965_20260810.md"
	}
	lever:  "emit"
	action: "Lire FIXLOG A1-A62. Revert/fix isRead. Puis résidus pad/H-store/ptrmeta-alias. Ne pas sous-compter les landed-wip."
	status: "codified"
	notes:  "Diff WIP +1106L. Landed invariants: type-before-assign, call-rettype, field-regname, arg-slice, param-ptrmeta, isSimpleBase, EqualFold, opload scalar."
}
