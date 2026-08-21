package findings

// Finding F-20260820-gcc-bit-builtins-emission
// Les fonctions intrinsèques GCC/Clang de manipulation de bits (__builtin_clz, __builtin_clzll,
// __builtin_ctz, __builtin_ctzll, __builtin_popcount, __builtin_bswap16/32/64)
// étaient émises comme des appels de fonctions inexistantes et stubbées.
// Résolu par l'abaissement direct vers le package standard Go math/bits (LeadingZeros, TrailingZeros, OnesCount, ReverseBytes).

"F-20260820-gcc-bit-builtins-emission": #Finding & {
	id:       "F-20260820-gcc-bit-builtins-emission"
	kernel:   "quickjs"
	stage:    "ast_opt"
	symptom:  "Appels stubbés non résolus pour __builtin_clz, __builtin_ctz, __builtin_popcount"
	evidence: {
		file_line: "emit/emit.go:2003"
		kat:       "pass"
	}
	lever:   "ast_rule"
	action:  "Abaissement direct des intrinsèques GCC de bits vers math/bits dans l'émetteur Go 1.27"
	status:  "landed"
	notes:   "Active l'exploitation matérielle zero-overhead des instructions BSR/BSF/POPCNT/LZCNT pour QuickJS et les bibliothèques C"
}
