package findings

F_sgoiter_field_array_slice_arg: #Finding & {
	id:     "F-sgoiter-field-array-slice-arg"
	kernel: "crypto_poly1305_init"
	stage:  "emit"
	symptom: "Load32_le_buf(ctx.R, …) émet ctx.R ([4]uint32) au lieu de ctx.R[:] ; idem Pad."
	evidence: {
		file_line: "emit arg(); mono_aead.go Load32_le_buf(ctx.R)"
		kat:       "fail"
	}
	lever:  "emit"
	action: "Si OpField Imm=ArrayLen ou elemIdx, arg call suffixe [:]. Ne pas confondre array field et scalaire."
	status: "landed"
	notes:  "20260811: arg() ajoute [:] si elemIdx/regPtr et name contient '.'. ctx.R[i] indexe le field; Load32_le_buf reçoit slices. Adjugé landed au sol."
}
