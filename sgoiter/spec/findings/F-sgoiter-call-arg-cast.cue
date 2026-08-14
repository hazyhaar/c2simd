package findings

F_sgoiter_call_arg_cast: #Finding & {
	id:     "F-sgoiter-call-arg-cast"
	kernel: "store64_le -> store32_le(u64)"
	stage:  "emit"
	symptom: "Appel store32_le(out, in_u64) sans cast → cannot use uint64 as uint32."
	evidence: {
		file_line: "emit/emit.go OpCall + env.callees param types"
		kat:       "n/a"
	}
	lever:  "emit"
	action: "Table des callees du module; cast args scalaires au Type du Param correspondant."
	status: "landed"
}
