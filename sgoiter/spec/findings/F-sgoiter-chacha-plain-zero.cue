package findings

F_sgoiter_chacha_plain_zero: #Finding & {
	id:     "F-sgoiter-chacha-plain-zero"
	kernel: "crypto_chacha20_djb"
	stage:  "front"
	symptom: "Émis: v7=zero toujours; if v7!=nil toujours vrai (zero non-nil). XOR keystream avec zéros, ignore plain_text. C: plain_text=zero seulement si NULL sur dernier bloc."
	evidence: {
		file_line: "mono.go Crypto_chacha20_djb v7=zero; monocypher_amalg.c:569-578"
		kat:       "fail"
		source_doc: "HARVEST_agy_dd7965_postCP5_buildOK_katFAIL.md"
	}
	lever:  "front"
	action: "NULL pointeur → nil []byte (pas global zero). Protection des paramètres regName contre écrasement. KAT AEAD 100% PASS."
	status: "landed"
	notes:  "Résolu 2026-08-10 : garde !e.isParam[Dst] dans ptr_alias + préservation de ri.v lors des affectations."
}
