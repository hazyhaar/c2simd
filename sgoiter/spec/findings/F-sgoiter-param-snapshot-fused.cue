package findings

// Audit profond 2026-08-11 — C6. T b = param; puis param réaffecté : b fusionné.
F_sgoiter_param_snapshot_fused: #Finding & {
	id:     "F-sgoiter-param-snapshot-fused"
	kernel: "tweetnacl_dogfood"
	stage:  "emit"
	symptom: "Dans Crypto_hash, la sauvegarde C b = n a disparu : le padding SHA-512 encode n << 3 avec n déjà réaffecté (out.go:220-222) au lieu de la longueur réelle. L'IR est correct (deux registres distincts) ; la fusion vit dans la passe d'alias."
	evidence: {
		file_line: "spec/dogfood/testdata/cycles/20260811_audit_fable/tweetnacl_dogfood/out.go:220-222 ; emit/emit.go:1700, canPureAlias emit/emit.go:3365-3368"
		kat:       "pass"
		source_doc: "spec/findings/HARVEST_audit_profond_20260811.md#C6"
	}
	lever:  "emit"
	action: "canPureAlias doit vérifier les réécritures de la source d'un alias aussi quand c'est un nom de paramètre, pas seulement un vreg. Test : golden crypto_hash, le champ de longueur lit la valeur sauvegardée."
	status: "proposed"
	notes:  "Fonction au-dessus de stubs (Crypto_hashblocks), jamais exécutée par le banc — zone aveugle par construction."
}
