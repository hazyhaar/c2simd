package findings

F_p2go_arrays_v02: #Finding & {
	id:      "F-p2go-arrays-v02"
	kernel:  "$a = [1,2,3]; $a[$i] ; count($a)"
	stage:   "types"
	symptom: "Sans tableaux, aucune donnée vectorisable : la marche SIMD (Jalon 4) était bloquée par le subset scalaire pur."
	evidence: {
		file_line: "types/types.go SlotKind ; ir/ir.go ArrAssign/Index/Count ; emit/emit.go"
		fixture:   "testdata/phpt/array_sum.phpt"
		kat:       "pass"
	}
	lever:  "types"
	action: "Second kind de slot (KindArr → []int64 local) : littéral en RHS d'un = simple, lecture/écriture indexée, count() ; un slot ne change jamais de kind (fail-loud int ↔ array) ; ni param, ni retour, ni copie de tableau (v0.3)."
	status: "landed"
	notes:  "foreach reste err_array : la voie normée du subset est [ … ] + for indexé."
}
