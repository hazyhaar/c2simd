// Relecture hot path : __ccgo_up domine le bruit après rotates idiomatiques.
package findings

F_20260810_ccgo_up_goulot: #Finding & {
	id:      "F-20260810-ccgo-up-goulot"
	kernel:  "*"
	stage:   "ast_opt"
	symptom: "Chaque accès *p C devient **(**T)(__ccgo_up(addr)) — indirection + unsafe par load/store ; goulot post-rotate (blake2b, chacha QR, poly1305_block5)."
	evidence: {
		file_line: "internal/astmatch/astmatch.go matchCcgoUpStar ; tests ccgo_up_*"
		kat:       "pass"
	}
	lever:   "ast_rule"
	action:  "*(**T)(__ccgo_up(E)) → (*T)(unsafe.Pointer(E)) ; scalaire **(**T) → *(*T)(unsafe.Pointer(E)) ; drop def __ccgo_up si 0 appel."
	status:  "landed"
	rule_id: "ccgo_up_star"
	notes:   "Sémantique préservée (__ccgo_up(n)=unsafe.Pointer(&n)). Pas d'analyse locale. Interdit de passer Go pointers (F-20260810-ccgo-pitfalls-research §1). Bench go1.27rc1+simd 2026-08-10 post-land (i9-14900K): SipHash +8.5%, BLAKE2b +16.5% (était ~0%), MD5 +13.5%; FastXOR −12% stable (mem-bound, à profiler séparément)."
}
