// Schéma findings p2go — thésaurus itératif du dogfooding (subset → IR → règles),
// repris du modèle sgoiter avec préfixe F-p2go-.
package findings

#Lever:  "php_source" | "ir_rule" | "emit" | "front" | "types" | "handwrite"
#Stage:  "front" | "types" | "ir" | "rules" | "emit" | "dogfood" | "doctrine"
#Status: "proposed" | "landed" | "rejected" | "codified"
#Kat:    "pass" | "fail" | "n/a"

#Evidence: {
	file_line?:    string
	fixture?:      string // testdata/phpt/*.phpt concerné
	bench_before?: string
	bench_after?:  string
	kat:           #Kat
	commit?:       string
	source_doc?:   string
}

#Finding: {
	id:       string & =~"^F-p2go-[a-z0-9-]+$"
	kernel:   string // construct PHP ou fixture visée
	stage:    #Stage
	symptom:  string
	evidence: #Evidence
	lever:    #Lever
	action:   string
	status:   #Status
	rule_id?: string
	notes?:   string
}
