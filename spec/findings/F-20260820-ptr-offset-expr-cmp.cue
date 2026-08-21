package findings

// Finding F-20260820-ptr-offset-expr-cmp
// Les comparaisons d'inégalité ou d'ordre entre curseurs et bornes de pointeurs
// (ex: while (p + 8 <= bEnd) ou while (p <= limit)) étaient rejetées par le front
// ou abaissées en comparaisons invalides de tranches Go (slice <= slice).
// Résolu par la fonction parsePtrOffsetExpr prenant en compte les arithmétiques p + N / p - N
// et comparant les registres d'offset de curseurs (offSlots).

"F-20260820-ptr-offset-expr-cmp": #Finding & {
	id:       "F-20260820-ptr-offset-expr-cmp"
	kernel:   "xxhash"
	stage:    "ast_opt"
	symptom:  "invalid operation: input <= input (operator <= not defined on slice)"
	evidence: {
		file_line: "front/front.go:1150"
		kat:       "pass"
	}
	lever:   "ast_rule"
	action:  "Dérivation des expressions d'offset de curseurs via parsePtrOffsetExpr pour tous les opérateurs relationnels"
	status:  "landed"
	notes:   "Valide la transpilation et la conformité bit-exacte 24/24 de xxHash64 (xxhash64_core)"
}
