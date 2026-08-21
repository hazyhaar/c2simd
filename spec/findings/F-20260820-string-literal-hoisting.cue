package findings

// Finding F-20260820-string-literal-hoisting
// Les littéraux chaînes de caractères (ex: "null", "%d", "%lg") dans les expressions C
// provoquaient un échec de parsing err_parse: expr: "null".
// Résolu par le hissage automatique des chaînes constantes en symboles globaux d'octets
// avec déduplication par hachage SHA-256 et support dans l'émetteur d'assignation OpMov.

"F-20260820-string-literal-hoisting": #Finding & {
	id:       "F-20260820-string-literal-hoisting"
	kernel:   "cjson"
	stage:    "ast_opt"
	symptom:  "err_parse: expr: \"null\" lors du parsing de littéraux chaînes"
	evidence: {
		file_line: "front/front.go:2232"
		kat:       "pass"
	}
	lever:   "ast_rule"
	action:  "Hissage des littéraux chaînes en symboles globaux constants avec émission Go déclarative"
	status:  "landed"
	notes:   "Permet la transpilation des fonctions d'affichage et de formatage de cJSON (print_number, print_value)"
}
