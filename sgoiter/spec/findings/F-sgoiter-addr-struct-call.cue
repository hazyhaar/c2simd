package findings

F_sgoiter_addr_struct_call: #Finding & {
	id:     "F-sgoiter-addr-struct-call"
	kernel: "crypto_poly1305(&ctx)"
	stage:  "emit"
	symptom: "&ctx (struct value) en ptr_alias sans & → call veut *T reçoit T."
	evidence: {
		file_line: "emit OpMov ptr_alias: si src !regPtr alors regName=&v"
		kat:       "n/a"
	}
	lever:  "emit"
	action: "ptr_alias depuis valeur struct → nom '&v4' pour les appels."
	status: "landed"
}
