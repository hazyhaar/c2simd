package findings

F_sgoiter_ptr_param_fstore_slice: #Finding & {
	id:     "F-sgoiter-ptr-param-fstore-slice"
	kernel: "c2_painter_init(p, pixels, ...)"
	stage:  "emit"
	symptom: "Un paramètre pointeur uint32_t* affecté à un champ struct slice (p->pixels = pixels) était émis comme scalaire *uint32 au lieu de []uint32 car ptrUsedAsScalarOnly ne vérifiait pas OpFStore."
	evidence: {
		file_line: "sgoiter/emit/emit.go: ptrUsedAsScalarOnly / OpFStore"
		kat:       "pass"
	}
	lever:  "emit"
	action: "Ajout de la détection de OpFStore dans ptrUsedAsScalarOnlyVisited pour inférer la signature slice []T lorsque la valeur est affectée dans un champ struct."
	status: "landed"
	notes:  "Permet l'initialisation idiomatique des contextes de dessin et buffers par passage de slice."
}
