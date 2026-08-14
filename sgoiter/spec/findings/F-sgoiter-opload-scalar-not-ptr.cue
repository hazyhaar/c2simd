package findings

F_sgoiter_opload_scalar_not_ptr: #Finding & {
	id:     "F-sgoiter-opload-scalar-not-ptr"
	kernel: "any load"
	stage:  "emit"
	symptom: "Après OpLoad, dst restait regPtr → typage slice/cursor faux en aval."
	evidence: {
		file_line: "emit OpLoad regPtr[dst]=false; AGY L469"
		kat:       "n/a"
		source_doc: "FIXLOG_agy_dd7965_20260810.md"
	}
	lever:  "emit"
	action: "Load produit toujours scalaire: regPtr[dst]=false."
	status: "landed"
	notes:  "landed-wip."
}
