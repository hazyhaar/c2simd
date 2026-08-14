package findings

// Audit profond 2026-08-11 — C18. Casts UB d'endianness des fixtures figés little-endian par l'émis.
F_sgoiter_endian_fixed_le: #Finding & {
	id:     "F-sgoiter-endian-fixed-le"
	kernel: "blake2b_compress|md5_transform"
	stage:  "doctrine"
	symptom: "((uint64_t*)block)[i] (blake2b src.c:39) et ((uint32_t*)block)[i] (md5 src.c:27) sont UB strict-aliasing et dépendants de l'hôte ; l'émis fige binary.LittleEndian (emit.go:1349-1361). Correct pour ces algorithmes et plus défini que le C, mais divergence de classe vs le C sur hôte big-endian, que l'oracle ne peut pas voir."
	evidence: {
		file_line: "emit/emit.go:1349-1361 ; blake2b out.go:20 ; md5 out.go:29"
		kat:       "pass"
		source_doc: "spec/findings/HARVEST_audit_profond_20260811.md#C18"
	}
	lever:  "c_source"
	action: "Graver la décision en doctrine (l'émis choisit la sémantique LE spécifiée par l'algorithme, pas le comportement hôte du C) et l'annoncer dans l'en-tête de l'émis quand elle s'applique. Aucune correction de code."
	status: "proposed"
	notes:  "Décision à documenter, pas un défaut à corriger. fast_xor porte le même motif (émis plus défini que sa source)."
}
