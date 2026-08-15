package findings

findings: "F-20260815-local-aggregate-init": #Finding & {
	id:      "F-20260815-local-aggregate-init"
	kernel:  "monocypher55"
	stage:   "ast_opt"
	symptom: "Perte des initialiseurs d'agrégats locaux (m_inv, k) et troncature de cast 64-bit causant le retour du vecteur nul sur Crypto_x25519_inverse."
	evidence: {
		file_line:    "c2simd/sgoiter/front/front.go:1489"
		bench_before: "0 MB/s (null vector)"
		bench_after:  "27.3 us/op (bit-exact vs GCC)"
		kat:          "pass"
		commit:       "HEAD"
		source_doc:   "/devhoros/c2simd/spec/findings/F-20260815-local-aggregate-init.cue"
	}
	lever:   "ast_rule"
	action:  "Récolte et émission des initialiseurs d'agrégats locaux non nuls dans front/emit, élargissement 64-bit automatique des opérandes de OpMul, et correction de la formule projective Montgomery bb + a24*e."
	status:  "landed"
	rule_id: "local_aggregate_init_and_mul_widen"
	notes:   "Restaure la bit-exactitude et la propriété d'inversion sk*(sk^-1*P)=P vérifiée par oracle C GCC et oracle différentiel ref10 x fe51."
}
