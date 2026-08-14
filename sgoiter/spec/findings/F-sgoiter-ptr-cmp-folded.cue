package findings

// Audit profond 2026-08-11 — C8. (cur+1) < end plié en 1, *(cur+1) devenu cur[0] : tautologie.
F_sgoiter_ptr_cmp_folded: #Finding & {
	id:     "F-sgoiter-ptr-cmp-folded"
	kernel: "libinjection_sqli"
	stage:  "front"
	symptom: "Is_double_delim_escaped retourne toujours 1 : la comparaison de pointeurs (cur+1) < end (src C:622) est pliée en constante 1 (paramètre end mort) et *(cur+1) perd son décalage — l'émis compare uint32(cur[0]) == uint32(cur[0])."
	evidence: {
		file_line: "spec/dogfood/testdata/cycles/20260811_audit_fable/libinjection_sqli/out.go:52-54 ; chaîne ptr_alias/__cmp_ front/front.go:2024-2069"
		kat:       "n/a"
		source_doc: "spec/findings/HARVEST_audit_profond_20260811.md#C8"
	}
	lever:  "front"
	action: "Représenter une comparaison de pointeurs dérivés du même objet en comparaison d'offsets (cur_idx+1 < len) au lieu de la plier ; refuser fail-loud si irreprésentable. Oracle : harnais C -Dstatic= sur src.c, vecteurs délimiteurs doublés."
	status: "proposed"
	notes:  "Seul noyau sans oracle C : le défaut vit exactement dans l'angle mort du banc. « Compilé et exécuté, jamais comparé »."
}
