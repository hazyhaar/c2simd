package findings

F_sgoiter_simd_direct_projections: #Finding & {
	id:      "F-sgoiter-simd-direct-projections"
	kernel:  "chacha20_djb_simd_4x"
	stage:   "emit"
	symptom: "Déversement des registres vectoriels Uint32x8 sur la pile via StoreArray et relecture scalaire élément par élément pour les transpositions de matrice ChaCha20, induisant 21% de temps CPU en accès pile L1"
	evidence: {
		file_line:    "pkg/monocypher55/chacha20_simd_amd64.go:110"
		bench_before: "1941.02 MB/s"
		bench_after:  "2821.77 MB/s"
		kat:          "pass"
		source_doc:   "pkg/secretstream55/stage_bench_test.go"
	}
	lever:   "handwrite"
	action:  "Émettre directement les projections vectorielles Uint32x8 sans StoreArray, coupler les rotations ARX 16/8 bits avec vpshufb (PermuteOrZeroGrouped) et maintenir le compteur CTR dans un accumulateur vectoriel pour éliminer tout accès mémoire résiduel"
	status:  "landed"
	notes:   "Doctrine Marche 6 SIMD : débit ChaCha20 pur à 2,82 Go/s (2,71× l'assembleur non-fused de x/crypto)"
}
