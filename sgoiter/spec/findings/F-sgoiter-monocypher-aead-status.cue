package findings

F_sgoiter_monocypher_aead_status: #Finding & {
	id:     "F-sgoiter-monocypher-aead-status"
	kernel: "monocypher-4.0.2 AEAD"
	stage:  "dogfood"
	symptom: "Harvest+build+ci OK; KAT 36B+1KB AEAD lock/unlock PASS; ptr+= reslice landed."
	evidence: {
		file_line: "TestMonoAEAD_MultiBlock_1KB; cipher_text=cipher_text[4:] emit"
		kat:       "pass"
		source_doc: "AUDIT_agy_victory_claim_20260810.md"
	}
	lever:  "front"
	action: "Copier mono.go+tests dogfood k si livrable versionné; option KAT vs gcc monocypher."
	status: "landed"
	notes:  "Milestone multi-bloc 2026-08-10. Dogfood aead file peut encore être mince — re-export recommandé."
}
