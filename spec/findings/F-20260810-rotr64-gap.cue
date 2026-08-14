// Dogfood blake2b : 0 bits.RotateLeft gagné — le C utilise ROTR64, pas ROTL.
package findings

F_20260810_rotr64_gap: #Finding & {
	id:      "F-20260810-rotr64-gap"
	kernel:  "blake2b_compress"
	stage:   "ast_opt"
	symptom: "blake2b_compress.c définit ROTR64(x,y)=((x)>>(y))^((x)<<(64-(y))) ; la passe P1.5 ne reconnaît que (x<<N)|(x>>(W-N)) (rotate LEFT via OR)."
	evidence: {
		file_line: "spec/dogfood/testdata/cycles/20260810/blake2b_compress/src.c:4 ROTR64 ; metrics bits_rotate_gained=0"
		kat:       "n/a"
		bench_before: "BLAKE2b raw 789.73 MB/s"
		bench_after:  "BLAKE2b opt 786.97 MB/s (~0 %, bruit)"
	}
	lever:  "ast_rule"
	action: "matchRotateBinary : ROR (x>>N)|^ (x<<(W-N)) → bits.RotateLeft(x, W-N) ; OR et XOR admis. Tests rotr64_* + KAT blake2b PASS. 32× RotateLeft64 dans opt/blake2b."
	status: "landed"
	notes:  "Perf blake2b reste ~0 % (bottleneck tls/__ccgo_up, pas les rotates — SSA matchait déjà). Gain sémantique/idiomatique + porte pour futures passes. Dogfood 20260810b : bits_rotate_gained=32."
}
