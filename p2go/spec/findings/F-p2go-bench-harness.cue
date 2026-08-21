package findings

F_p2go_bench_harness: #Finding & {
	id:      "F-p2go-bench-harness"
	kernel:  "cmd/p2go-bench, testdata/bench/*.php"
	stage:   "dogfood"
	symptom: "Aucun chiffre : les gains de la transpilation et des règles SIMD étaient affirmés sans mesure (interdit par la doctrine du pôle)."
	evidence: {
		file_line:    "cmd/p2go-bench/main.go ; spec/bench/NOTE_BENCH_V04.md"
		bench_after:  "sum ×30 vs php, simd/scalaire ×2.32 (sum) ×1.39 (dot) ×1.23 (minmax) ×3.31 (ascii-case), i9-14900K 2026-08-20"
		kat:          "pass"
	}
	lever:  "handwrite"
	action: "Banc tri-voies php / go-scalaire / go-simd à toolchain Go IDENTIQUE (go1.27rc3, seul GOEXPERIMENT varie — isole l'effet des règles) ; gate de parité octet-à-octet AVANT toute mesure ; témoins scalaires purs (crc32, fnv1a) à ×1.0 prouvant que le banc ne s'auto-mesure pas ; charges LCG dans le domaine int64 signé PHP."
	status: "landed"
}
