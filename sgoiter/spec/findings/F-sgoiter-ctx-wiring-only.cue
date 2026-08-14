package findings

F_sgoiter_ctx_wiring_only: #Finding & {
	id:     "F-sgoiter-ctx-wiring-only"
	kernel: "architecture sgoiter vs API"
	stage:  "doctrine"
	symptom: "Homonymie ctx: usager demande context.Context dans sgoiter; emit utilise ctx=TypeName; C monocypher a *_ctx structs."
	evidence: {
		file_line: "emit arg(v,ctx TypeName); crypto_poly1305_ctx; AGY dd7965 L145-L150 @22:26"
		kat:       "n/a"
		source_doc: "HARVEST_agy_dd7965_2220_2231.md"
	}
	lever:  "front"
	action: "Ne jamais injecter context.Context dans l'emit. Cancel/deadline = wiring. Structs *_ctx C = moisson+emit (ptrMeta/fields) — autre problème."
	status: "codified"
	notes:  "Doctrine 2026-08-10 22:26. Ne pas fusionner avec F-ptrmeta-alias-lost (état C, pas stdlib context)."
}
