package findings

F_sgoiter_narrow_shift_mask: #Finding & {
	id:     "F-sgoiter-narrow-shift-mask"
	kernel: "base64_simd"
	stage:  "emit"
	symptom: "b64_table[int(uint64(v12) >> uint8(18) & uint64(63))] alors que v12 est uint32 — élargissements redondants."
	evidence: {
		file_line: "emit.go narrowIndexShiftMask ; base64 émis : b64_table[int((v12 >> 18) & 63)] sur les sept sites"
		kat:       "pass"
		bench_after: "emit/postinc_narrow_test.go TestNarrowIndexShiftMask ; tribench 11/11 compared"
	}
	lever:  "emit"
	action: "Si shift+mask sur valeur déjà ≤32 bits et résultat index : (v12 >> 18) & 63 puis int(…). Combinable T1/T2."
	status: "landed"
	notes:  "Élargir avant un décalage droit ne peut pas faire apparaître de bits : la réécriture est sûre tant que le décalage reste sous la largeur déclarée de la valeur, ce que la passe vérifie. Un décalage au-delà (>= 32 sur un uint32) garde la forme élargie."
}
