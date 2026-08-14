package findings

F_sgoiter_hoist_unused: #Finding & {
	id:     "F-sgoiter-hoist-unused"
	kernel: "crypto_chacha20_djb"
	stage:  "emit"
	symptom: "declared and not used: v87 — hoist déclare tous les dst pass1 y compris morts / mono-branche."
	evidence: {
		file_line: "emit emitFunc hoist loop; mono_aead.go v87"
		kat:       "n/a"
	}
	lever:  "emit"
	action: "Prune hoistTypes via isRead : les variables non relues ou purement locales sont omises de la section de déclaration hoisted."
	status: "landed"
	notes:  "Résolu 2026-08-10 : validation go build du package AEAD sans variable inutilisée."
}
