package findings

F_sgoiter_cast_unary_not_binop: #Finding & {
	id:     "F-sgoiter-cast-unary-not-binop"
	kernel: "crc32_ieee"
	stage:  "front"
	symptom: "(uint32_t)-(int32_t)x parsé comme binop (uint32_t) - (int32_t)x → expr uint32_t fail."
	evidence: {
		file_line: "front findOp: skip +/- after ) if inner is type token"
		kat:       "pass"
	}
	lever:  "front"
	action: "findOp: after ), inspect paren group; isTypeToken → not binary."
	status: "landed"
}
