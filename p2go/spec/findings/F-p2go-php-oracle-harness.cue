package findings

F_p2go_php_oracle_harness: #Finding & {
	id:      "F-p2go-php-oracle-harness"
	kernel:  "testdata/algorithms/*.php"
	stage:   "dogfood"
	symptom: "Les EXPECT étaient écrits à la main (v0.1-v0.3) — un EXPECT faux validerait un transpileur faux ; aucun oracle PHP réel n'était branché."
	evidence: {
		file_line: "algo_oracle_test.go TestAlgorithmsVsPhpOracle"
		kat:       "pass"
	}
	lever:  "handwrite"
	action: "Harnais d'oracle vif : chaque algorithme est exécuté par le CLI php 8.3 ET par le Go transpilé, stdout comparés OCTET À OCTET. Fail-loud si php absent. Les fixtures .phpt des algorithmes et cas ciblés sont GÉNÉRÉES depuis la sortie de l'oracle, jamais calculées de tête."
	status: "landed"
	notes:  "Double filet : vecteurs canoniques connus (crc32('123456789') = 3421780262 = 0xCBF43926 constaté) + oracle vivant."
}
