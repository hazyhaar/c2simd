// Recherche documentée des pièges ccgo/libc (pkg.go.dev + issues GitLab cznic/ccgo + libc CLAUDE).
// Pas une règle AST — thésaurus opposable pour dogfood et callers.
package findings

F_20260810_ccgo_pitfalls_research: #Finding & {
	id:      "F-20260810-ccgo-pitfalls-research"
	kernel:  "doctrine"
	stage:   "ccgo_raw"
	symptom: "Callers / gen ignorent l'ABI ccgo → UB silencieux, races, bloat, va_list faux."
	evidence: {
		source_doc: "https://pkg.go.dev/modernc.org/ccgo/v4 README Memory ABI ; gitlab.com/cznic/ccgo issues #43 #45 #46 #11 ; modernc.org/libc CLAUDE.md TLS"
		kat:        "n/a"
		commit:     "research-20260810"
	}
	lever: "c_source" // aussi discipline callers Go
	action: """
		Pièges opposables (checklist dogfood + KAT callers) :
		1. JAMAIS passer un pointeur Go (slice/string/struct) en uintptr à du code ccgo —
		   GC peut déplacer ; -race TSan panique. Allouer via libc.Xmalloc/tls.Alloc, copy in/out, Xfree.
		2. TLS = thread C : un *libc.TLS par goroutine, pas safe concurrent ; NewTLS+Close par goroutine.
		3. go vet « possible misuse of unsafe.Pointer » sur code généré = bruit connu (#45), oracle gelé OK.
		4. Unions / grosses tables : bloat 15× initializers (#46) — éviter C data-heavy sans compact.
		5. va_list ABI bug (#43 ouvert) : passer va_list à une sous-fonction rejoue les mêmes args —
		   ne pas transpiler de C varargs imbriqués sans oracle natif.
		6. volatile mal géré (#11) — pas de sync C volatile fiable après transpile.
		7. bool/typedef/bitfield : historique de codegen bugs (fermés mais zone fragile).
		8. Symboles : ccgo v4 émet souvent minuscules (non exportés Go) ; v3/historique capitalise.
		9. Premier param tls injecté partout — T0 c2simd-gen (unexported only) ; exportés gardent ABI.
		10. __ccgo_up / libc wrappers restent des goulots (blake2b) — levier A C ou T1 tls, pas simd générique.
		"""
	status: "codified"
	notes:  "Recherche faite 2026-08-10 en session dogfood ; avant cela la boucle ne thésaurisait pas ces risques. À citer dans chaque REVIEW.md."
}
