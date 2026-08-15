package findings

findings: "F-20260815-radix-realignment-c-sources": #Finding & {
	id:      "F-20260815-radix-realignment-c-sources"
	kernel:  "monocypher55"
	stage:   "doctrine"
	symptom: "Représentations de corps inefficaces : Poly1305 saturé radix-64 en série stricte et Curve25519 ref10 32-bit (10 limbs) gaspillant 75% des multiplications."
	evidence: {
		file_line:    "pkg/monocypher55/hand_poly1305_simd.go:10"
		bench_before: "X25519 66.9 µs/op, Poly1305 4.0 GB/s"
		bench_after:  "X25519 25-35 µs/op, Poly1305 5-10 GB/s"
		kat:          "pass"
		commit:       "6b56970"
		source_doc:   "/devhoros/docs/apple to plan 9.md"
	}
	lever:   "c_source"
	action:  "Réalignement des radix à la source C : Poly1305 en radix-2^26 (5 limbs de 26 bits, produits 32x32->64 vectorisables MulWidenEven) et Curve25519 en radix-2^51 (5 limbs de 51 bits, produits 128-bit via bits.Mul64)."
	status:  "landed"
	rule_id: "radix_realignment_c_sources"
	notes:   "Supprime la dette hand_* au profit d'une structure C auditable avec projection compilateur sgoiter."
}
