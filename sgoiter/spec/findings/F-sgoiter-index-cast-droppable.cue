package findings

// Audit profond 2026-08-11 — C16. Angle neuf Q10 : déposer int( au site d'index sans retyper.
F_sgoiter_index_cast_droppable: #Finding & {
	id:     "F-sgoiter-index-cast-droppable"
	kernel: "fnv1a_64|base64_simd|blake2b_compress|*"
	stage:  "emit"
	symptom: "La spec Go accepte tout type entier comme index : data[v4] avec v4 uint64 est légal. Le wrapper int( au site d'index est déposable SANS le retypage de registre que Q10 documente. Pire : sur plateforme 32 bits, int(v4) tronque silencieusement là où data[v4] paniquerait — le wrapper actuel dégrade la sûreté."
	evidence: {
		file_line: "spec/dogfood/testdata/cycles/20260811_audit_fable/fnv1a_64/out.go:10 ; site d'enveloppe emit/emit.go:2150, dropSelfCasts emit/emit.go:2156"
		kat:       "pass"
		source_doc: "spec/findings/HARVEST_audit_profond_20260811.md#C16"
	}
	lever:  "emit"
	action: "Ne plus envelopper les index de type entier non signé (émettre data[v4] nu) ; réserver le retypage Q10 aux registres où il apporte la forme. Attention couplage : les passes postinc (emit.go:2246) et T16 (emit.go:2282) matchent textuellement int( — les adapter d'abord. Métrique : grep -o 'int(' par noyau (blake2b 40, base64 21)."
	status: "proposed"
	notes:  "Complète Q10 (TODO_NEXT) sans le remplacer : Q10 reste le chantier de fond du typage des registres d'induction."
}
