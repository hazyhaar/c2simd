package findings

// Finding F-20260820-deref-post-pre-inc
// Les expressions C combinant déréférencement et incrément de pointeur (*p++ et *++p)
// étaient abaissées en déréférencement direct d'un registre scalaire entier (*v69),
// produisant une erreur de compilation Go "cannot indirect v69 (variable of type uint64)".
// Résolu par le traitement explicite de *p++ et *++p dans parseExpr avec chargement indexé
// par le registre d'offset offSlot (p[oldv] ou p[slot]).

"F-20260820-deref-post-pre-inc": #Finding & {
	id:       "F-20260820-deref-post-pre-inc"
	kernel:   "miniz"
	stage:    "ast_opt"
	symptom:  "cannot indirect v69 (variable of type uint64) lors du déréférencement avec incrément *p++"
	evidence: {
		file_line: "front/front.go:2460"
		kat:       "pass"
	}
	lever:   "ast_rule"
	action:  "Dépouillement des formes *p++ et *++p avec émission de chargements indexés p[offSlot] et incréments d'offsets"
	status:  "landed"
	notes:   "Valide la conformité bit-exacte 25/25 face à gcc -O2 de miniz_adler32"
}
