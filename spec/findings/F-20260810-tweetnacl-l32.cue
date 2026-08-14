// Dogfood tweetnacl DL : helper L32 non matché par motif binaire (distance variable + mask).
package findings

F_20260810_tweetnacl_l32: #Finding & {
	id:      "F-20260810-tweetnacl-l32"
	kernel:  "tweetnacl_dogfood"
	stage:   "ast_opt"
	symptom: "L32(tls,x,c) tweetnacl : 0 RotateLeft avant règle ; typedef u32=unsigned long (uint64 LP64) casse RotateLeft32 sans cast."
	evidence: {
		file_line: "rules.go Symbol=L32 ; astmatch L32→u32(bits.RotateLeft32(uint32(x),int(c)))"
		kat:       "pass"
		bench_before: "opt_bits_rotate=0, build fail type u32 vs uint32"
		bench_after:  "4× RotateLeft32 dans core salsa ; go build opt OK"
	}
	lever:   "ast_rule"
	action:  "RuleDef L32 DeadCode + rewrite appels avec cast uint32/u32. Test tweetnacl_L32_call."
	status:  "landed"
	rule_id: "L32"
	notes:   "Source : tweetnacl.cr.yp.to public domain + stub randombytes. Pas de code offensif."
}
