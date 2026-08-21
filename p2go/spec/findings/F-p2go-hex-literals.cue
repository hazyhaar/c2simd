package findings

F_p2go_hex_literals: #Finding & {
	id:      "F-p2go-hex-literals"
	kernel:  "0xEDB88320, 0xFFFFFFFF"
	stage:   "front"
	symptom: "Littéraux hexadécimaux refusés err_parse — les constantes de CRC32, Murmur3 et FNV s'écrivent en hexa."
	evidence: {
		file_line: "front/front.go lexNumber (branche 0x)"
		fixture:   "testdata/phpt/bitwise.phpt"
		kat:       "pass"
	}
	lever:  "front"
	action: "0x… parsé en base 16 non-signée puis réinterprété int64 (0xFFFFFFFFFFFFFFFF = -1, aligné sur PHP 64-bit qui garde les hexa en int)."
	status: "landed"
	notes:  "0b… et séparateurs _ : v0.5 si le corpus les exige."
}
