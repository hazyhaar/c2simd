package findings

finding: "F-sgoiter-vte-advanced-protocols": #Finding & {
	id:      "F-sgoiter-vte-advanced-protocols"
	kernel:  "c2vtparser"
	stage:   "dogfood"
	lever:   "handwrite"
	status:  "landed"
	symptom: "Nécessité de supporter les protocoles terminaux avancés (OSC 52 presse-papier, OSC 8 hyperliens, DEC 2026 synchronisation d'affichage anti-tearing, SGR-1006 souris étendue) avec 0 allocation sur le chemin chaud et sas de sécurité anti-réflexion."
	evidence: {
		file_line:  "pkg/c2vtparser/extensions.go:1"
		kat:        "pass"
		source_doc: "pkg/c2vtparser/extensions_test.go"
	}
	action: """
		1. Implémentation du buffer statique d'accumulation OSC (4096 octets de base, cap 1 Mo max).
		2. Traitement sécurisé d'OSC 52 (décodage base64 direct, callbacks OnClipboardWrite/OnClipboardQuery isolés sans réflexion PTY).
		3. Support d'OSC 8 (gestion des hyperliens URI avec identifiant optionnel et dissociation atomique).
		4. Support de DEC 2026 (CSI ? 2026 h/l) pour la synchronisation de trame et le rendu anti-tearing.
		5. Support de DECSET/DECRST 1000/1002/1003/1006/1016/2004 pour la souris et le bracketed paste avec helpers d'encodage.
		"""
	notes: "Validation complète sous go test -race -benchmem avec 0 allocation sur le chemin d'émulation terminale."
}
