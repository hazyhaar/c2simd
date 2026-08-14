package findings

F_sgoiter_ptr_cast_index: #Finding & {
	id:     "F-sgoiter-ptr-cast-index"
	kernel: "siphash24|fast_xor|md5|blake2b"
	stage:  "front"
	symptom: "((uint64_t*)k)[0] refusé: isSimpleBase false sur cast."
	evidence: {
		file_line: "front isPtrCastBase; index path"
		kat:       "pass"
	}
	lever:  "front"
	action: "isPtrCastBase (TYPE*)ident ; isSimpleBase true."
	status: "landed"
}
