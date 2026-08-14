package findings

F_sgoiter_kat_lt64_only: #Finding & {
	id:     "F-sgoiter-kat-lt64-only"
	kernel: "crypto_aead_*"
	stage:  "dogfood"
	symptom: "KAT 36B seul insuffisant; multi-bloc cassé jusqu'à fix ptr+= reslice."
	evidence: {
		file_line: "TestMonoAEAD_MultiBlock_1KB; front_test.go path ../../spec"
		kat:       "pass"
		source_doc: "AUDIT_agy_victory_claim_20260810.md"
	}
	lever:  "emit"
	action: "Gate multi-bloc obligatoire (1KB). Ne plus landed AEAD sur KAT <64 seuls."
	status: "landed"
	notes:  "Résolu avec F-offslot-cursor-reset. ci_check invoque MultiBlock_1KB si amalg présent."
}
