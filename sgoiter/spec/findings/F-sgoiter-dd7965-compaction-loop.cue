package findings

F_sgoiter_dd7965_compaction_loop: #Finding & {
	id:     "F-sgoiter-dd7965-compaction-loop"
	kernel: "monocypher_aead emit"
	stage:  "emit"
	symptom: "5 compactions (CP0-5) même brief « reprends » ; uint8→H uint32 revient CP1-CP3 ; build AEAD jamais vert ; martelage emit/front."
	evidence: {
		file_line: "brain dd7965 CHECKPOINT 0-5; DIAG_dd7965_5compactions.md"
		kat:       "fail"
		source_doc: "DIAG_dd7965_5compactions.md"
	}
	lever:  "emit"
	action: "Stop agentic thrash. Oracle: go build AEAD filtré OU golden IR poly H-store. Un finding=un test. Recharger FIXLOG avant reprise."
	status: "codified"
	notes:  "CP1-3 rond types. Post-CP5: BUILD OK + KAT FAIL (voir F-aead-build-ok-kat-fail) — sort du rond compile, entre phase sémantique."
}

