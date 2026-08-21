package findings

F_sgoiter_shortcircuit_and: #Finding & {
	id:      "F-sgoiter-shortcircuit-and"
	kernel:  "court-circuit && et charges conditionnelles"
	stage:   "emit"
	symptom: "En C: base + x < width && (row[base + x] & 0x80). Le Go émis chargeait row[v23] avant le test de borne, causant un panic index out of range si la tranche était courte."
	evidence: {
		file_line: "sgoiter/emit/emit.go:emitIfShortCircuit (validé par emit/shortcircuit_test.go)"
		kat:       "pass"
	}
	lever:  "emit"
	action: "Détection des chaînes && avec dépendances de charge (OpLoad); émission de blocs conditionnels imbriqués ou de sas booléen n'évaluant l'opérande droit que si le gauche est vrai."
	status: "landed"
}
