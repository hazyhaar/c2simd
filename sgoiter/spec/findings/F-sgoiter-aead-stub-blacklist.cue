package findings

F_sgoiter_aead_stub_blacklist: #Finding & {
	id:     "F-sgoiter-aead-stub-blacklist"
	kernel: "crypto_aead_*"
	stage:  "emit"
	symptom: "FillStubs ...any sur crypto_aead_* donne build OK trompeur (AEAD mort)."
	evidence: {
		file_line: "emit/stubs.go FillStubs skip crypto_aead_*"
		kat:       "n/a"
	}
	lever:  "emit"
	action: "Blacklist préfixe crypto_aead_ dans FillStubs; fail-loud si symbole AEAD manquant au KAT."
	status: "landed"
	notes:  "poly1305_update encore stub possible — KAT AEAD doit exiger 0 stub sur chaîne lock."
}
