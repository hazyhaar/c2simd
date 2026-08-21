package findings

F_p2go_match: #Finding & {
	id:      "F-p2go-match"
	kernel:  "match ($v) { 1 => a, 2, 3 => b, default => c }"
	stage:   "front"
	symptom: "L'expression match PHP 8 était inconnue du parser (identifiant traité comme appel de fonction)."
	evidence: {
		file_line: "front/front.go parseMatch ; front/desugar.go extract(*Match)"
		fixture:   "testdata/phpt/match.phpt"
		kat:       "pass"
	}
	lever:  "front"
	action: "Parsé en nœud Match puis désucré par le hoisting en Switch sur temporaire — utilisable en toute position d'expression (affectation, argument, echo). default OBLIGATOIRE : UnhandledMatchError n'est pas émulé."
	status: "landed"
	notes:  "match n'a pas de fallthrough : la traduction switch Go est directe."
}
