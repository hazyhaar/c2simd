package findings

F_sgoiter_offslot_cursor_reset: #Finding & {
	id:     "F-sgoiter-offslot-cursor-reset"
	kernel: "crypto_chacha20_djb"
	stage:  "front"
	symptom: "ptr+=N via offSlot réinit 0 chaque itération → store/load sur tête de slice; multi-bloc FAIL index 4."
	evidence: {
		file_line: "front.go ptr+= → ptr_add+ptr_alias; emit ptr = ptr[N:]; TestMonoAEAD_MultiBlock_1KB PASS"
		kat:       "pass"
		source_doc: "AUDIT_agy_victory_claim_20260810.md"
	}
	lever:  "front"
	action: "ptr+=N sur []byte: OpAdd Sym=ptr_add puis Mov ptr_alias; emit reslice [N:]. Gate 1KB en ci_check."
	status: "landed"
	notes:  "2026-08-10. Path test front corrigé ../../spec. Sol: 128B chacha + 1KB AEAD PASS."
}
