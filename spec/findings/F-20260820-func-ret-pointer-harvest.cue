package findings

// Finding F-20260820-func-ret-pointer-harvest
// Les signatures de fonctions C retournant des pointeurs (ex: uint8_t* func(...) ou
// char *func(...)) n'étaient pas capturées par la regex reFunc de front/front.go,
// provoquant l'absence d'extraction de fonctions internes (stubs appelés non harvestés).
// Résolu par la prise en compte de l'astérisque optionnel dans reFunc et transmission du type pointeur.

"F-20260820-func-ret-pointer-harvest": #Finding & {
	id:       "F-20260820-func-ret-pointer-harvest"
	kernel:   "fastlz"
	stage:    "ast_opt"
	symptom:  "Fonctions retournant des pointeurs omises du parcours et laissées sous forme de stubs non harvestés"
	evidence: {
		file_line: "front/front.go:115"
		kat:       "pass"
	}
	lever:   "ast_rule"
	action:  "Étend reFunc pour matcher les retours pointeurs 'Type *' et 'Type*' et propager la sémantique de tranche/curseur"
	status:  "landed"
	notes:   "Permet la transpilation 100% intégrale et autonome de la bibliothèque FastLZ (fastlz.c)"
}
