package findings

F_sgoiter_poly1305_radix64_reduction: #Finding & {
	id:      "F-sgoiter-poly1305-radix64-reduction"
	kernel:  "Poly_blocks"
	stage:   "emit"
	symptom: "Représentation 32-bit à 5 membres imposant 22 multiplications 64-bit par bloc de 16 octets et créant un goulot de calcul majeur sur le temps total de l'AEAD"
	evidence: {
		file_line:    "pkg/monocypher55/hand_poly1305_simd.go:1"
		bench_before: "2448.57 MB/s"
		bench_after:  "3929.74 MB/s"
		kat:          "pass"
		source_doc:   "pkg/secretstream55/stage_bench_test.go"
	}
	lever:   "handwrite"
	action:  "Écrire Poly_blocks en arithmétique saturée 64-bit (2 limbs de 64 bits + h2), utilisant math/bits.Mul64 et Add64 pour réduire les multiplications de 22 à 6 par bloc de 16 octets"
	status:  "landed"
	notes:   "Doctrine Marche 6 SIMD : abaissement de complexité arithmétique 3.6x pour Poly1305 appliqué dans hand_poly1305_simd.go"
}
