package findings

F_p2go_array_signatures: #Finding & {
	id:      "F-p2go-array-signatures"
	kernel:  "function f(array $a): array { … }"
	stage:   "types"
	symptom: "Les tableaux ne passaient ni en argument ni en retour de fonction — divergence de fond : PHP copie les tableaux par valeur, un slice Go partage."
	evidence: {
		file_line: "types/types.go checkArrWritable/checkArrArg ; ir/ir.go ArrCopy ; emit append([]int64(nil), x...)"
		fixture:   "testdata/phpt/arrays_fn.phpt, testdata/phpt/fail_arrparam_write.phpt"
		kat:       "pass"
	}
	lever:  "types"
	action: "Hint array en param → slice partagé mais paramètre en LECTURE SEULE (toute écriture refusée err_parse : la sémantique de copie n'est pas imitée, elle est gardée) ; hint : array en retour → copie défensive append([]int64(nil), x...) au return (nœud IR ArrCopy)."
	status: "landed"
	notes:  "Copie de tableau var-à-var ($b = $a) reste refusée — v0.4."
}
