package findings

F_sgoiter_simd_ascii_fastpath: #Finding & {
	id:      "F-sgoiter-simd-ascii-fastpath"
	kernel:  "c2vtparser"
	stage:   "dogfood"
	symptom: "Le scanning séquentiel scalaire des octets ASCII imprimables (0x20..0x7E) limite le débit sous les 1.5 Go/s."
	evidence: {
		file_line:  "c2simd/pkg/c2vtparser/parser.go:128"
		kat:        "pass"
		source_doc: "c2simd/sgoiter/TODO_TRIBENCH_FINDINGS.md"
	}
	lever:  "emit"
	action: "Abaissement du prédicat d'intervalle en soustraction vectorielle non-signée Uint8x32 avec détection par masque ToBits()."
	status: "landed"
	notes:  "Permet d'atteindre un débit supérieur à 3.5 Go/s sur flux de logs denses."
}
