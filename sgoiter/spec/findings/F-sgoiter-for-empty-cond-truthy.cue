package findings

F_sgoiter_for_empty_cond_truthy: #Finding & {
	id:     "F-sgoiter-for-empty-cond-truthy"
	kernel: "sgoiter/front parseCond"
	stage:  "front"
	symptom: "for (;;) traduit en for 0 != 0 { causant un blocage d'exécution de boucle infinie."
	evidence: {
		file_line: "sgoiter/front/front.go: parseCond"
		kat:       "pass"
	}
	lever:  "front"
	action: "Émission d'un OpConst implicite valant 1 au lieu de 0 pour que la condition émette for {."
	status: "landed"
}
