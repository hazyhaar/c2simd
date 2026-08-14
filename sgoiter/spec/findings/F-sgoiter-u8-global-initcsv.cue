package findings

F_sgoiter_u8_global_initcsv: #Finding & {
	id:     "F-sgoiter-u8-global-initcsv"
	kernel: "blake2b_compress flat sigma"
	stage:  "front"
	symptom: "static const uint8_t sigma[N]={…} non harvesté → expr blake2b_sigma."
	evidence: {
		file_line: "struct harvestGlobalsExtra u8+InitCSV; emit []byte"
		kat:       "pass"
	}
	lever:  "front"
	action: "reArr inclut u8/uint8_t; emit gt=byte."
	status: "landed"
	notes:  "Dogfood blake utilise sigma 1D aplati (pas vrai 2D)."
}
