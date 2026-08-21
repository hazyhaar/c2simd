package findings

// Finding F-20260820-c-compound-types-normalization
// Le remplacement naïf du mot-clé unsigned transformait les types composés C
// (ex: unsigned short, unsigned long long) en types hybrides invalides (uint32_t short, uint32_t long).
// Résolu par un pipeline de réécriture ordonné descendant du plus long au plus court (unsigned short int -> uint16_t).

"F-20260820-c-compound-types-normalization": #Finding & {
	id:       "F-20260820-c-compound-types-normalization"
	kernel:   "utf8proc"
	stage:    "ast_opt"
	symptom:  "err_parse: decl: uint32_t short *entry lors du parsing de variables unsigned short"
	evidence: {
		file_line: "front/front.go:219"
		kat:       "pass"
	}
	lever:   "ast_rule"
	action:  "Normalisation ordonnée et exhaustive de tous les types composés signed/unsigned entiers C"
	status:  "landed"
	notes:   "Supprime toutes les ambiguïtés de taille entière sur utf8proc, fastlz, yyjson et stb"
}
