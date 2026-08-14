// Finding rétro : rotl32 → bits.RotateLeft32 + purge DeadCode de la déf. rotl32.
package findings

F_20260810_rotl32_bits: #Finding & {
	id:      "F-20260810-rotl32-bits"
	kernel:  "*"
	stage:   "ast_opt"
	symptom: "Appels rotl32(tls, x, n) / déf. rotl32 issus de ccgo : indirection + param tls mort, empêche ROLQ natif."
	evidence: {
		file_line: "internal/astmatch/astmatch.go (appel rotl32/64) ; rules.go Symbol=rotl32 DeadCode=true"
		kat:       "pass"
		// SipHash §2.3 : après opt AST parfois −4 % car SSA Go reconnaît déjà ROLQ —
		// la règle reste correcte (forme idiomatique) même si le gain n'est pas universel.
		bench_before: "rotl32(tls,x,n) wrapper ccgo"
		bench_after:  "bits.RotateLeft32(x, int(n)) ; déf. rotl32 purgée"
	}
	lever:   "ast_rule"
	action:  "RuleDef rotl32 Kind=rewrite : réécriture des appels + DeadCode sur la définition. rotl64 idem (appel, pas d'entrée table séparée)."
	status:  "landed"
	rule_id: "rotl32"
}
