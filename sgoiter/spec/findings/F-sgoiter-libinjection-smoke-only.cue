package findings

// libinjection : surface banc = no_oracle ; CI = smoke gcc driver seulement.
F_sgoiter_libinjection_smoke_only: #Finding & {
	id:     "F-sgoiter-libinjection-smoke-only"
	kernel: "libinjection_sqli"
	stage:  "dogfood"
	symptom: "tribench SkipC : pas de bit-exact sgoiter vs gcc sur libinjection_is_sqli. ci_check compile/exécute libinjection_kat_driver.c (smoke RET/FP) sans comparer l'émis sgoiter."
	evidence: {
		file_line: "tribench/catalog.go SkipC; scripts/ci_check.sh libinjection_kat_driver; testdata/c_kat/libinjection_kat_driver.c"
		kat:       "n/a"
	}
	lever:  "handwrite"
	action: "Statut honnête smoke-only. Oracle bit-exact futur : harvest libinjection_sqli + harness Go vs driver C sur les 6 cas du driver."
	status: "codified"
	notes:  "2026-08-11: ne pas compter libinjection dans un score 12/12 bit-exact."
}
