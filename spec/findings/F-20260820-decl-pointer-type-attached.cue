package findings

// Finding F-20260820-decl-pointer-type-attached
// Le parseur front/front.go échouait sur les déclarations de pointeurs avec astérisque
// collé au type (ex: uint8_t* p, uint64_t* p) ou précédées de qualificateurs (const, unsigned).
// Résolu par normalizeDeclStmt séparant Type* en Type * et mappant les types C étendus.

"F-20260820-decl-pointer-type-attached": #Finding & {
	id:       "F-20260820-decl-pointer-type-attached"
	kernel:   "*"
	stage:    "ast_opt"
	symptom:  "err_parse: lhs: uint8_t* p lors du parsing des déclarations avec astérisque collé ou qualificateurs"
	evidence: {
		file_line: "front/front.go:1472"
		kat:       "pass"
	}
	lever:   "ast_rule"
	action:  "Séparation mécanique de Type* en Type * et extension de isTypeToken / mapType pour les qualificateurs (unsigned, const, static, volatile)"
	status:  "landed"
	notes:   "Débloque le parsing de crypto_argon2 dans Monocypher et des fonctions de lecture flz_* dans FastLZ"
}
