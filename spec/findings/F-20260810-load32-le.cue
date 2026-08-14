// Finding rétro : corps load32_le / store32_le → charge/stockage *uint32 via unsafe.Pointer.
package findings

F_20260810_load32_le: #Finding & {
	id:      "F-20260810-load32-le"
	kernel:  "*"
	stage:   "ast_opt"
	symptom: "load32_le/store32_le ccgo : boucle octet-par-octet ou helpers libc, coûteux sur hot path crypto."
	evidence: {
		file_line: "internal/astmatch/astmatch.go load32_le/store32_le body rewrite"
		kat:       "pass"
		bench_before: "corps ccgo multi-octets / libc"
		bench_after:  "return *(*uint32)(unsafe.Pointer(s)) / store symétrique"
	}
	lever:   "ast_rule"
	action:  "RuleDef load32_le + store32_le Kind=rewrite : remplacement du corps par cast unsafe aligné LE (x86). Replacement table documente unsafe, pas encoding/binary (historique)."
	status:  "landed"
	rule_id: "load32_le"
	notes:   "Paramètres de corps attendus : load32_le(..., s) et store32_le(..., out, in) — noms figés dans la réécriture AST."
}
