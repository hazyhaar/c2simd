package findings

F_sgoiter_local_array_stack_nil_slice: #Finding & {
	id:      "F-sgoiter-local-array-stack-nil-slice"
	kernel:  "monocypher_blake2b_chacha"
	stage:   "emit"
	symptom: "Tableau local C de taille fixe (uint8_t buf[N]) émis en tranche nulle non allouée var v []byte provoquant une panique out-of-bounds à l'accès."
	evidence: {
		file_line: "blake2b.go:134, chacha20.go:150"
		kat:       "fail"
	}
	lever:  "emit"
	action: "Mapper tout tableau C local de taille fixe vers un tableau Go var v [N]byte alloué sur la pile et non vers une tranche slice nil."
	status: "proposed"
}
