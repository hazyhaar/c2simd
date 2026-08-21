package findings

F_sgoiter_myers_hunk_partitioning: #Finding & {
	id:      "F-sgoiter-myers-hunk-partitioning"
	kernel:  "c2myers"
	stage:   "dogfood"
	symptom: "L'absence de fenêtrage contextLines fusionnait indistinctement les modifications éloignées en un seul hunk continu."
	evidence: {
		file_line:  "c2simd/pkg/c2myers/diff.go:158"
		kat:        "pass"
		source_doc: "c2simd/sgoiter/TODO_TRIBENCH_FINDINGS.md"
	}
	lever:  "ir_rule"
	action: "Règle de partitionnement et fusion de plages de modifications selon l'intervalle 2*contextLines."
	status: "landed"
	notes:  "Garantit la conformité stricte avec les patchs unifiés git diff -p1."
}
