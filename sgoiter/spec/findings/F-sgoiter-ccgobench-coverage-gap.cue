package findings

F_sgoiter_ccgobench_coverage_gap: #Finding & {
	id:     "F-sgoiter-ccgobench-coverage-gap"
	kernel: "monocypher_aead|sgoiter findings"
	stage:  "dogfood"
	symptom: "Banc AGY pkg/ccgobench (5 piliers) existe mais tests dummy + CheckDeterminism=ccgo ; n'oppose aucun finding emit monocypher. Résidus AEAD hors ci_check."
	evidence: {
		file_line: "pkg/ccgobench/*; sgoiter/scripts/ci_check.sh; COVERAGE_banc_vs_findings_20260810.md"
		kat:       "n/a"
		source_doc: "COVERAGE_banc_vs_findings_20260810.md"
	}
	lever:  "emit"
	action: "Gate dogfood k: filter AEAD→emit→go build en CI; KAT load64/chacha_h; option adapter ccgobench CheckDeterminism→sgoiter; struct layout poly ctx; IR assert 0 stub aead."
	status: "landed"
	notes:  "20260811: opposition réelle = ci_check (labs+murmur+mono AEAD 1KB+parity monocypher_sgoiter) + go test ./sgoiter (kernels KAT, blake2d, mem). ccgobench reste méthodo optionnelle, pas bloquant."
}
