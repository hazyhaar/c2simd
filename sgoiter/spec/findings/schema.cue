// Schéma findings sgoiter — thésaurus itératif (subset → IR → règles).
package findings

#Lever:  "c_source" | "ir_rule" | "emit" | "front" | "handwrite"
#Stage:  "front" | "ir" | "emit" | "dogfood" | "doctrine"
#Status: "proposed" | "landed" | "rejected" | "codified"
#Kat:    "pass" | "fail" | "n/a"

#Evidence: {
	file_line?:    string
	bench_before?: string
	bench_after?:  string
	kat:           #Kat
	commit?:       string
	source_doc?:   string
}

#Finding: {
	id:       string & =~"^F-sgoiter-[a-z0-9-]+$"
	kernel:   string
	stage:    #Stage
	symptom:  string
	evidence: #Evidence
	lever:    #Lever
	action:   string
	status:   #Status
	rule_id?: string
	notes?:   string
}
