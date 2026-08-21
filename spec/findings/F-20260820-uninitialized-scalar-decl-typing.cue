package findings

// Finding F-20260820-uninitialized-scalar-decl-typing
// Les variables scalaires déclarées sans initialiseur (ex: uint64_t i;) n'émettaient aucune instruction IR,
// ce qui provoquait l'inférence erronée du type dominant flottant de la fonction (e.dom, ex: float32)
// lors de la réassignation dans une boucle for (i = 0), produisant des erreurs de type Go sur les index de tableaux.
// Résolu par l'émission systématique de l'instruction ir.OpMov avec Sym: "decl_uninit" et typage strict.

"F-20260820-uninitialized-scalar-decl-typing": #Finding & {
	id:       "F-20260820-uninitialized-scalar-decl-typing"
	kernel:   "simsimd"
	stage:    "ast_opt"
	symptom:  "invalid argument: index v5 (variable of type float32) must be integer sur boucles avec variables déclarées en amont"
	evidence: {
		file_line: "front/front.go:1732"
		kat:       "pass"
	}
	lever:   "ast_rule"
	action:  "Émission de decl_uninit pour fixer le type déclaré des registres scalaires sans initialiseur"
	status:  "landed"
	notes:   "Garantit que les compteurs de boucle i restent uint64 indépendamment du type de retour flottant de la fonction"
}
