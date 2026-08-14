package findings

// AGY dd7965 dogfood visuel 2026-08-11 — modifié au sol sans CUE (thésaurisé a posteriori).
F_sgoiter_wrapint_double_cast: #Finding & {
	id:     "F-sgoiter-wrapint-double-cast"
	kernel: "fast_xor|siphash24|*"
	stage:  "emit"
	symptom: "Index émis `int(int(vN))` : e.arg(..., TypInt) renvoie déjà int(...), OpLoad/OpStore re-wrappaient."
	evidence: {
		file_line: "emit.go wrapInt + OpLoad/OpStore idx"
		kat:       "pass"
	}
	lever:  "emit"
	action: "helper wrapInt(s) : si déjà préfixé int(…), ne pas re-caster."
	status: "landed"
	notes:  "Session AGY dd7965 relecture out.go 20260811_sgoiter12_rerun ; ci_check OK après."
}
