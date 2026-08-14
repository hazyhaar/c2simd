// Schéma des findings c2simd — registre append-only de micro-opts de transpile.
// Un finding = un fait opposable (symptôme + levier + statut), pas une règle runtime.
// Chaîne : finding → levier A|B|C → rule testée | hand-write | rejet.
//
// Usage : chaque F-*.cue est un fichier du même package `findings` qui conforme
// une instance à #Finding (cue vet ./spec/findings/).
package findings

#Lever:  "c_source" | "ast_rule" | "handwrite"
#Stage:  "ccgo_raw" | "ast_opt" | "handwrite" | "doctrine"
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
	id:      string & =~"^F-[0-9]{8}-[a-z0-9-]+$"
	kernel:  string // noyau touché, ou "doctrine" / "*" si transverse
	stage:   #Stage
	symptom: string
	evidence: #Evidence
	lever:   #Lever
	action:  string
	status:  #Status
	rule_id?: string // si landed/codified dans ArchtimeRulesTable
	notes?:   string
}
