package findings

F_sgoiter_pad_index_add: #Finding & {
	id:     "F-sgoiter-pad-index-add"
	kernel: "poly_blocks"
	stage:  "front"
	symptom: "ctx->pad[i] devient v + ctx.Pad[:] (Add scalar+slice) au lieu de index/load elem."
	evidence: {
		file_line: "mono_aead.go ~1472 invalid operation v48 + ctx.Pad[:]"
		kat:       "fail"
	}
	lever:  "front"
	action: "Indexation correcte des membres de tableaux de structures en Go (`ctx.Pad[i]`)."
	status: "landed"
	notes:  "Résolu 2026-08-10 : élimination des additions invalides entre scalaires et tranches de champs."
}
