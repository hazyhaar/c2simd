package findings

// Audit profond 2026-08-11 — C17. La fixture md5_transform n'est pas un MD5 complet.
F_sgoiter_md5_fixture_truncated: #Finding & {
	id:     "F-sgoiter-md5-fixture-truncated"
	kernel: "md5_transform"
	stage:  "dogfood"
	symptom: "src.c:29-32 ne contient que les quatre premières étapes FF de la ronde F ; les macros G/H/I (src.c:11-13) sont définies mais jamais invoquées, aucune table K ni table de décalages. L'émis est fidèle à cette troncature (4 RotateLeft32 sur 64 attendus) : l'oracle bit-exact valide un fragment, pas un MD5."
	evidence: {
		file_line: "spec/dogfood/testdata/cycles/20260811_audit_fable/md5_transform/src.c:11-13,29-32"
		kat:       "pass"
		source_doc: "spec/findings/HARVEST_audit_profond_20260811.md#C17"
	}
	lever:  "c_source"
	action: "Compléter la fixture en MD5 réel (64 étapes, table K) ou la renommer md5_ff4 pour que le banc ne prête pas au noyau une couverture qu'il n'a pas."
	status: "proposed"
	notes:  "Le défaut vit dans la fixture, pas dans le transpileur."
}
