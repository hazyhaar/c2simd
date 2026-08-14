package findings

F_sgoiter_struct_equal_fold: #Finding & {
	id:     "F-sgoiter-struct-equal-fold"
	kernel: "crypto_poly1305_ctx"
	stage:  "front"
	symptom: "findStruct/fieldOf case-sensitive rataient Crypto_* vs crypto_*."
	evidence: {
		file_line: "struct.go EqualFold; AGY L507 L293"
		kat:       "pass"
		source_doc: "FIXLOG_agy_dd7965_20260810.md"
	}
	lever:  "front"
	action: "EqualFold sur struct name et field name; TestHarvestStructsMono."
	status: "landed"
	notes:  "landed-wip."
}
