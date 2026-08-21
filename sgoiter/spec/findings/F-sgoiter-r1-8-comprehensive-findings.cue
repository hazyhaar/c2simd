package findings

// --- ROUND 1 & 2 : TYPAGE, GLOBALES ET ARITHMÉTIQUE DE BASE ---

F_sgoiter_endian_little_exact: #Finding & {
	id:     "F-sgoiter-endian-little-exact"
	kernel: "Load32_le / Store32_le"
	stage:  "emit"
	symptom: "Incohérence sur le décodage d'entiers multi-octets selon l'architecture hôte."
	evidence: {
		file_line: "sgoiter/emit/emit.go: binary.LittleEndian mapping"
		kat:       "pass"
	}
	lever:  "emit"
	action: "Utilisation systématique des primitives LittleEndian déterministes Go sans accès non aligné brut."
	status: "landed"
}

F_sgoiter_scalar_vs_slice_pointer: #Finding & {
	id:     "F-sgoiter-scalar-vs-slice-pointer"
	kernel: "scalarPtr detection"
	stage:  "emit"
	symptom: "Confusion entre pointeur scalaire *T et tranche []T générant des conversions de type invalides."
	evidence: {
		file_line: "sgoiter/emit/emit.go: ptrUsedAsScalarOnly"
		kat:       "pass"
	}
	lever:  "emit"
	action: "Analyse d'usage des registres pour typer en *T si scalaire pur ou []T si indexation."
	status: "landed"
}

// --- ROUND 3 : LIBINJECTION, SQLI ET HOISTING BCE ---

F_sgoiter_bce_qualified_fields: #Finding & {
	id:     "F-sgoiter-bce-qualified-fields"
	kernel: "libinjection_sqli / ctx.Field"
	stage:  "emit"
	symptom: "Échec de l'élimination des bornes (BCE) sur les accès aux champs de structures qualifiés."
	evidence: {
		file_line: "sgoiter/emit/emit.go: bounds check hoisting"
		kat:       "pass"
	}
	lever:  "emit"
	action: "Neutralisation du hoisting BCE sur les champs qualifiés pour préserver l'accès direct aux structures."
	status: "landed"
}

F_sgoiter_libinjection_corpus_9352: #Finding & {
	id:     "F-sgoiter-libinjection-corpus-9352"
	kernel: "libinjection 9352 SQL inputs"
	stage:  "dogfood"
	symptom: "Cas limites de parsing SQL avec commentaires imbriqués et empreintes lexicales complexes."
	evidence: {
		file_line: "pkg/dogfood/libinjection/dogfood_test.go"
		kat:       "pass"
	}
	lever:  "emit"
	action: "Validation intégrale des 9352 entrées SQL contre l'oracle de référence C."
	status: "codified"
}

// --- ROUND 4 : ARITHMÉTIQUE 128-BIT ET PLIAGE CONSTANT ---

F_sgoiter_arith_128bit_muladd: #Finding & {
	id:     "F-sgoiter-arith-128bit-muladd"
	kernel: "math/bits Mul64 & Add64"
	stage:  "emit"
	symptom: "Multiplications et additions 128 bits (uint128_t) impossibles en Go natif standard."
	evidence: {
		file_line: "sgoiter/emit/emit.go: 128-bit lowered to math/bits"
		kat:       "pass"
	}
	lever:  "emit"
	action: "Abaissement systématique de l'arithmétique large 128 bits vers math/bits.Mul64 et math/bits.Add64."
	status: "landed"
}

F_sgoiter_multiline_field_decls: #Finding & {
	id:     "F-sgoiter-multiline-field-decls"
	kernel: "front struct parsing"
	stage:  "front"
	symptom: "Déclarations multi-champs sur une ligne (int a, b, c;) mal parsées dans les structures C."
	evidence: {
		file_line: "sgoiter/front/struct.go"
		kat:       "pass"
	}
	lever:  "front"
	action: "Dépliage syntaxique automatique des déclarations de variables et champs multiples."
	status: "landed"
}

// --- ROUND 5 : UNIONS, SIZEOF ET NORMALISATION TYPES ---

F_sgoiter_union_and_nested_sizeof: #Finding & {
	id:     "F-sgoiter-union-and-nested-sizeof"
	kernel: "sizeof(T) / union layout"
	stage:  "front"
	symptom: "Évaluation dynamique de sizeof imbriqués et alignement des unions C."
	evidence: {
		file_line: "sgoiter/front/preprocess.go: sizeof folding"
		kat:       "pass"
	}
	lever:  "front"
	action: "Pliage statique à la compilation (ARCHTIME) des sizeof et normalisation des alias __int64."
	status: "landed"
}

// --- ROUND 6 : UTF8PROC, RECURSION, 2D ARRAYS ET CSE ---

F_sgoiter_recursive_loop_visited: #Finding & {
	id:     "F-sgoiter-recursive-loop-visited"
	kernel: "utf8proc visited set"
	stage:  "front"
	symptom: "Boucle de récursion infinie lors de la résolution des types et macros imbriquées."
	evidence: {
		file_line: "sgoiter/front/front.go: visited map"
		kat:       "pass"
	}
	lever:  "front"
	action: "Enregistrement d'un ensemble de nœuds visités pour couper les cycles de résolution récursive."
	status: "landed"
}

F_sgoiter_fn_return_field_access: #Finding & {
	id:     "F-sgoiter-fn-return-field-access"
	kernel: "fn()->field parsing"
	stage:  "front"
	symptom: "Accès direct à un champ sur le résultat d'un appel de fonction non géré syntaxiquement."
	evidence: {
		file_line: "sgoiter/front/front.go: arrow operator on call"
		kat:       "pass"
	}
	lever:  "front"
	action: "Matérialisation d'un registre temporaire intermédiaire pour capturer le retour et accéder au champ."
	status: "landed"
}

F_sgoiter_implicit_2d_array_dims: #Finding & {
	id:     "F-sgoiter-implicit-2d-array-dims"
	kernel: "s[][3] 2D array parsing"
	stage:  "front"
	symptom: "Tableaux 2D avec première dimension implicite mal dimensionnés."
	evidence: {
		file_line: "sgoiter/front/front.go: 2D dimension inference"
		kat:       "pass"
	}
	lever:  "front"
	action: "Inférence automatique de la taille globale à partir du nombre d'éléments de l'initialiseur statique."
	status: "landed"
}

F_sgoiter_cse_literal_exclusion: #Finding & {
	id:     "F-sgoiter-cse-literal-exclusion"
	kernel: "CSE pass / literal folding"
	stage:  "emit"
	symptom: "L'élimination des sous-expressions communes (CSE) créait des alias inutiles sur de simples littéraux entiers."
	evidence: {
		file_line: "sgoiter/emit/emit.go: CSE exclusion"
		kat:       "pass"
	}
	lever:  "emit"
	action: "Exclusion explicite des constantes et littéraux immédiats du dictionnaire de CSE."
	status: "landed"
}

// --- ROUND 7 : CJSON, DOUBLE PTR, ROTATIONS ET CONSTANTS UINT64 ---

F_sgoiter_double_ptr_advance: #Finding & {
	id:     "F-sgoiter-double-ptr-advance"
	kernel: "double_ptr_adv1 / cJSON"
	stage:  "emit"
	symptom: "Avancement de double pointeur C (**p)++ émettant une syntaxe Go invalide."
	evidence: {
		file_line: "sgoiter/emit/emit.go: double_ptr_adv1"
		kat:       "pass"
	}
	lever:  "emit"
	action: "Émission idiomatique *input = (*input)[1:] pour les doubles pointeurs de flux de parsing."
	status: "landed"
}

F_sgoiter_uint64_const_broadening: #Finding & {
	id:     "F-sgoiter-uint64-const-broadening"
	kernel: "emitIns constImm widening"
	stage:  "emit"
	symptom: "Élargissement parasite à uint64 sur les opérandes constants dans les expressions binaires."
	evidence: {
		file_line: "sgoiter/emit/emit.go:2324"
		kat:       "pass"
	}
	lever:  "emit"
	action: "Suppression du forçage uint64 sur les constantes immédiates pour respecter le type dominant."
	status: "landed"
}

F_sgoiter_wipe_scalar_exemption: #Finding & {
	id:     "F-sgoiter-wipe-scalar-exemption"
	kernel: "crypto_wipe / memset / bzero"
	stage:  "emit"
	symptom: "crypto_wipe(ctx, sizeof(*ctx)) forçait ctx en tranche de structures au lieu de pointeur scalaire."
	evidence: {
		file_line: "sgoiter/emit/emit.go:3920"
		kat:       "pass"
	}
	lever:  "emit"
	action: "Exemption des fonctions de wipe mémoire de l'analyse d'indexation ptrUsedAsScalarOnly."
	status: "landed"
}

// --- ROUND 8 : SIMSIMD, HADAMARD, TUIDIFF ET VTPARSER ---

F_sgoiter_simsimd_dot_l2sq_f32: #Finding & {
	id:     "F-sgoiter-simsimd-dot-l2sq-f32"
	kernel: "SimSIMD dot & L2sq f32"
	stage:  "dogfood"
	symptom: "Parité bit-exacte des produits scalaires et distances L2 float32 contre GCC -O2."
	evidence: {
		file_line: "sgoiter/round8_kat_oracle_test.go"
		kat:       "pass"
	}
	lever:  "emit"
	action: "Transpilation mécanique et validation 100% conforme sur vecteurs f32."
	status: "codified"
}

F_sgoiter_fast_hadamard_fht_f32: #Finding & {
	id:     "F-sgoiter-fast-hadamard-fht-f32"
	kernel: "Fast Walsh-Hadamard Transform"
	stage:  "dogfood"
	symptom: "Calculs vectoriels en place de papillons Hadamard 2^N."
	evidence: {
		file_line: "sgoiter/round8_kat_oracle_test.go"
		kat:       "pass"
	}
	lever:  "emit"
	action: "Validation de la transformée rapide in-place sur tranches float32 avec parité GCC -O2."
	status: "codified"
}

F_sgoiter_tuidiff_grid_stride_simd: #Finding & {
	id:     "F-sgoiter-tuidiff-grid-stride-simd"
	kernel: "c2tuidiff / grid diff"
	stage:  "dogfood"
	symptom: "Comparaison vectorielle de lignes de cellules avec stride mémoire."
	evidence: {
		file_line: "pkg/c2tuidiff/oracle_c_test.go"
		kat:       "pass"
	}
	lever:  "emit"
	action: "Génération stricte de diff_gen.go et validation d'oracle C sur grilles de 12 000 cellules."
	status: "codified"
}

F_sgoiter_vtparser_stream_grid_scroll: #Finding & {
	id:     "F-sgoiter-vtparser-stream-grid-scroll"
	kernel: "c2vtparser / ANSI FSM"
	stage:  "dogfood"
	symptom: "Défilement de grilles et mutations de styles SGR en streaming."
	evidence: {
		file_line: "pkg/c2vtparser/vt_test.go"
		kat:       "pass"
	}
	lever:  "emit"
	action: "Génération stricte de parser_gen.go et validation d'équivalence de défilement contre GCC."
	status: "codified"
}
