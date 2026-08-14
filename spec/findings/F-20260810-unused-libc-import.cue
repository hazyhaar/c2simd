// Dogfood cycle 20260810 : opt siphash/blake2b ne compilaient plus après élision tls T0.
package findings

F_20260810_unused_libc_import: #Finding & {
	id:      "F-20260810-unused-libc-import"
	kernel:  "*"
	stage:   "ast_opt"
	symptom: "Après élision totale de tls, l'import modernc.org/libc reste et casse `go build` (imported and not used)."
	evidence: {
		file_line: "internal/astmatch/astmatch.go dropUnusedImports"
		kat:       "pass"
		bench_before: "build_opt_ok=0 siphash24, blake2b_compress (dogfood 20260810)"
		bench_after:  "build_opt_ok=1 les 5 kernels après dropUnusedImports"
	}
	lever:   "ast_rule"
	action:  "Passe finale dropUnusedImports : Inspect sélecteurs, DeleteImport des chemins dont le base name n'est plus référencé."
	status:  "landed"
	notes:   "Découvert uniquement en dogfood C→ccgo→gen→go build, pas par les tests unitaires rotl seuls."
}
