package findings

// Audit profond 2026-08-11 — C15. Résidus T1/T16 : expressions, résultats d'appel, immédiats.
F_sgoiter_identity_cast_expr: #Finding & {
	id:     "F-sgoiter-identity-cast-expr"
	kernel: "murmur3_x86_32|siphash24|tweetnacl_dogfood|blake2b_compress|base64_simd|fast_xor|poly1305_block5"
	stage:  "emit"
	symptom: "T1 landed ne couvre que les identifiants nus : restent uint32(bits.RotateLeft32(...)) et uint32(Fmix32(v8)) (murmur out.go:10,59), cast au return composite (siphash out.go:170), 9 return uint64(...) (tweetnacl), immédiats IV uint64(0x…) (blake2b), sur-élargissement u64 de l'assemblage base64 (out.go:15,27,29), uint64(7) sur &^ (fast_xor out.go:12), % et / absents de binLitCastPat."
	evidence: {
		file_line: "selfCastPat emit/emit.go:2139 ; binLitCastPat emit/emit.go:2162 ; formatImm emit/emit.go:3402-3409"
		kat:       "pass"
		source_doc: "spec/findings/HARVEST_audit_profond_20260811.md#C15"
	}
	lever:  "emit"
	action: "Étendre le dépôt de casts aux expressions dont le type est déductible (résultat d'appel typé, composition d'opérandes de même type, immédiat en slot typé) ; ajouter % / à l'alternance d'opérateurs. Même famille : hoistLoopGuards (emit.go:2185-2212) sans analyse de dupliquabilité des définitions intercalées pures (fast_xor out.go:9-11)."
	status: "proposed"
	notes:  "Extension directe de F-sgoiter-identity-cast (landed). Compteur décidable : grep -c 'return uint64(' tweetnacl = 9 → 0."
}
