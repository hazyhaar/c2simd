package findings

F_sgoiter_v06_dogfood_11of12: #Finding & {
	id:     "F-sgoiter-v06-dogfood-11of12"
	kernel: "base64|chacha_qr|crc32|fast_xor|fnv|md5|murmur|poly_block5|siphash|tweetnacl|libinjection"
	stage:  "dogfood"
	symptom: "12 kernels c_sources: harvest/build partiel; stubs concentrés tweetnacl (9); blake2b 2D bloqué."
	evidence: {
		file_line: "spec/dogfood/testdata/cycles/20260810j_sgoiter12_fresh/index.json"
		kat:       "pass"
		commit:    "e5c92b8"
	}
	lever:  "front"
	action: "Globals string, j++, char lit, ptr offSlot, cast LE, shl widen byte, murmur KAT gcc."
	status: "landed"
	notes:  "libinjection=1 fn réelle is_mysql_comment zéro stub; pas monocypher AEAD."
}
