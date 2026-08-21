package findings

F_sgoiter_store_endian_putuint: #Finding & {
	id:      "F-sgoiter-store-endian-putuint"
	kernel:  "monocypher_utils_store"
	stage:   "emit"
	symptom: "Store32_le et Store64_le émis en décalages d'octets manuels au lieu d'appels directs à binary.LittleEndian.PutUint32/PutUint64."
	evidence: {
		file_line: "monocypher_utils.go:26-40"
		kat:       "pass"
	}
	lever:  "emit"
	action: "Émettre directement binary.LittleEndian.PutUint32 et PutUint64 lors de la détection du motif d'écriture little-endian."
	status: "proposed"
}
