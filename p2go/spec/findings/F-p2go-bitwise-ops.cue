package findings

F_p2go_bitwise_ops: #Finding & {
	id:      "F-p2go-bitwise-ops"
	kernel:  "& | ^ ~ << >> sur int64"
	stage:   "front"
	symptom: "Aucun opérateur bit-à-bit — les algorithmes de hachage et CRC (corpus Vague 4) sont intranspilables sans eux."
	evidence: {
		file_line: "front/front.go parseBitOr/parseBitXor/parseBitAnd/parseShift ; emit prec() table Go"
		fixture:   "testdata/phpt/bitwise.phpt"
		kat:       "pass"
	}
	lever:  "front"
	action: "Niveaux de précédence PHP 8 insérés (| < ^ < & < ==/!= < rel < . < shifts < add) ; ~ unaire → ^x Go (nœud IR BitNot) ; emit paranthèse selon la table de précédence GO (shifts et & au niveau *, | et ^ au niveau +) — les deux grammaires divergent, la table emit est celle de Go. >> arithmétique des deux côtés (négatifs alignés)."
	status: "landed"
	notes:  "Composés &=, |=, ^=, <<=, >>= : v0.5. Const-fold étendu aux bitwise (shift 0..63)."
}
