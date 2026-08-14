// Fixette dogfood extrême (regex/lz4) : deux bugs ccgo→Go build.
package findings

F_20260810_uintptr_neg_iqlibc: #Finding & {
	id:      "F-20260810-uintptr-neg-iqlibc"
	kernel:  "tiny_regex|lz4"
	stage:   "ast_opt"
	symptom: "1) str+uintptr(-1) → constant -1 overflows uintptr. 2) iqlibc.__builtin_memmove undefined (lz4 LZ4_memmove)."
	evidence: {
		file_line: "astmatch matchUintptrNegConst ; iqlibc→libc.X*"
		kat:       "pass"
		bench_before: "tiny_regex+lz4 BUILD FAIL"
		bench_after:  "BUILD OK après gen v2 fixette"
	}
	lever:   "ast_rule"
	action:  "x+uintptr(-N)→x-uintptr(N) ; iqlibc.__builtin_foo→libc.Xfoo. Tests unitaires dédiés."
	status:  "landed"
}
