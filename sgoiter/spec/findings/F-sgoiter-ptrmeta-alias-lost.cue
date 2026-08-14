package findings

F_sgoiter_ptrmeta_alias_lost: #Finding & {
	id:     "F-sgoiter-ptrmeta-alias-lost"
	kernel: "crypto_poly1305_* ctx"
	stage:  "front"
	symptom: "Param ctx *Struct a ptrMeta; pe() rebind vers scratch sans copier meta → lookup struct name vide → fallback types uint8."
	evidence: {
		file_line: "front/front.go parseFunc pe regs map; AGY transcript L396-L488"
		kat:       "fail"
	}
	lever:  "front"
	action: "Sur alias registre, copier ptrMeta, regType struct, field table et préserver ri.v sur lhs."
	status: "landed"
	notes:  "Résolu 2026-08-10 : attribution de v: r pour les déclarations locales de pointeurs et maintien de ptrMeta."
}
