package findings

F_p2go_do_while: #Finding & {
	id:      "F-p2go-do-while"
	kernel:  "do { … } while (cond);"
	stage:   "dogfood"
	symptom: "Corpus do_while.php refusé err_parse (statement inattendu \"do\") — construct courant des ports C/PHP."
	evidence: {
		file_line: "front/front.go parseDoWhile ; ir/ir.go lowerStmts (désucrage)"
		fixture:   "testdata/phpt/do_while.phpt"
		kat:       "pass"
	}
	lever:  "front"
	action: "Parse DoWhile au front ; désucrage à l'IR en corps-copie + While (Go n'a pas de do-while, l'abaissement produit des nœuds neufs donc la duplication est sûre)."
	status: "landed"
	notes:  "La condition est vérifiée APRÈS le corps par types/ (les affectations du corps portent la cond)."
}
