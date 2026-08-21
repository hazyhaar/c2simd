package findings

F_sgoiter_utf8_split_chunk_boundary: #Finding & {
	id:     "F-sgoiter-utf8-split-chunk-boundary"
	kernel: "c2vtparser / streaming UTF-8 PTY"
	stage:  "dogfood"
	symptom: "Perte ou corruption de runes multi-octets (notamment 4 octets) fragmentées aux frontières de blocs PTY de 4096 octets."
	evidence: {
		file_line: "pkg/c2vtparser/cve_regression_test.go: TestUTF8_SplitChunkBoundary"
		kat:       "pass"
	}
	lever:  "emit"
	action: "Gestion d'état tamponné pendingUTF8/pendingLen préservant les fragments entre appels consécutifs de Feed."
	status: "codified"
}
