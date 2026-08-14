package findings

// Audit profond 2026-08-11 — C5. Caches de valeurs pointées réutilisés après écritures.
F_sgoiter_alias_cache_across_store: #Finding & {
	id:     "F-sgoiter-alias-cache-across-store"
	kernel: "chacha20_qr"
	stage:  "emit"
	symptom: "L'émis met en cache les lectures pointeur (v4 := *b, out.go:7) et les réutilise après des écritures mémoire (out.go:15,18,19,21,22) là où le C relit *b. Sans restrict côté C, un appel aliasé (qr(&x[0],&x[0],…)) diverge."
	evidence: {
		file_line: "spec/dogfood/testdata/cycles/20260811_audit_fable/chacha20_qr/out.go:7-22"
		kat:       "pass"
		source_doc: "spec/findings/HARVEST_audit_profond_20260811.md#C5"
	}
	lever:  "emit"
	action: "Invalider les caches de déréférencement à chaque store potentiellement aliasé (ou ne cacher que sous preuve de non-aliasing). Oracle : test quarter-round avec deux paramètres identiques, comparé au C."
	status: "proposed"
	notes:  "Le banc est bit-exact uniquement parce que le double-round appelle quatre adresses distinctes. Angle distinct de T12 (forme de l'ABI)."
}
