package findings

F_sgoiter_iife_bool_ternary_elim: #Finding & {
	id:      "F-sgoiter-iife-bool-ternary-elim"
	kernel:  "monocypher_slide_chacha"
	stage:   "emit"
	symptom: "Conjonctions logiques et conditions ternaires émises sous forme d'invocations de fonctions anonymes imbriquées (IIFE) détruisant le court-circuitage."
	evidence: {
		file_line: "monocypher_utils.go:1029, chacha20.go:207"
		kat:       "pass"
	}
	lever:  "emit"
	action: "Réécrire les expressions booléennes et conditions en opérateurs natifs &&/|| et embranchements directs sans encapsulation IIFE."
	status: "proposed"
}
