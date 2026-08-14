package findings

F_sgoiter_shift_lit_bare: #Finding & {
	id:     "F-sgoiter-shift-lit-bare"
	kernel: "*"
	stage:  "emit"
	symptom: "Décalages émis >> uint8(1) / << uint8(16) pour quantités constantes."
	evidence: {
		file_line: "emit.go bareShiftCounts ; crc émis : v2 = (v2 >> 1) ^ 0xedb88320 & -(v2 & 1)"
		kat:       "pass"
		bench_after: "emit/identity_cast_test.go TestBareShiftCounts ; tribench 11/11 compared"
		source_doc: "spec/findings/HARVEST_dogfood_yeux_post_p0p1_20260811.md#T2"
	}
	lever:  "emit"
	action: "Si quantité = littéral entier constant, émettre bare int untyped (>> 1)."
	status: "landed"
	notes:  "Une quantité variable garde sa conversion. Cette passe précède désormais dropIdentityCasts et narrowIndexShiftMask, qui reconnaissent un décalage déjà nu."
}
