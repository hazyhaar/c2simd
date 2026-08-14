package findings

F_sgoiter_stub_cmp_redecl: #Finding & {
	id:     "F-sgoiter-stub-cmp-redecl"
	kernel: "monocypher_amalg full emit"
	stage:  "emit"
	symptom: "__cmp___ redeclared in this block (×N) sur emit amalg non filtré."
	evidence: {
		file_line: "emit/stubs.go FillStubs; mono_aead full ~10250"
		kat:       "n/a"
	}
	lever:  "emit"
	action: "Déduplication automatique des helpers de comparaison synthétiques `__cmp_*` par module dans FillStubs."
	status: "landed"
	notes:  "Résolu 2026-08-10 : déduplication des stubs synthétiques."
}
