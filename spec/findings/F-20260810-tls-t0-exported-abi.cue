// Dogfood : élision T0 sur symboles exportés casse l'ABI des packages opt/ commités.
package findings

F_20260810_tls_t0_exported_abi: #Finding & {
	id:      "F-20260810-tls-t0-exported-abi"
	kernel:  "*"
	stage:   "ast_opt"
	symptom: "T0 stripait tls sur Md5_transform_block/Siphash24 exportés → drift ABI vs multi_bench/KAT qui passent encore tls en 1er arg."
	evidence: {
		file_line: "internal/astmatch/astmatch.go pureTLSFuncs + ast.IsExported guard"
		kat:       "pass"
		source_doc: "spec/c2simd_transpiler_2026_peer_review.md Q4"
	}
	lever:   "ast_rule"
	action:  "N'élider tls (param + call sites) que si !ast.IsExported(name). Exportés T0 : param mort conservé (ABI stable)."
	status:  "landed"
	rule_id: ""
	notes:   "ccgo v4 émet souvent des symboles minuscules (non exportés) → élision OK en dogfood frais. raw/ historiques capitalisent (Md5_*) → tls conservé."
}
