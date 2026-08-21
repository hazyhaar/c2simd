package findings

F_p2go_upper_native_php_gap: #Finding & {
	id:      "F-p2go-upper-native-php-gap"
	kernel:  "strtoupper natif PHP vs helper émis"
	stage:   "dogfood"
	symptom: "Contre-mesure du banc : le strtoupper NATIF de PHP (C vectorisé, ≈15 Go/s) bat les deux voies Go — php 15.3 ms vs go scalaire 171 ms vs go simd 51.7 ms sur 233 Mo. Le PHP interprété ne perd que quand il interprète ; ses builtins C restent des adversaires sérieux."
	evidence: {
		file_line:    "spec/bench/NOTE_BENCH_V04.md (bench_upper.php)"
		bench_before: "go scalaire 1.4 Go/s"
		bench_after:  "go simd 4.5 Go/s (×3.31) — php natif 15.2 Go/s"
		kat:          "pass"
	}
	lever:  "emit"
	action: "v0.5 : déroulage 2×32 octets par itération appliqué et RE-MESURÉ — 51.7 → 48.9 ms (×3.44 vs scalaire, gain ≈5 % seulement). Verdict mesuré : le goulot n'est PAS la boucle vectorielle mais les allocations []byte(s)/string(b) par appel ; les résorber exige une réécriture in-place (analyse de vie de la string source) — marche v0.6."
	status: "landed"
	notes:  "Landed au sens du BANC : mesures avant/après gravées ; l'hypothèse « déroulage suffit » a été testée et INFIRMÉE au banc plutôt qu'affirmée."
}
