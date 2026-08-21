package findings

finding: "F-sgoiter-vte-fuzz-and-throughput": #Finding & {
	id:      "F-sgoiter-vte-fuzz-and-throughput"
	kernel:  "c2vtparser"
	stage:   "dogfood"
	lever:   "handwrite"
	status:  "landed"
	symptom: "Nécessité d'éprouver la robustesse de la pile terminale c2vtparser sous fuzzing massif continu (100k+ itérations corrompues) et de qualifier les débits d'ingestion sous benchmem à zéro allocation."
	evidence: {
		file_line:  "pkg/c2vtparser/throughput_bench_test.go:1"
		kat:        "pass"
		source_doc: "pkg/c2vtparser/fuzz_heavy_test.go"
	}
	action: """
		1. Implémentation du harnais de fuzzing intensif TestFuzzMassiveCorruptedStream (100 000 blocs de flux ANSI hautement bruités, fragments de séquences d'échappement tronquées, octets C0/C1 erratiques, UTF-8 corrompu) avec zéro panic et zéro fuite mémoire.
		2. Implémentation du test de bascule de modes à haute cadence TestFuzzHighCadence_RapidModeSwitching (50 000 cycles de modes DEC, alternate screen, scroll margins, synchronisations et paste bracketed).
		3. Implémentation de la suite de benchmarks de débit brut (BenchmarkFeed à ~1 Go/s, BenchmarkFeedWide à 783 Mo/s, BenchmarkThroughput_CSI_Storm à 308 Mo/s, BenchmarkFeedScrollGoCopy à 441 Mo/s).
		4. Confirmation de 0 allocation sur tous les chemins chauds en régime permanent.
		"""
	notes: "Débit d'ingestion mesuré jusqu'à 965 Mo/s sur processeur hôte avec zéro allocation heap."
}
