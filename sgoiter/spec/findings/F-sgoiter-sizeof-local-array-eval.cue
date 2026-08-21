package findings

F_sgoiter_sizeof_local_array_eval: #Finding & {
	id:      "F-sgoiter-sizeof-local-array-eval"
	kernel:  "sgoiter/front sizeof local arrays & uint16 type mapping"
	stage:   "front"
	symptom: "foldSizeofTypes repliait sizeof(stack) en constante 16 au lieu de 8192 octets pour un tableau uint8_t stack[8192], provoquant des faux codes d'erreur -2 sur toute chaîne LZW > 16 octets. De plus, uint16_t était faussement mappé en TypUint32."
	evidence: {
		file_line: "sgoiter/front/preprocess.go: foldSizeofTypes, sgoiter/front/front.go: mapType"
		kat:       "pass"
	}
	lever:  "front"
	action: "Empêcher le repliement prématuré de sizeof sur des identifiants de variables dans preprocess.go ; évaluer sizeof(var) dans front.go via ri.localArr * elemSize ; mapper uint16_t vers ir.TypUint16 et uint16 en Go."
	status: "landed"
}
