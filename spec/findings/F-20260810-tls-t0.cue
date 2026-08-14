// Finding rétro : élision du paramètre tls *libc.TLS pour fonctions T0 (jamais lu).
package findings

F_20260810_tls_t0: #Finding & {
	id:      "F-20260810-tls-t0"
	kernel:  "*"
	stage:   "ast_opt"
	symptom: "ccgo injecte tls *libc.TLS en premier paramètre partout ; sur fonctions pures le paramètre est mort et pollue registres + sites d'appel (BLAKE2b goulot §2.3)."
	evidence: {
		file_line: "internal/astmatch/astmatch.go P2.1 pureTLSFuncs + strip call args"
		kat:       "pass"
		source_doc: "spec/c2simd_transpiler_2026_peer_review.md:77-84 (Q4 T0/T1/T2)"
		bench_before: "BLAKE2b compress raw 842 MB/s (goulot tls / __ccgo_up)"
		bench_after:  "opt AST +1,1 % — tls T0 seul ne suffit pas si T2 Alloc reste ; T0 strip quand même correct"
	}
	lever:  "ast_rule"
	action: "Passe structurelle (hors RuleDef Symbol) : classer T0 si corps sans ident tls → retirer 1er param + retirer arg tls aux call sites. T1/T2 non encore enchaînés en v2."
	status: "landed"
	notes:  "Pas de rule_id Symbol : passe transverse. Finding ancré Q4 peer review. Suite possible : T1 call-graph bottom-up (finding séparé proposed)."
}
