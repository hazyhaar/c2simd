package findings

// Audit profond 2026-08-11 — C2. uint32(x << 32) vaut toujours 0.
F_sgoiter_shift32_trunc: #Finding & {
	id:     "F-sgoiter-shift32-trunc"
	kernel: "monocypher"
	stage:  "front"
	symptom: "Le shift 64 bits du C est rejoué en 32 bits après troncature : uint32(uint64(Load32_le(nonce)) << 32) = 0 (nonce IETF l.416), idem compteur djb (l.409) et carry de Multiply (l.1435, produit 64 bits tronqué avant accumulation)."
	evidence: {
		file_line: "spec/dogfood/testdata/cycles/20260810k_monocypher/sgoiter_out/monocypher_aead_sgoiter.go:409,416,1435"
		kat:       "pass"
		source_doc: "spec/findings/HARVEST_audit_profond_20260811.md#C2"
	}
	lever:  "front"
	action: "Propager le type élargi de l'opérande d'un shift ≥ largeur du type étroit jusqu'au site d'usage ; le même motif est correct dans Poly_blocks — cas de test minimal pour isoler la divergence de typage."
	status: "landed"
	notes:  "Levé 2026-08-11 : élévation OpAdd/OpSub vers uint64 (emit) + TestChacha20IETF_VsC (ctr=0x1000) bit-exact vs C."
}
