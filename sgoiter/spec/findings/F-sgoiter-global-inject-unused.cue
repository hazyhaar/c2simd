package findings

F_sgoiter_global_inject_unused: #Finding & {
	id:     "F-sgoiter-global-inject-unused"
	kernel: "monocypher emit"
	stage:  "front"
	symptom: "Injecter tous les globals dans chaque func → declared and not used (v0..v9)."
	evidence: {
		file_line: "front/front.go parseFunc globals filter by body ref"
		kat:       "n/a"
	}
	lever:  "front"
	action: "N'injecter que les globals dont le nom apparaît (word-boundary) dans body/params."
	status: "landed"
}
