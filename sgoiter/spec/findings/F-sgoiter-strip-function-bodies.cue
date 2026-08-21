package findings

F_sgoiter_strip_function_bodies: #Finding & {
	id:     "F-sgoiter-strip-function-bodies"
	kernel: "harvestGlobals / stripFunctionBodies"
	stage:  "front"
	symptom: "Variables locales initialisées promues à tort en constantes globales de paquet."
	evidence: {
		file_line: "sgoiter/front/front.go: stripFunctionBodies"
		kat:       "pass"
	}
	lever:  "front"
	action: "Neutralisation du corps des fonctions C lors de la récolte des symboles globaux."
	status: "landed"
}
