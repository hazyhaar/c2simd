// Leçons AVOID issues du dogfood adversarial + libinjection (défensif), pas d'exploits.
package findings

F_20260810_avoid_patterns_adversarial: #Finding & {
	id:      "F-20260810-avoid-patterns-adversarial"
	kernel:  "doctrine"
	stage:   "ccgo_raw"
	symptom: "Motifs C hostiles au transpile / à la prod Go pure — à documenter comme AVOID post-ccgo."
	evidence: {
		file_line: "spec/dogfood/testdata/cycles/20260810f/adv_*/opt.go ; libinjection_sqli metrics"
		kat:       "n/a"
		source_doc: "spec/dogfood/testdata/cycles/20260810f/REPORT.md"
	}
	lever: "c_source"
	action: """
		AVOID (effets de bord transpile observés) :
		1. Type punning non aligné (*(uint32_t*)p) → survit en __ccgo_up double-deref ;
		   le gen NE sécurise PAS l'alignement ; callers Go doivent copier via encoding/binary.
		2. ROL à distance variable (a<<k)|(a>>(32-k)) avec k runtime → PAS de rewrite bits.* ;
		   laisser SSA ou hand-write.
		3. Gros buffers stack C (ws[4096]) → [4096]byte sur stack Go = risque goroutine stack ;
		   préférer heap tls.Alloc / make pour code servi.
		4. Chaînes d'appels profondes tls-only → OK après T0 point fixe (6/6 élidé adv_tls_depth).
		5. Tables data énormes (libinjection_sqli_data.h → 40k L Go) : bloat ccgo, 305× __ccgo_up ;
		   ne pas embarquer data-heavy sans compact (issue ccgo #46).
		6. Jamais pointer Go natif vers API ccgo (pitfalls research) — alias lab le confirme.
		SECURISATION gen : ne pas « réparer » le punning en silence ; fail-loud ou doc AVOID.
		"""
	status: "codified"
	notes:  "Corpus : lab adv_* + libinjection (BSD, détection SQLi défensive). Exploits/red-team refusés."
}
