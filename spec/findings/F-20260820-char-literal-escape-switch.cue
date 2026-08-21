package findings

// Finding F-20260820-char-literal-escape-switch
// Les étiquettes de cas case contenant des constantes de caractères échappées
// (ex: case '\"': ou case ':':) provoquaient des erreurs de parsing err_parse: case label: '\"'.
// Résolu par l'analyseur de deux-points indexCaseColon et la table d'échappement C complète dans parseIntLit.

"F-20260820-char-literal-escape-switch": #Finding & {
	id:       "F-20260820-char-literal-escape-switch"
	kernel:   "cjson"
	stage:    "ast_opt"
	symptom:  "err_parse: case label: '\"' lors de l'analyse d'un bloc switch avec constantes de caractères"
	evidence: {
		file_line: "front/front.go:3093"
		kat:       "pass"
	}
	lever:   "ast_rule"
	action:  "Décodage exhaustif des séquences d'échappement de caractères C et recherche sans collision du délimiteur de cas"
	status:  "landed"
	notes:   "Permet la transpilation de print_string_ptr et des formateurs d'échappement JSON de cJSON"
}
