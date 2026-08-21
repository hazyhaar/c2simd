package findings

F_p2go_string_concat: #Finding & {
	id:      "F-p2go-string-concat"
	kernel:  "\"a\" . \"b\""
	stage:   "dogfood"
	symptom: "Corpus concat.php refusé err_parse sur l'opérateur point — les strings ne sont pas première classe en v0.1 (confinées à echo)."
	evidence: {
		file_line: "testdata/dogfood/concat.php"
		kat:       "n/a"
	}
	lever:  "types"
	action: "Strings première classe : kind de slot KindStr, « . » et « .= » (précédence PHP 8, sous +/-), opérande int converti strconv.FormatInt (nœud IR ItoS), égalité de strings native, builtin strlen, echo mixte. Truthiness de string en condition REFUSÉE ('' ET '0' falsy PHP — piège non imité)."
	status: "landed"
	notes:  "Fixture testdata/phpt/strings.phpt ; corpus concat.php converti refused → ok."
}
