// Finding rétro : interdiction de règle AST générique « toute boucle C → SIMD ».
// Dogfood du registre findings — formalise une décision déjà opposable au peer review.
package findings

F_20260810_q2_generic_simd: #Finding & {
	id:      "F-20260810-q2-generic-simd"
	kernel:  "doctrine"
	stage:   "doctrine"
	symptom: "Tentation de règle AST générique « toute boucle C → SIMD » (auto-vectorisation arbitraire)."
	evidence: {
		file_line:  "spec/c2simd_transpiler_2026_peer_review.md:62-65 (Q2) ; :104 (interdiction anti-dérive)"
		source_doc: "spec/c2simd_transpiler_2026_peer_review.md"
		commit:     "490ef3d"
		kat:        "n/a"
		// Preuves empiriques §2.3 (i9-14900K, go1.27rc1 GOEXPERIMENT=simd) :
		// MD5 +4,7 % ; SipHash −4,1 % (SSA native ROLQ) ; BLAKE2b +1,1 % (tls) ; Fast XOR 0,0 % (BW).
		bench_before: "raw ccgo (SipHash 3692 MB/s, XOR 20408 MB/s, BLAKE2b 842 MB/s)"
		bench_after:  "ast opt générique : SipHash 3539 (−4,1 %), XOR 20118 (0 %), BLAKE2b 851 (+1,1 %)"
	}
	// Levier : doctrine (pas un patch C ni une règle gen à ajouter).
	// Classé handwrite au sens du schéma levier fermé A|B|C : les gains SIMD
	// légitimes passent par dispatch fermé de signature de noyau ou hand-write
	// (chacha20_quarter_x4, poly1305_block_16, simd_*.go) — jamais par une
	// règle générique de boucle.
	lever: "handwrite"
	action: """
		Interdiction contractuelle : aucun RuleDef générique « boucle→simd ».
		c2simd-gen reste un dispatch fermé par signature de noyau + passes structurelles
		(rotl, load/store LE, élision tls). La vectorisation de boucles arbitraires
		relève de la passe SSA Go. Tout candidat SIMD float/ANN hors signature fermée
		va en lab hand-written (cmd/c2simd-noncrypto-lab) ou hors c2simd (ex. horosvec fuse).
		"""
	status: "codified"
	notes: """
		Émetteur : dossier d'architecture (cl-ment, 2026-08-06), destinataires pairs
		Grok/Qwen/Kimi. Double occurrence Q2 + bloc anti-dérive. Conséquence registre :
		RuleDef admissibles = structurels (rotl/load LE/tls) ou signature de noyau précise ;
		poly_blocks/chacha20_rounds dans ArchtimeRulesTable sont des pointeurs hand-write,
		pas des rewrites AST auto — à marquer Category handwrite_pointer au prochain nettoyage.
		"""
}
