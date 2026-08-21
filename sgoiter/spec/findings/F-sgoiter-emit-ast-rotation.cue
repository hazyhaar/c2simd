package findings

F_sgoiter_emit_ast_rotation: #Finding & {
	id:      "F-sgoiter-emit-ast-rotation"
	kernel:  "murmur3_lab"
	stage:   "dogfood"
	symptom: "Dérive silencieuse des passes regex d'inlining Rotl/Rotr face aux mutations de types de décalage (r uint8 vs r int8)."
	evidence: {
		file_line:  "c2simd/sgoiter/emit/arch_passes.go:56"
		kat:        "pass"
		source_doc: "c2simd/sgoiter/TODO_TRIBENCH_FINDINGS.md"
	}
	lever:  "emit"
	action: "Inlining de bits.RotateLeft32/64 avec support multi-types (int8/uint8/int32/int) et test d'absence de wrapper résiduel obligatoire."
	status: "landed"
	notes:  "Garantit la disparition totale des wrappers Rotl/Rotr sans régressions de forme."
}
