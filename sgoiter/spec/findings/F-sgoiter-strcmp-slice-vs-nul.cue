package findings

// Audit profond 2026-08-11 — C14. strcmp émis en comparaison de tranches entières.
F_sgoiter_strcmp_slice_vs_nul: #Finding & {
	id:     "F-sgoiter-strcmp-slice-vs-nul"
	kernel: "libinjection_sqli"
	stage:  "emit"
	symptom: "Streq (out.go:31) compare les tranches complètes là où strcmp C s'arrête au premier NUL ; les appelants amont (empreintes en tampon fixe, src C:2167-2193) sont précisément la classe divergente. Incohérence interne : le builtin strchr gère le NUL embarqué, strcmp non."
	evidence: {
		file_line: "spec/dogfood/testdata/cycles/20260811_audit_fable/libinjection_sqli/out.go:30-32 ; emit/emit.go:3905-3907"
		kat:       "n/a"
		source_doc: "spec/findings/HARVEST_audit_profond_20260811.md#C14"
	}
	lever:  "emit"
	action: "Borner la comparaison au premier NUL de chaque opérande (sémantique C-string) dans le builtin strcmp. Oracle : vecteurs à NUL embarqué via harnais C -Dstatic=."
	status: "proposed"
	notes:  "P2 tant que les appelants amont ne sont pas moissonnés ; devient P1 dès que la moisson s'élargit aux empreintes."
}
