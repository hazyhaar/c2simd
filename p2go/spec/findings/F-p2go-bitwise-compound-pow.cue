package findings

F_p2go_bitwise_compound_pow: #Finding & {
	id:      "F-p2go-bitwise-compound-pow"
	kernel:  "&= |= ^= <<= >>= ; ** ; min/max variadiques"
	stage:   "front"
	symptom: "Composés bitwise refusés, ** inconnu, min/max limités à 2 arguments (v0.5 chantiers 1, 6, 9)."
	evidence: {
		file_line: "front/front.go multiOps/parsePow ; emit/builtins.go (min/max variadiques)"
		fixture:   "testdata/phpt/bitwise_compound_pow.phpt"
		kat:       "pass"
	}
	lever:  "front"
	action: "Composés ajoutés à la chaîne générique x = x <op> e (ordre du matching par préfixe : <<= avant <<) ; ** parsé PLUS FORT que l'unaire (-2**2 = -(2**2)), associatif à droite, plié en pow(a,b) — exposant négatif = panic p2goPow (domaine float refusé) ; min/max ≥ 2 arguments int, émis sur les builtins Go natifs variadiques. EXPECT vérifié par l'oracle php."
	status: "landed"
}
