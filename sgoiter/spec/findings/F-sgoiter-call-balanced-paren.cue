package findings

F_sgoiter_call_balanced_paren: #Finding & {
	id:     "F-sgoiter-call-balanced-paren"
	kernel: "load64_le = load32_le(s) | ((u64)load32_le(s+4)<<32)"
	stage:  "front"
	symptom: "Call matcher prenait f(x) | g(y) comme un seul call f → empty/parse cassé."
	evidence: {
		file_line: "front/front.go call: matchParen close == len-1"
		kat:       "n/a"
	}
	lever:  "front"
	action: "Appel pur seulement si la ')' appariée à la première '(' est le dernier caractère."
	status: "landed"
}
