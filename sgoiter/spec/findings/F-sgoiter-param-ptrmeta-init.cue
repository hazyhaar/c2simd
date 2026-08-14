package findings

F_sgoiter_param_ptrmeta_init: #Finding & {
	id:     "F-sgoiter-param-ptrmeta-init"
	kernel: "crypto_poly1305_*"
	stage:  "front"
	symptom: "ptrMeta absent sur reg param → field types introuvables."
	evidence: {
		file_line: "front.go parseFunc params ptrMeta[r]=ri; AGY L305"
		kat:       "n/a"
		source_doc: "FIXLOG_agy_dd7965_20260810.md"
	}
	lever:  "front"
	action: "ptrMeta[r]=regInfo à la création de chaque param (+ fallbacks sname regs/Params)."
	status: "landed"
	notes:  "landed-wip. Ne clôt pas F-ptrmeta-alias-lost (rebind scratch pe)."
}
