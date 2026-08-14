package findings

// Audit profond 2026-08-11 — C7. i64/int64_t rangés sous TypUint64 : l'IR n'a pas de signé large.
F_sgoiter_i64_mapped_unsigned: #Finding & {
	id:     "F-sgoiter-i64-mapped-unsigned"
	kernel: "tweetnacl_dogfood|monocypher"
	stage:  "front"
	symptom: "gf (= i64[16], signé) émis []uint64 aux signatures Neq25519/Par25519 (tweetnacl out.go:120,130) ; les temporaires i64 de fe_frombytes_mask émis uint64 (monocypher l.952-962) avec shifts logiques (l.975) là où le C fait des shifts arithmétiques — faux sur tout limbe négatif (routinier après fe_sub)."
	evidence: {
		file_line: "front/front.go:2819 (mapType) ; monocypher_aead_sgoiter.go:951-977 ; tweetnacl out.go:120,130"
		kat:       "pass"
		source_doc: "spec/findings/HARVEST_audit_profond_20260811.md#C7"
	}
	lever:  "front"
	action: "Introduire TypInt64 dans l'IR et mapper i64/int64_t dessus ; les carries 25519 exigent le shift arithmétique. Oracle : KAT fe_sub puis fe_mul sur limbe négatif, comparé au C."
	status: "landed"
	notes:  "Levé 2026-08-11: ir.TypInt64 + mapType int64_t/i64 ; emit goType int64 ; >> arithmétique Go. Fe_frombytes_mask temps en int64. Parité mono AEAD + FeIsOdd verts. gf tweetnacl encore partiel (stubs Pack)."
}
