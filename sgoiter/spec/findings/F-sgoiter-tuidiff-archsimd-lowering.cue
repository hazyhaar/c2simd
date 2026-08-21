package findings

F_sgoiter_tuidiff_archsimd_lowering: #Finding & {
	id:      "F-sgoiter-tuidiff-archsimd-lowering"
	kernel:  "tuidiff"
	stage:   "dogfood"
	symptom: "L'abaissement d'inégalité vectorielle vers archsimd.Equal nécessite une inversion de masque bit-à-bit et une spécialisation arm64 StoreArray."
	evidence: {
		file_line:  "c2simd/pkg/c2tuidiff/diff_simd_amd64.go:79, diff_simd_arm64.go:77"
		kat:        "pass"
		source_doc: "c2simd/sgoiter/TODO_TRIBENCH_FINDINGS.md"
	}
	lever:  "emit"
	action: "Génération de ^vf.Equal(vb).ToBits() sur amd64 et StoreArray + extraction scalaire sur arm64 pour les masques dirty scan."
	status: "landed"
	notes:  "Dogfooding c2tuidiff validé bit-exact sur 8 dimensions et intégré dans pkg/dogfood/tuidiff."
}
