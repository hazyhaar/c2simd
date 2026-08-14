package findings

F_sgoiter_cmp_prefix_variants: #Finding & {
	id:     "F-sgoiter-cmp-prefix-variants"
	kernel: "monocypher conditions"
	stage:  "emit"
	symptom: "Inline cmp seulement __cmp_ ; variantes _cmp_ tombaient en stubs / redecl."
	evidence: {
		file_line: "emit.go OpCall cmp prefix; AGY L229"
		kat:       "n/a"
		source_doc: "FIXLOG_agy_dd7965_20260810.md"
	}
	lever:  "emit"
	action: "Accepter __cmp_ et _cmp_ ; stubs have[exportName]."
	status: "landed"
	notes:  "landed-wip. Lié F-stub-cmp-redecl pour amalg full."
}
