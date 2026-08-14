package findings

F_sgoiter_memset_memcpy_builtin: #Finding & {
	id:     "F-sgoiter-memset-memcpy-builtin"
	kernel: "libinjection_sqli|*"
	stage:  "emit"
	symptom: "memset/memcpy/strcmp → stubs args...any (sémantique nulle)."
	evidence: {
		file_line: "emitBuiltinCall memset/memcpy/memmove/strcmp; FillStubs skip"
		kat:       "pass"
	}
	lever:  "emit"
	action: "memset 0→p[i]=0; memcpy→copy élément; void call sans _=."
	status: "landed"
	notes:  "n plafonné par len(slice). Dogfood libinjection St_clear/St_copy."
}
