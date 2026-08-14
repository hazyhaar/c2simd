package findings

F_sgoiter_hoist_regtype: #Finding & {
	id:     "F-sgoiter-hoist-regtype"
	kernel: "monocypher_aead emit two-pass"
	stage:  "emit"
	symptom: "Hoist pass1 collecte types avant regType final ou type unique faux → var uint32 puis assign byte; slices []uint64 pour cursors byte; call uint32 dans slot uint64."
	evidence: {
		file_line: "emit/emit.go writeAssign hoist + emitFunc two-pass; OpCall order writeAssign before regType; AGY dd7965 RC1"
		kat:       "fail"
		source_doc: "HARVEST_agy_dd7965_20260810.md"
	}
	lever:  "emit"
	action: "Pass1 doit finaliser regType (simuler side-effects types sans I/O) AVANT snapshot hoistTypes; cast assign si T(rhs); cursors byte = []byte. go build AEAD OK."
	status: "landed"
	notes:  "Résolu 2026-08-10 post-AGY : pass1 hoistTypes + resolution des types de registres validés."
}
