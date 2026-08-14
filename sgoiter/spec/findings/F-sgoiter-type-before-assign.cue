package findings

F_sgoiter_type_before_assign: #Finding & {
	id:     "F-sgoiter-type-before-assign"
	kernel: "emit invariant"
	stage:  "emit"
	symptom: "writeAssign lit regType trop tôt (hoist ou :=) si regType n'est pas posé avant l'appel."
	evidence: {
		file_line: "emit.go OpLoad/OpField/binop/global/Call; AGY L193-L557 FIXLOG A3-A5,A16,A52"
		kat:       "n/a"
		source_doc: "FIXLOG_agy_dd7965_20260810.md"
	}
	lever:  "emit"
	action: "Invariant dur: e.regType[dst]=… (et regPtr/elemIdx) AVANT tout writeAssign. Appliqué load/field/binop/global/call."
	status: "landed"
	notes:  "landed-wip non commité. Socle de presque tous les fixes types AGY."
}
