package findings

F_sgoiter_aead_build_ok_kat_fail: #Finding & {
	id:     "F-sgoiter-aead-build-ok-kat-fail"
	kernel: "monocypher_aead"
	stage:  "dogfood"
	symptom: "Build OK; KAT 36B puis multi-bloc 1KB PASS après fix ptr+= reslice."
	evidence: {
		file_line: "TestMonoAEAD_MultiBlock_1KB PASS; go test front; ci_check monocypher gate"
		kat:       "pass"
		source_doc: "AUDIT_agy_victory_claim_20260810.md"
	}
	lever:  "emit"
	action: "Id historique (kat-fail). Statut landed multi-bloc. Conserver gate 1KB."
	status: "landed"
	notes:  "2026-08-10 multi-bloc clos. Id conservé pour historique."
}
