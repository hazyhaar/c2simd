package findings

F_sgoiter_aead_fused_streaming_256b: #Finding & {
	id:      "F-sgoiter-aead-fused-streaming-256b"
	kernel:  "Crypto_aead_write"
	stage:   "emit"
	symptom: "Séquencement en deux passes distinctes (chiffrement complet ChaCha20 puis hachage complet Poly1305) doublant le trafic mémoire/cache et interdisant le masquage du coût de calcul de la MAC"
	evidence: {
		file_line:    "pkg/monocypher55/chacha20_simd_amd64.go:240"
		bench_before: "1081.77 MB/s"
		bench_after:  "2370.85 MB/s"
		kat:          "pass"
		source_doc:   "pkg/secretstream55/compare_aead_bench_test.go"
	}
	lever:   "handwrite"
	action:  "Combiner le chiffrement ChaCha20 vectorisé et le hachage Poly1305 scalaire 64-bit en micro-entrelacement au sein des 10 double-rounds de chaque chunk de 256 octets pour masquer le temps de calcul Poly1305 par exécution out-of-order sur les ports ALU"
	status:  "landed"
	notes:   "Doctrine Marche 6 SIMD : double-buffering matériel chunk N-1 Poly / chunk N ChaCha20 appliqué dans hand_aead_fused.go"
}
