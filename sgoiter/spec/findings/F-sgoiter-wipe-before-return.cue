package findings

// Audit profond 2026-08-11 — C1. Fe_isodd retourne toujours 0, Fe_isequal toujours 1.
F_sgoiter_wipe_before_return: #Finding & {
	id:     "F-sgoiter-wipe-before-return"
	kernel: "monocypher"
	stage:  "emit"
	symptom: "Le wipe C de fin de fonction est émis AVANT l'expression de retour qui lit le buffer : Fe_isodd lit v1[0] après remise à zéro (l.1382-1383), Fe_isequal compare deux buffers wipés (l.1393-1395)."
	evidence: {
		file_line: "spec/dogfood/testdata/cycles/20260810k_monocypher/sgoiter_out/monocypher_aead_sgoiter.go:1378-1396 ; C monocypher_amalg.c:1639-1659"
		kat:       "pass"
		source_doc: "spec/findings/HARVEST_audit_profond_20260811.md#C1"
	}
	lever:  "emit"
	action: "Ordonner le lowering du return : évaluer l'expression de retour dans un temporaire AVANT d'émettre les wipes de fin de scope. Test golden : Fe_isodd contient le return avant la boucle de wipe."
	status: "landed"
	notes:  "Levé 2026-08-11 (3be6e74): rhsReassignedBetween barre for _i:=range buf ; snapshot avant wipe. KAT TestFeIsOdd_WipeCheck (Fe_isodd(1)==1)."
}
