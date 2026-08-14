package findings

F_sgoiter_issimplebase: #Finding & {
	id:     "F-sgoiter-issimplebase"
	kernel: "(ctx->h)[i]|p[i]"
	stage:  "front"
	symptom: "Index sur bases non-ident pures refusés ou mal parsés → Add/slice foireux."
	evidence: {
		file_line: "front isSimpleBase; AGY L480-483"
		kat:       "n/a"
		source_doc: "FIXLOG_agy_dd7965_20260810.md"
	}
	lever:  "front"
	action: "isSimpleBase: ident | a->b | a.b (parens trim)."
	status: "landed"
	notes:  "landed-wip. AGY: 'operator precedence / invalid operation' largement réduits."
}
