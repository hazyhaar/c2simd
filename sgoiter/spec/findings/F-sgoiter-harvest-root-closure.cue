package findings

F_sgoiter_harvest_root_closure: #Finding & {
	id:     "F-sgoiter-harvest-root-closure"
	kernel: "tweetnacl_dogfood"
	stage:  "front"
	symptom: "Module émis contient helpers/tables hors surface oracle (Sigma*, K, iv, Randombytes, Pack panic) alors que le harnais n’exerce que verify_16/Vn."
	evidence: {
		file_line: "emit/rootclosure.go ; CLI -roots-only et -roots ; `sgoiter -roots vn` sur tweetnacl_dogfood retient 1 fonction sur 13 et aucune des 4 tables"
		kat:       "pass"
		bench_after: "emit/rootclosure_test.go (3 cas) ; tribench 11/11 compared inchangé"
		source_doc: "spec/findings/HARVEST_dogfood_yeux_post_p0p1_20260811.md#T11"
	}
	lever:  "front"
	action: "Option -harvest-roots / défaut = exports + clôture d’appels ; drop globals non atteignables. Stubs panic restent pour refs résiduelles."
	status: "landed"
	notes:  "Le mot-clé `static` du C est relevé sur la source BRUTE : normalize() l'efface avant le parsing, si bien qu'une capture dans la regex de découverte ne voyait rien. Le drapeau reste OPT-IN et le défaut ne bascule pas : les points d'entrée d'un fichier C ne sont pas ceux qu'un harnais exerce — tweetnacl_dogfood n'expose que randombytes, alors que le banc appelle vn, déclarée static. D'où -roots qui nomme les racines. Un jeu de racines qui ne correspond à rien garde le module entier plutôt que d'émettre un fichier vide."
}
