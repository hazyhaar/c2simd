package findings

finding: "F-sgoiter-vte-conformance-vttest": #Finding & {
	id:      "F-sgoiter-vte-conformance-vttest"
	kernel:  "c2vtparser"
	stage:   "dogfood"
	lever:   "handwrite"
	status:  "landed"
	symptom: "Nécessité de garantir la conformité formelle rigoureuse contre les suites de référence vttest et esctest pour l'ensemble des primitives d'émulation terminale VT100/VT500 avec zéro allocation sur les chemins chauds."
	evidence: {
		file_line:  "pkg/c2vtparser/conformance_vttest_test.go:1"
		kat:        "pass"
		source_doc: "pkg/c2vtparser/esctest_test.go"
	}
	action: """
		1. Mouvements et bornes de curseur : support exhaustif de CUU, CUD, CUF, CUB, CNL, CPL, CHA, CUP, HVP, VPR, HPA, VPA, HPR avec clamping strict aux coordonnées 1-based et dimensions de grille.
		2. Édition et effacement : implémentation in-place de ED (0, 1, 2, 3 scrollback), EL (0, 1, 2), ICH (Insert Character), DCH (Delete Character), IL (Insert Line), DL (Delete Line), ECH (Erase Character).
		3. Marges et scrolling DECSTBM : configuration CSI top;bot r, homing automatique en (1, 1), défilement confiné strictement à la région [Top, Bottom], et immunité des lignes extérieures.
		4. Tabulations matérielles : gestion matérielle par masque binaire [4]uint64 (256 colonnes), HT (0x09), HTS (ESC H), TBC 0 (CSI 0g) et TBC 3 (CSI 3g).
		5. Modes DEC privés : support de DECTCEM (?25), DECAWM (?7 auto-wrap avec gestion d'armement/désarmement WrapPending et écrasement en marge droite si inactif), DECOLM (?3 80/132 colonnes), DECSCNM (?5 vidéo inverse).
		6. Jeu de caractères graphiques DEC Special Graphics : prise en charge des désignations G0/G1 (ESC ( 0 / ESC ) 0 / SI / SO) avec table de translation archtime de 32 runes VT100 box drawing.
		"""
	notes: "Validation intégrale sous go test -v -race -run 'TestConformance|TestEscTest' ./... et preuve formelle de zéro allocation (0 allocs/op) sur tous les chemins d'exécution."
}
