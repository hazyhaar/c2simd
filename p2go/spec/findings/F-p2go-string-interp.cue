package findings

F_p2go_string_interp: #Finding & {
	id:      "F-p2go-string-interp"
	kernel:  "\"compte $n unités\""
	stage:   "front"
	symptom: "L'interpolation $ident en double quote — l'idiome string PHP le plus fréquent — était refusée err_interp en bloc."
	evidence: {
		file_line: "front/front.go lexString/emitStringTokens"
		fixture:   "testdata/phpt/strings.phpt"
		kat:       "pass"
	}
	lever:  "front"
	action: "Désucrage AU LEXER : \"a $x b\" émet les tokens ( \"a \" . $x . \" b\" ) et réutilise la chaîne concat complète (types, ItoS, emit). Les formes ${…} et {$…} restent err_interp."
	status: "landed"
	notes:  "Le segment de tête est conservé même vide pour ancrer le kind string de l'expression."
}
