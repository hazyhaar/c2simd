package findings

// Finding F-20260820-preprocessor-linear-macro-scan
// Le préprocesseur C foldDefines ré-évaluait le texte entier depuis l'indice 0 après chaque expansion,
// causant des boucles infinies ou un temps quadratique O(N^2) sur des fichiers d'en-tête monolithiques (ex: stb_image.h).
// Résolu par le scan linéaire monotone à progression stricte avec strings.Builder et pré-filtrage strings.Contains.

"F-20260820-preprocessor-linear-macro-scan": #Finding & {
	id:       "F-20260820-preprocessor-linear-macro-scan"
	kernel:   "stb"
	stage:    "ast_opt"
	symptom:  "Blocage d'exécution sur stb_image.h (7800+ lignes) par ré-évaluation quadratique des macros"
	evidence: {
		file_line: "front/preprocess.go:348"
		kat:       "pass"
	}
	lever:   "ast_rule"
	action:  "Scan linéaire monotone d'expansion de macros sans récursion infinie et pré-filtrage de présence"
	status:  "landed"
	notes:   "Réduit le temps de prétraitement de stb_image.h de l'infini à 1,06 seconde"
}
