package findings

// Finding F-20260820-stb-tablefree-crc32-kernel
// Transpilation et vérification bit-exacte de l'algorithme CRC32 sans table mémoire
// extrait de stb_image face à gcc -O2.

"F-20260820-stb-tablefree-crc32-kernel": #Finding & {
	id:       "F-20260820-stb-tablefree-crc32-kernel"
	kernel:   "stb"
	stage:    "ast_opt"
	symptom:  "Vérification différentielle du polynôme générateur 0xEDB88320 sans table lookup"
	evidence: {
		file_line: "tribench/catalog.go:84"
		kat:       "pass"
	}
	lever:   "ast_rule"
	action:  "Émission de la boucle de décalage et masque de bits CRC32 Go 1.27 sans dépendance de tableau"
	status:  "landed"
	notes:   "Valide le score bit-exact 30/30 face à gcc -O2"
}
