package findings

F_sgoiter_blake2b_2d_index: #Finding & {
	id:     "F-sgoiter-blake2b-2d-index"
	kernel: "blake2b_compress"
	stage:  "front"
	symptom: "Dogfood 20260810j : harvest_fail blake2b_compress — err_parse m[blake2b_sigma[r][2*0]] puis err_empty."
	evidence: {
		file_line: "…/20260810j_sgoiter12_fresh/blake2b_compress/sgoiter.err; AGY L144 @22:05-22:26"
		kat:       "fail"
		commit:    "e5c92b8"
		source_doc: "HARVEST_agy_dd7965_2220_2231.md"
	}
	lever:  "front"
	action: "Parse index 2D + expr (table[r][2*i]) ; flatten sigma ou [N][M]T; ne pas skip silencieux toute la fn."
	status: "landed"
	notes:  "20260811: harvest T name[R][C] flat + index name[i][j]→i*C+j. Canon blake2b_compress.c build+smoke OK."
}

