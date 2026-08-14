package findings

F_sgoiter_v05_array_dowhile_stubs: #Finding & {
	id:     "F-sgoiter-v05-array-dowhile-stubs"
	kernel: "array_lab|memmove_lab|fastlz|md5"
	stage:  "front"
	symptom: "Tableaux T a[N], do-while *p++, callees manquants bloquent build après harvest partiel."
	evidence: {
		file_line: "OpAlloca SKDoWhile ptr_adv1 FillStubs ; testdata/c/{array,memmove}_lab.c"
		kat:       "pass"
	}
	lever:   "front"
	action:  "Arrays params/locals, do-while, *p++= *q++, bare calls, FillStubs for missing callees."
	status:  "landed"
}
