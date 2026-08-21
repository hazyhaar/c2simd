package findings

// Finding F-20260820-simsimd-scalar-typedefs
// Les bibliothèques mathématiques et de calcul vectoriel amont (ex: SimSIMD)
// utilisent des types scalaires spécifiques (nk_f64_t, nk_f32_t, nk_i32_t, nk_u32_t, nk_size_t).
// Leur absence dans le tableau de reconnaissance de type provoquait l'abandon du parsing des fonctions associées.
// Résolu par l'enrichissement systématique de isTypeToken et mapType et la découverte récursive des en-têtes include/.

"F-20260820-simsimd-scalar-typedefs": #Finding & {
	id:       "F-20260820-simsimd-scalar-typedefs"
	kernel:   "simsimd"
	stage:    "ast_opt"
	symptom:  "err_parse: lhs: nk_f64_t sum lors du parsing des noyaux scalaires SimSIMD"
	evidence: {
		file_line: "front/front.go:3745"
		kat:       "pass"
	}
	lever:   "ast_rule"
	action:  "Enregistrement des types scalaires SimSIMD et extension du graphe de recherche d'en-têtes include"
	status:  "landed"
	notes:   "Permet la transpilation de plus de 1300 lignes de code Go vectoriel et d'états de réduction de SimSIMD"
}
