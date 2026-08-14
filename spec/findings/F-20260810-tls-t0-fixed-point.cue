// Dogfood chacha20_qr : double_round gardait tls mort (single-pass T0).
package findings

F_20260810_tls_t0_fixed_point: #Finding & {
	id:      "F-20260810-tls-t0-fixed-point"
	kernel:  "chacha20_qr"
	stage:   "ast_opt"
	symptom: "T0 single-pass : leaf stripée mais caller mid/double_round conserve tls car le scan initial voyait encore tls dans les call args."
	evidence: {
		file_line: "internal/astmatch/astmatch.go findUnexportedT0 + elideUnexportedT0 loop"
		kat:       "pass"
		bench_before: "chacha20_qr tls_params_elided=1 (QR only)"
		bench_after:  "tls_params_elided=2 (QR + double_round) ; test tls_t0_fixed_point_caller"
	}
	lever:   "ast_rule"
	action:  "Après Apply principal, boucle point fixe ≤8 : findUnexportedT0 → elideUnexportedT0. Aligné peer review Q4 T1 bottom-up (version itérative)."
	status:  "landed"
}
