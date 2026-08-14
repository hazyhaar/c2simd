package findings

// Dogfood yeux post-e5bab32 — fnv/murmur/crc return T(v) et littéraux sur-typés.
F_sgoiter_identity_cast: #Finding & {
	id:     "F-sgoiter-identity-cast"
	kernel: "fnv1a_64|murmur3_x86_32|crc32_ieee|*"
	stage:  "emit"
	symptom: "return uint64(v2) / uint32(h) alors que regType est déjà T ; uint64(1) sur slot uint64 ; uint64(0xcbf…) redondant."
	evidence: {
		file_line: "emit.go dropIdentityCasts ; fnv émis : return v2 / v4 = v4 + 1 / v2 = 0xcbf29ce484222325"
		kat:       "pass"
		bench_after: "emit/identity_cast_test.go TestDropIdentityCasts ; tribench 11/11 compared"
		source_doc: "spec/findings/HARVEST_dogfood_yeux_post_p0p1_20260811.md#T1"
	}
	lever:  "emit"
	action: "Si expr typée T et cast T(expr) → expr. Littéral sans wrapper si type slot connu. Tests emit golden substrings fnv."
	status: "landed"
	notes:  "Types lus dans le bloc émis (var vN T, vN := T(…), paramètres). Une déclaration courte garde sa conversion : x := 1 serait un int. Étendu aux comparaisons et opérateurs binaires, et au return sur ^v / -v. crc32 : 3 conversions → 1, fnv : 3 → 1."
}
