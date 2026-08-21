package findings

F_sgoiter_split_domain_heuristic: #Finding & {
	id:      "F-sgoiter-split-domain-heuristic"
	kernel:  "monocypher_split"
	stage:   "emit"
	symptom: "Sous-routines cryptographiques (Elligator, Slide, G_rounds) déversées par défaut dans monocypher_utils.go par le classifieur modulaire."
	evidence: {
		file_line: "split.go:87, monocypher_utils.go:1025-1258"
		kat:       "pass"
	}
	lever:  "emit"
	action: "Raffiner les règles classifyDecl pour router elligator, slide et argon2 vers des modules de domaine dédiés."
	status: "proposed"
}
