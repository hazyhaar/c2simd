package findings

F_sgoiter_rodata_array: #Finding & {
	id:     "F-sgoiter-rodata-array"
	kernel: "blake2b_compress|base64_simd|tweetnacl_dogfood"
	stage:  "emit"
	symptom: "var t = []byte{…} / []uint64{…} au package : header slice + backing mutable ; surface pub faible."
	evidence: {
		file_line: "emit.go globals ; base64 : const b64_table = \"ABC…\" ; blake : var blake2b_sigma = [192]byte{…} ; tweetnacl : K [80]uint64, iv [64]byte"
		kat:       "pass"
		bench_after: "emit/identity_cast_test.go TestRodataTablesAreNotSlices ; tribench 11/11 compared"
		source_doc: "spec/findings/HARVEST_dogfood_yeux_post_p0p1_20260811.md#T6"
	}
	lever:  "emit"
	action: "Tables numériques : var t = [N]T{…} ou […]T{…}. Tables ASCII indexables (b64) : const t = \"…\" (Gemini) — index t[i] → byte, zéro header slice. Nom non exporté."
	status: "landed"
	notes:  "Garde ajoutée après régression : une table PASSÉE EN ARGUMENT garde son type slice — une const string ne satisfait pas un paramètre []byte. Le cas est réel (constante chacha20 de monocypher) et n'a PAS été vu par le banc : c'est TestMonoAEAD_MultiBlock_1KB, hors des douze noyaux, qui a échoué à la compilation. globalsPassedToCalls repère ces tables via le Mov \"global:\" puis l'OpCall."
}
