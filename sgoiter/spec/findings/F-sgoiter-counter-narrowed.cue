package findings

// Audit profond 2026-08-11 — C11. Compteur size_t rétréci uint32 : boucle non terminante.
F_sgoiter_counter_narrowed: #Finding & {
	id:     "F-sgoiter-counter-narrowed"
	kernel: "crc32_ieee"
	stage:  "emit"
	symptom: "var v4 uint32 (out.go:6) pour un compteur size_t comparé à len_ uint64 ; la garde for uint64(v4) < len_ (out.go:10) ne termine pas pour len_ ≥ 2^32 (v4 wrappe, uint64(v4) plafonne à 0xFFFFFFFF)."
	evidence: {
		file_line: "spec/dogfood/testdata/cycles/20260811_audit_fable/crc32_ieee/out.go:5-10 ; front.go:2819 mappe pourtant size_t→uint64 ; hoist/declType emit/emit.go:493-496, 918-920"
		kat:       "pass"
		source_doc: "spec/findings/HARVEST_audit_profond_20260811.md#C11"
	}
	lever:  "emit"
	action: "Mesurer le point de perte exact (déclaration sans initialiseur → declType vide → type dominant uint32 ?) avant patch, puis faire suivre au compteur le type du front. Oracle : boucle synthétique len_ = 2^32+1, timeout."
	status: "proposed"
	notes:  "Cause non localisée au sol — le suspect (fallback type dominant) est nommé mais pas mesuré en face, conformément à la discipline « jamais conclure par élimination »."
}
