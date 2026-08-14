package findings

F_sgoiter_opstore_elem_default_u8: #Finding & {
	id:     "F-sgoiter-opstore-elem-default-u8"
	kernel: "poly_blocks|crypto_poly1305_init"
	stage:  "emit"
	symptom: "ctx.H[i] = uint8(vN) alors que H est [5]uint32 ; Counter = uint8 → uint64. OpStore/arg forcent cast étroit."
	evidence: {
		file_line: "emit/emit.go OpStore ~698-728; OpField default TypUint8; mono_aead.go:1271"
		kat:       "fail"
	}
	lever:  "emit"
	action: "Elem store = type du champ struct layout. Zeroing range sûr pour crypto_wipe."
	status: "landed"
	notes:  "Résolu 2026-08-10 : typage exact conservé et zérotage range sécurisé sur les structures et tranches."
}
