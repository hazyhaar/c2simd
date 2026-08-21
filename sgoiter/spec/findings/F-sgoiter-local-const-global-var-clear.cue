package findings

F_sgoiter_local_const_global_var_clear: #Finding & {
	id:      "F-sgoiter-local-const-global-var-clear"
	kernel:  "monocypher_curve25519_inverse"
	stage:   "emit"
	symptom: "Table de constante locale de fonction m_inv promue en variable globale de paquet var M_inv puis remise à zéro par clear(), corrompant les calculs ultérieurs et induisant une course de données."
	evidence: {
		file_line: "curve25519.go:360, monocypher_types.go:127"
		kat:       "fail"
	}
	lever:  "emit"
	action: "Conclure la portée locale des tables constantes à la fonction elle-même sous forme d'initialisation sur la pile."
	status: "proposed"
}
