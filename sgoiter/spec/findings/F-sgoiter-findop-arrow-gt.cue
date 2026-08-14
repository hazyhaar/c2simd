package findings

// Footgun findOp: matching binary ops inside C digraphs/trigraphs-like tokens.
F_sgoiter_findop_arrow_gt: #Finding & {
	id:     "F-sgoiter-findop-arrow-gt"
	kernel: "monocypher|ctx->field"
	stage:  "front"
	symptom: "ctx->c_idx parse empty expr: findOp matches '>' inside '->' (and '<' inside '<<')."
	evidence: {
		file_line: "front/front.go findOp; TestPeDirect2 ctx->c_idx"
		kat:       "pass"
	}
	lever:  "front"
	action: "findOp: skip '-' if next '>'; skip '>' if prev '-'; skip '<' inside << <=; same | &."
	status: "landed"
	notes:  "Sans ce garde, tout accès champ monocypher échoue silencieusement en empty expr."
}
