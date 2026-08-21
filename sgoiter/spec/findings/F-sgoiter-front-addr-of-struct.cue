package findings

F_sgoiter_front_addr_of_struct: #Finding & {
	id:     "F-sgoiter-front-addr-of-struct"
	kernel: "parseExpr &ctx"
	stage:  "front"
	symptom: "Prise d'adresse locale &ctx sans indication du type d'élément struct."
	evidence: {
		file_line: "sgoiter/front/front.go:2837"
		kat:       "pass"
	}
	lever:  "front"
	action: "Émission systématique de sym='addr_of' avec Elem=ri.structName pour &ctx."
	status: "landed"
}
