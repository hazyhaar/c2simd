package findings

// Gemini + sol 2026-08-11 — cjson_number_dogfood.
F_sgoiter_loop_cond_combine: #Finding & {
	id:     "F-sgoiter-loop-cond-combine"
	kernel: "cjson_number_dogfood|*"
	stage:  "emit"
	symptom: "while (i<len && s[i]>='0' && s[i]<='9') → for { if !(i<len){break}; if !(s[i]>='0'){break}; if !(s[i]<='9'){break}; … }"
	evidence: {
		file_line: "emit.go hoistLoopGuards ; cjson émis : for v3 < len_ { … }"
		kat:       "pass"
		bench_after: "emit/loop_guard_test.go TestHoistLoopGuards ; tribench 11/11 compared"
		source_doc: "spec/findings/HARVEST_dogfood_yeux_post_p0p1_20260811.md"
	}
	lever:  "emit"
	action: "Recombiner N gardes if !(C){break} en tête de for{} sans effet de bord avant le premier break manquant → for C1 && C2 && C3 { corps }. Stop si store/call entre gardes."
	status: "landed"
	notes:  "Landed PARTIEL, et la limite est la bonne. Seules les gardes posées directement sous le `for` remontent ; plusieurs gardes consécutives se joignent par &&. Les deux gardes suivantes de cjson restent en place : entre elles vivent `v15 := s[i]` et `v17 := '0'`, que le CORPS consomme (v5 = v5*10 + (v15 - v17)). Les remonter dupliquerait le load ou priverait le corps de sa valeur — ce n'est pas un gain de forme, c'est un changement de programme."
}
