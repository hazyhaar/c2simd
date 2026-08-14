package findings

F_sgoiter_go_block_scope: #Finding & {
	id:     "F-sgoiter-go-block-scope"
	kernel: "chacha20_djb cipher_text+=4 in if/else"
	stage:  "emit"
	symptom: "v := 0 dans if { for } ; else utilise v → undefined (scope Go)."
	evidence: {
		file_line: "emit writeAssign hoist two-pass; var at func top"
		kat:       "n/a"
	}
	lever:  "emit"
	action: "Pass1 collect dst types; déclarer var au début de func; pass2 assign =."
	status: "landed"
}
