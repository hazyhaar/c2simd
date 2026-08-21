package findings

// Finding F-20260820-preprocessor-autotools-generic-headers
// Les bibliothèques C distribuées avec système Autotools/CMake (ex: PCRE2) référencent
// des en-têtes configurés (config.h, pcre2.h) absents de l'arborescence brute sans exécution préalable de ./configure.
// Résolu par le repli transparent sur les variantes génériques (*.generic) et l'omission gracieuse des en-têtes d'export.

"F-20260820-preprocessor-autotools-generic-headers": #Finding & {
	id:       "F-20260820-preprocessor-autotools-generic-headers"
	kernel:   "pcre2"
	stage:    "ast_opt"
	symptom:  "err_include: include not found: config.h lors de l'ingestion de sources C amont"
	evidence: {
		file_line: "front/preprocess.go:185"
		kat:       "pass"
	}
	lever:   "ast_rule"
	action:  "Découverte automatique des fichiers .generic de repli et tolérance d'omission pour les fichiers de configuration de plateforme"
	status:  "landed"
	notes:   "Permet la transpilation autonome et sans compilation intermédiaire des sources amont de PCRE2"
}
