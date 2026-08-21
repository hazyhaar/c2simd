package findings

finding: "F-sgoiter-ccitt55-huffman-g3g4": #Finding & {
	id:      "F-sgoiter-ccitt55-huffman-g3g4"
	kernel:  "ccitt55"
	stage:   "emit"
	lever:   "emit"
	status:  "landed"
	symptom: "Transpilation de CCITT G3/G4 : blocage sur les tables 2D d'arbres Huffman, déréférencement scalaire erroné sur pointeurs mutés (*p = v) et inversion de l'ordre d'évaluation de la lecture de bits (br_next)."
	evidence: {
		file_line:  "c2simd/sgoiter/emit/emit.go:401,3988,5475"
		kat:        "pass"
		source_doc: "worktrees/pdfast-sgoiter/pkg/ccitt55/sources/ccitt.c"
	}
	action: """
		1. Support dans emit.go de l'émission directe des tables d'arbres 2D [][2]int16 lorsque g.Cols == 2.
		2. Correction de ptrUsedAsScalarOnlyVisited pour distinguer les assignations scalaires (OpStore à 2 arguments) des assignations indexées (3 arguments).
		3. Correction de rhsReassignedBetween / rhsAssignedAfter et foldTempCopies pour vérifier la présence des tokens identifiants dans le LHS (notamment les champs de structure b.Bits) afin d'interdire tout repliement à travers une mutation.
		"""
	notes: "Validation complète sous GOWORK=off CGO_ENABLED=1 GOEXPERIMENT=simd go test -race ./... avec parité bit-exacte contre oracle gcc -O2."
}
