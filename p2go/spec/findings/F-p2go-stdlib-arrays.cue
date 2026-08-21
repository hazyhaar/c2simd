package findings

F_p2go_stdlib_arrays: #Finding & {
	id:      "F-p2go-stdlib-arrays"
	kernel:  "array_push, array_pop, array_reverse, array_slice, array_fill, in_array"
	stage:   "types"
	symptom: "Aucune opération de tableau — pile explicite (quicksort itératif) et manipulations du corpus bloquées."
	evidence: {
		file_line: "types/types.go (array_push statement, array_pop RHS direct) ; ir ArrPush/ArrPop ; emit/builtins.go"
		fixture:   "testdata/phpt/arrays_builtins.phpt"
		kat:       "pass"
	}
	lever:  "types"
	action: "Les MUTATIONS sortent du monde expression : array_push = statement seul (append), array_pop = RHS direct d'affectation (2 lignes émises) — jamais enfouies dans une expression. reverse/slice/fill retournent des COPIES ; in_array retourne bool, contexte condition seul ; array_fill exige start littéral 0 (clefs denses). array_map REFUSÉ : callables hors subset, boucle explicite exigée."
	status: "landed"
}
