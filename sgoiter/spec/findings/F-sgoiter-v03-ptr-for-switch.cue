package findings

F_sgoiter_v03_ptr_for_switch: #Finding & {
	id:     "F-sgoiter-v03-ptr-for-switch"
	kernel: "murmur3_x86_32"
	stage:  "front"
	symptom: "Helpers rotl/fmix harvestés mais hash complet bloqué sur T*, for, switch, index négatif C."
	evidence: {
		file_line: "sgoiter/front v0.3 ; testdata/c/murmur3_lab.c ; go test KAT no-panic"
		kat:       "pass"
	}
	lever:   "front"
	action:  "Pointeurs scalaires+scale, for/switch structurés, unary minus en int, Load LE via encoding/binary, alias root+baseOff."
	status:  "landed"
}
