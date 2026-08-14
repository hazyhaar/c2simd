package findings

F_sgoiter_unsigned_strip: #Finding & {
	id:     "F-sgoiter-unsigned-strip"
	kernel: "poly_blocks unsigned end"
	stage:  "front"
	symptom: "normalize strip \\bunsigned\\b laisse 'end' sans type → param cassé."
	evidence: {
		file_line: "front/front.go normalize; poly_blocks(..., unsigned end)"
		kat:       "n/a"
	}
	lever:  "front"
	action: "Avant strip: unsigned int→uint32_t, unsigned char→uint8_t, bare unsigned→uint32_t; volatile strip."
	status: "landed"
}
