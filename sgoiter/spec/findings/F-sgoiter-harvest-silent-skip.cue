package findings

// Audit profond 2026-08-11 — C12. Fonction non moissonnée : aucune trace dans l'émis.
F_sgoiter_harvest_silent_skip: #Finding & {
	id:     "F-sgoiter-harvest-silent-skip"
	kernel: "chacha20_qr|*"
	stage:  "front"
	symptom: "chacha20_double_round (src.c:14-23) absente de l'émis sans stub ni commentaire (grep double_round out.go → 0). La moisson range les rejets dans Skipped sans les matérialiser — un symbole APPELÉ non harvesté est stubbé bruyant, un symbole non appelé disparaît en silence."
	evidence: {
		file_line: "spec/dogfood/testdata/cycles/20260811_audit_fable/chacha20_qr/out.go (une seule fonction) ; front/front.go:114-136"
		kat:       "pass"
		source_doc: "spec/findings/HARVEST_audit_profond_20260811.md#C12"
	}
	lever:  "front"
	action: "Matérialiser Skipped dans l'émis : un commentaire d'en-tête « N fonction(s) non moissonnée(s) : … » au minimum, aligné sur l'annonce stderr des stubs. Test : golden chacha20_qr contient la mention double_round."
	status: "proposed"
	notes:  "Résout aussi le mystère « 1 kB de C → 25 lignes ». Cohérence fail-loud avec les stubs annoncés."
}
