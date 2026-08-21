package findings

F_p2go_simd_ascii_case: #Finding & {
	id:      "F-p2go-simd-ascii-case"
	kernel:  "strtoupper($s) / strtolower($s)"
	stage:   "emit"
	symptom: "La bascule de casse ASCII (octet à octet, gros volumes de texte) restait scalaire."
	evidence: {
		file_line: "emit/simd.go (helpers upper/lower) ; emit/builtins.go (enregistrement dual)"
		fixture:   "testdata/phpt/ascii_case_long.phpt"
		kat:       "pass"
	}
	lever:   "emit"
	action:  "L'abaissement du builtin bascule sur helper dual : 32 octets par itération Uint8x32, bande a..z (resp. A..Z) détectée en comparaison SIGNÉE Int8x32 (VPCMPGTB) — les octets UTF-8 ≥ 0x80 sont négatifs donc hors bande et INTACTS, exactement la sémantique ASCII-only de strtoupper PHP ; ±32 appliqué sous masque IfElse ; strings < 32 octets et queue en scalaire."
	status:  "landed"
	rule_id: "simd_ascii_case"
}
