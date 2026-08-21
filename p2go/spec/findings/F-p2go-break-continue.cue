package findings

F_p2go_break_continue: #Finding & {
	id:      "F-p2go-break-continue"
	kernel:  "break; continue; dans while/for/foreach"
	stage:   "front"
	symptom: "break n'existait que comme terminateur de case ; continue était inconnu — les scans avec sortie anticipée étaient inécrivables (v0.5 chantier 2)."
	evidence: {
		file_line: "front/front.go (loopDepth/switchDepth, jumpEscapes) ; ir Break/Continue"
		fixture:   "testdata/phpt/break_continue.phpt"
		kat:       "pass"
	}
	lever:  "front"
	action: "Profondeurs de boucle et de case au parser : break légal en boucle OU switch (le plus proche englobant, sémantiques Go et PHP alignées), continue en boucle seule ; break n/continue n à niveaux refusés. Garde do…while : un break/continue qui s'échapperait du corps dupliqué est refusé (jumpEscapes — un break est capturé par un switch interne, un continue jamais). Sémantique du break-de-switch-dans-boucle vérifiée par l'oracle (xzxz)."
	status: "landed"
	notes:  "L'EXPECT initial écrit de tête était FAUX (xzxzx) — l'oracle php a corrigé (xzxz) : la doctrine EXPECT-depuis-l'oracle a encore payé."
}
