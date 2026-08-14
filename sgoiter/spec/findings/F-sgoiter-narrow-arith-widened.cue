package findings

// Audit profond 2026-08-11 — C10. Arithmétique/cast 32 bits C élargis en int Go 64 bits.
F_sgoiter_narrow_arith_widened: #Finding & {
	id:     "F-sgoiter-narrow-arith-widened"
	kernel: "poly1305_block5|murmur3_x86_32"
	stage:  "front"
	symptom: "5 * r[k] (unsigned 32 bits, wrappe en C) émis 5 * int(v25) puis uint64(...) sans wrap (poly1305 out.go:15,26 — divergence dès r[k] ≥ 858993460) ; (int)(len / 4) émis int(len_ / 4) sans troncature 32 bits (murmur3 out.go:27 — divergence dès len ≥ 2^33)."
	evidence: {
		file_line: "spec/dogfood/testdata/cycles/20260811_audit_fable/poly1305_block5/out.go:15,18,22,26 ; murmur3_x86_32/out.go:27 ; typage TypInt front/front.go:2809-2828"
		kat:       "pass"
		source_doc: "spec/findings/HARVEST_audit_profond_20260811.md#C10"
	}
	lever:  "front"
	action: "Porter la largeur C de l'expression (usual arithmetic conversions) dans l'IR : un produit unsigned 32 bits s'émet uint32, un cast (int) 32 bits s'émet int32. Oracle : vecteurs hors-domaine (r[k] > 2^29, len 8 Gio) comparés au binaire C."
	status: "proposed"
	notes:  "poly1305 : hors contrat limbes 26 bits (src.c:7), la classe transpileur reste réelle. Les vecteurs du banc ne dépassent jamais ces bornes — invisibles par construction."
}
