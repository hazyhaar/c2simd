package findings

F_sgoiter_front_struct_blocklist: #Finding & {
	id:      "F-sgoiter-front-struct-blocklist"
	kernel:  "monocypher_front"
	stage:   "front"
	symptom: "Rejet direct dans front.go:209 des structures composites bloquant l'ingestion automatique de crypto_argon2 et crypto_eddsa_check_equation."
	evidence: {
		file_line: "front.go:209"
		kat:       "fail"
	}
	lever:  "front"
	action: "Étendre l'ingestion d'IR du front-end pour traiter les blocs de structures composites imbriquées sans liste d'exclusion manuelle."
	status: "proposed"
}
