package findings

F_p2go_foreach: #Finding & {
	id:      "F-p2go-foreach"
	kernel:  "foreach ($a as $i => $v) { … }"
	stage:   "front"
	symptom: "Corpus foreach.php refusé err_array — la boucle idiomatique PHP sur tableaux manquait, seule la forme for indexée passait."
	evidence: {
		file_line: "front/front.go parseForeach"
		fixture:   "testdata/phpt/foreach.phpt"
		kat:       "pass"
	}
	lever:  "front"
	action: "Désucrage AU PARSER en for indexé : compteur gensym p2go_feN (ou la clé $i nommée), $v = $a[compteur] injecté en tête de corps. By-ref (&$v) refusé — muter $v n'écrit pas dans le tableau (sémantique valeur PHP conservée par construction). foreach sur expression non-$var : v0.4."
	status: "landed"
	notes:  "Un foreach de réduction désucré redevient éligible à la règle SIMD simd_sum. array(…) reste err_array : la voie normée est [ … ]."
}
