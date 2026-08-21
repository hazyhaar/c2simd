package findings

F_sgoiter_struct_scalar_vs_slice_param: #Finding & {
	id:     "F-sgoiter-struct-scalar-vs-slice-param"
	kernel: "C2_vt_put_cell / struct params"
	stage:  "emit"
	symptom: "Paramètre pointeur de structure typé *Struct au lieu de []Struct quand utilisé en tranche."
	evidence: {
		file_line: "sgoiter/emit/emit.go: walkDecls et emitIns param struct typing"
		kat:       "pass"
	}
	lever:  "emit"
	action: "Distinguer e.scalarPtr pour émettre *Struct uniquement si accès scalaire pur, sinon []Struct."
	status: "landed"
}
