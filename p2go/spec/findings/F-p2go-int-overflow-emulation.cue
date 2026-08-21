package findings

F_p2go_int_overflow_emulation: #Finding & {
	id:      "F-p2go-int-overflow-emulation"
	kernel:  "mul64/mul32 en demi-mots (fnv1a_64, murmur3_32)"
	stage:   "doctrine"
	symptom: "PHP promeut tout overflow arithmétique int en FLOAT (y compris les littéraux > PHP_INT_MAX) ; Go wrappe mod 2^64 — un hachage 64-bit naïf diverge structurellement entre l'oracle et l'émis."
	evidence: {
		file_line: "testdata/algorithms/fnv1a_64.php mul64 ; murmur3_32.php mul32"
		kat:       "pass"
	}
	lever:  "php_source"
	action: "Doctrine du corpus : les algorithmes source restent DANS le domaine int64 signé PHP — multiplications décomposées en colonnes 16-bit (produits ≤ 2^32, sommes < 2^62), constantes > PHP_INT_MAX composées en bitwise ((hi << 32) | lo, les shifts PHP ne promeuvent pas), additions 32-bit masquées. Les deux mondes restent alors bit-exacts sans instrumentation du transpileur."
	status: "landed"
	notes:  "L'alternative (détection d'overflow émise côté Go) imiterait le float PHP — refusée, hors subset int strict."
}
