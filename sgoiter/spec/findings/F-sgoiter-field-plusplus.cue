package findings

F_sgoiter_field_plusplus: #Finding & {
	id:     "F-sgoiter-field-plusplus"
	kernel: "crypto_poly1305_update ctx->c_idx++"
	stage:  "front"
	symptom: "ctx->c_idx++ / field++ : postfix appelle parseStore sans handler p->field → empty/store lhs."
	evidence: {
		file_line: "front/front.go parseStore arrow; TestFor0"
		kat:       "n/a"
	}
	lever:  "front"
	action:  "parseStore p->field; postfix expr handles ->; pe() restore f.Body (ne pas wipe parent)."
	status:  "landed"
	notes:   "Bug connexe: pe() faisait f.Body=nil et effaçait le corps parent en cours de postfix/store."
}
