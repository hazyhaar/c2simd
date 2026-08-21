package findings

finding: "F-sgoiter-c2display-pure-go": #Finding & {
	id:      "F-sgoiter-c2display-pure-go"
	kernel:  "c2display"
	stage:   "dogfood"
	lever:   "front"
	status:  "landed"
	symptom: "Remplacement de libglfw3, libX11 et libwayland par un client de protocoles d'affichage X11, Wayland et Headless natif en Go pur (CGO_ENABLED=0)."
	evidence: {
		file_line: "pkg/c2display/display.go:1"
		kat:       "pass"
		source_doc: "pkg/c2display/display_test.go"
	}
	action: """
		1. Implémentation du protocole binaire X11 (x11.go) via socket UNIX (/tmp/.X11-unix/X0), gestion des formats de pixel ZPixmap 32-bit, découpage en tranches multi-scanlines et dispatch d'événements asynchrones.
		2. Implémentation du protocole Wayland (wayland.go) avec mémoire partagée wl_shm par descripteurs anonymes memfd, projection mmap et surfaces xdg_toplevel.
		3. Pilote Headless virtuel (headless.go) pour exécution et tests automatisés sans serveur graphique.
		4. Validation complète sous CGO_ENABLED=0 avec 100% de tests verts (dont connexion réelle à l'écran local DISPLAY=:0).
		"""
	notes: "Permet la génération d'exécutables graphiques statiques sans aucune dépendance dynamique libX11 ou libGL."
}
