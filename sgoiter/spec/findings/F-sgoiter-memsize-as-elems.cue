package findings

// Audit profond 2026-08-11 — C9. sizeof (octets) appliqué comme nombre d'éléments, borne silencieuse.
F_sgoiter_memsize_as_elems: #Finding & {
	id:     "F-sgoiter-memsize-as-elems"
	kernel: "libinjection_sqli"
	stage:  "emit"
	symptom: "memset/memcpy de stoken_t (sizeof = 64 octets) émis en boucles de 64 ÉLÉMENTS sur []int (8 octets chacun), avec garde i<len(p) qui rend toute copie partielle silencieuse — contraire au fail-loud."
	evidence: {
		file_line: "spec/dogfood/testdata/cycles/20260811_audit_fable/libinjection_sqli/out.go:34-40 ; builtins emit/emit.go:3892-3903 (le commentaire admet l'approximation)"
		kat:       "n/a"
		source_doc: "spec/findings/HARVEST_audit_profond_20260811.md#C9"
	}
	lever:  "emit"
	action: "Convertir sizeof en nombre d'éléments via la taille de l'élément du type destinataire ; remplacer la borne silencieuse par un panic si n dépasse len. Test : St_copy sur slices de tailles inégales."
	status: "proposed"
	notes:  "Deux fautes empilées : l'unité (octets vs éléments) et la garde qui masque la seconde."
}
