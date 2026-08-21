package findings

F_p2go_stdlib_math: #Finding & {
	id:      "F-p2go-stdlib-math"
	kernel:  "abs, min, max, pow, floor, ceil, round, intdiv"
	stage:   "types"
	symptom: "Aucun builtin mathématique — les algorithmes du corpus (bornes, exponentiation) les exigent."
	evidence: {
		file_line: "types/types.go builtinSigs ; emit/builtins.go"
		fixture:   "testdata/phpt/math_builtins.phpt"
		kat:       "pass"
	}
	lever:  "emit"
	action: "Signatures typées int : min/max → builtins Go natifs (1.21+) ; abs/pow → helpers (pow par carrés successifs, exposant négatif = panic fail-loud, domaine float PHP non imité) ; floor/ceil/round = identité sur int (l'écart float PHP n'est pas observable via echo d'un int) ; sqrt REFUSÉ (résultat irrationnel hors subset int)."
	status: "landed"
}
