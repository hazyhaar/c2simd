package findings

F_p2go_intdiv_builtin: #Finding & {
	id:      "F-p2go-intdiv-builtin"
	kernel:  "intdiv($a, $b)"
	stage:   "dogfood"
	symptom: "Corpus intdiv.php refusé err_parse (fonction inconnue intdiv) — la division entière canonique PHP 7+ du subset int strict."
	evidence: {
		file_line: "types/types.go builtins ; ir/ir.go lowerCallExpr"
		fixture:   "testdata/phpt/intdiv.phpt"
		kat:       "pass"
	}
	lever:  "ir_rule"
	action: "Table builtins (nom → arité) en types/ ; pliage intdiv(a,b) → Bin{/} à l'IR — PHP intdiv et Go / tronquent tous deux vers zéro, parité exacte."
	status: "landed"
	notes:  "Premier builtin ; strlen/count/abs suivront le même chemin (v0.2/v0.3)."
}
