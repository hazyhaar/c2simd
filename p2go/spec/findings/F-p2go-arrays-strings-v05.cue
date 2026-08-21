package findings

F_p2go_arrays_strings_v05: #Finding & {
	id:      "F-p2go-arrays-strings-v05"
	kernel:  "$b = $a (tableaux) ; foreach (expr as …) ; \"a\" < \"b\""
	stage:   "types"
	symptom: "Copie de tableau var-à-var refusée, foreach limité à $var, comparaisons ordonnées de strings refusées (v0.5 chantiers 4, 5, 8)."
	evidence: {
		file_line: "types/types.go (Assign Var arr, rel str×str) ; ir ArrCopy ; front Block/parseForeach"
		fixture:   "testdata/phpt/tc_arrays_v05.phpt, testdata/phpt/tc_strings_v05.phpt"
		kat:       "pass"
	}
	lever:  "types"
	action: "$b = $a émet une COPIE réelle (ArrCopy, sémantique valeur PHP — muter $b ne touche pas $a, vérifié par l'oracle) ; foreach accepte toute expression tableau, matérialisée dans un temporaire p2go_faN via le nouveau nœud Block (séquence plate aplatie à l'IR) ; < <= > >= sur string×string = ordre lexicographique octet à octet, identique strcmp PHP / Go natif."
	status: "landed"
}
