package findings

F_sgoiter_q2_no_generic_simd: #Finding & {
	id:     "F-sgoiter-q2-no-generic-simd"
	kernel: "doctrine"
	stage:  "doctrine"
	symptom: "Tentation de règle IR générique « toute boucle → SIMD / simd.Slice »."
	evidence: {
		kat:         "n/a"
		source_doc:  "sgoiter/SPEC.md §3 ; c2simd F-20260810-q2-generic-simd"
	}
	lever:  "ir_rule"
	action: "Interdit mécanique : aucune RuleDef ne peut cibler une vectorisation générique de boucle. SIMD seulement noyau nommé + tag goexperiment + KAT."
	status: "codified"
	notes:  "Hérite doctrine peer review c2simd Q2."
}
