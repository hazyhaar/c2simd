package findings

findings: "F-20260815-archtime-monocypher-pipeline": #Finding & {
	id:      "F-20260815-archtime-monocypher-pipeline"
	kernel:  "monocypher55"
	stage:   "doctrine"
	symptom: "Dynamisme runtime et couplages fragiles dans la chaîne d'émission et de validation Monocypher (boucles de permutation, tables sigma, exponentiations L-2, couplage de passes)."
	evidence: {
		file_line:    "c2simd/sgoiter/emit/emit.go:800"
		bench_before: "1053 MB/s"
		bench_after:  "1692 MB/s"
		kat:          "pass"
		commit:       "81f7e1a"
		source_doc:   "/devhoros/docs/apple to plan 9.md"
	}
	lever:   "ast_rule"
	action:  "Déclarer les exclusions et règles ARCHTIME en tables compilées : pliage des constantes littérales, déroulage de la chaîne d'addition x25519 inverse (L-2), déroulage de la table sigma Blake2b, et élimination des filtres Python au profit de l'exclusion native."
	status:  "landed"
	rule_id: "archtime_monocypher_pipeline"
	notes:   "Élimine 253 branches runtime dans x25519 inverse, 192 bounds checks dans Blake2b et supprime la dépendance python3."
}
