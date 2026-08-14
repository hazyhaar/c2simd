package findings

F_sgoiter_v02_partial_harvest: #Finding & {
	id:     "F-sgoiter-v02-partial-harvest"
	kernel: "murmur3|xxhash|fastlz"
	stage:  "front"
	symptom: "Fichiers biscuit réels échouent en bloc sur #include/pointeurs alors que des helpers scalaires (rotl, fmix) sont extractibles."
	evidence: {
		file_line: "sgoiter/front ParsePartial ; testdata/c/murmur3_lab.c"
		kat:       "pass"
	}
	lever:   "front"
	action:  "v0.2 : strip #lines, harvest fonction-par-fonction, types uint*_t, compound assigns, calls. Pointeurs/for restent skip."
	status:  "landed"
	notes:   "Incrément dogfood — le but est d'élargir, pas de rester fail-loud éternel."
}
