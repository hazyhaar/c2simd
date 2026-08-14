package findings

// Audit profond 2026-08-11 — C3. & et >> partagent le niveau multiplicatif en Go.
F_sgoiter_precedence_mask_regroup: #Finding & {
	id:     "F-sgoiter-precedence-mask-regroup"
	kernel: "monocypher"
	stage:  "emit"
	symptom: "C : load24 & (0xffffff >> nb_mask) (amalg.c:1441,1451). Émis : Load24_le(s[29:]) & uint32(0xffffff) >> uint8(nb_mask) << 2 — en Go l'associativité gauche du niveau multiplicatif donne ((load24 & 0xffffff) >> nb) << 2. Tout fe_frombytes (nb_mask=1) décode faux."
	evidence: {
		file_line: "spec/dogfood/testdata/cycles/20260810k_monocypher/sgoiter_out/monocypher_aead_sgoiter.go:972"
		kat:       "pass"
		source_doc: "spec/findings/HARVEST_audit_profond_20260811.md#C3"
	}
	lever:  "emit"
	action: "L'émetteur doit parenthéser selon l'arbre IR, jamais selon le texte : conserver les parenthèses structurelles quand la précédence Go diffère de la précédence C (& vs shifts). Test : golden fe_frombytes_mask avec nb_mask=1."
	status: "landed"
	notes:  "Levé 2026-08-11 : needsParen ne traitait plus T(x)>>k comme primaire ; inline garde (load & (mask>>nb))<<2. Yeux fe_frombytes_mask s[29:]."
}
