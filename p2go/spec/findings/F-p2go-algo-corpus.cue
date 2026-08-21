package findings

F_p2go_algo_corpus: #Finding & {
	id:      "F-p2go-algo-corpus"
	kernel:  "crc32_ieee, fnv1a_64, murmur3_32, quicksort_i64, base64_core, chacha20_qr"
	stage:   "dogfood"
	symptom: "Aucun programme RÉEL transpilé — le subset n'était exercé que par des cas minimaux synthétiques."
	evidence: {
		file_line: "testdata/algorithms/ (6 fichiers) ; algo_oracle_test.go"
		kat:       "pass"
	}
	lever:  "php_source"
	action: "Corpus de 6 algorithmes en PHP pur, tous bit-exacts vs oracle php 8.3 : CRC32 IEEE sans table (vecteur 0xCBF43926 constaté), FNV-1a 64 (mul64 demi-mots), Murmur3 x86 32 (rotl+fmix), quicksort itératif à pile explicite array_push/array_pop (partition de Lomuto), base64 encode+décode via alphabet (roundtrip vérifié), quarter-round ChaCha20 (vecteurs RFC 7539 §2.1.1 et §2.2.1)."
	status: "landed"
	notes:  "quicksort vit au top-level (params tableau en lecture seule) ; chacha qr copie l'état via array_slice — les gardes de copie PHP ont contraint le style sans bloquer."
}
