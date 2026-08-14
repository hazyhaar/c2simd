package findings

F_sgoiter_struct_arrow_monocypher: #Finding & {
	id:     "F-sgoiter-struct-arrow-monocypher"
	kernel: "monocypher-4.0.2 aead/poly/chacha"
	stage:  "front"
	symptom: "poly/aead ctx, COPY/ZERO (ctx->f)[i], reject struct/-> bloquaient harvest AEAD."
	evidence: {
		file_line: "front/struct.go harvestStructs; OpField OpFStore; rejectFuncBody sans ->"
		kat:       "n/a"
		source_doc: "TODO_SECRETSTREAM.md; upstream/monocypher/4.0.2/"
	}
	lever:  "front"
	action: "typedef struct fields; ctx->f / ctx->f[i] / (ctx->f)[i]=; local struct alloca; &ctx; sizeof approx."
	status: "landed"
	notes:  "crypto_aead_lock/unlock/init/write/read + chacha h/djb/x + poly init/final/blocks harvestés; update encore fragile; build Go emit WIP unused globals."
}
