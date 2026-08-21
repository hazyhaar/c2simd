package findings

F_p2go_simd_sum_reduction: #Finding & {
	id:      "F-p2go-simd-sum-reduction"
	kernel:  "for ($i=0; $i<count($a); $i++) { $s += $a[$i]; }"
	stage:   "rules"
	symptom: "La boucle de réduction canonique s'émettait en scalaire pur — la strate simd/archsimd (LE BUT du pôle) n'était pas exercée par p2go."
	evidence: {
		file_line: "rules/simd_sum.go matchSumLoop ; emit/emit.go sumSimdFile"
		fixture:   "testdata/phpt/array_sum.phpt"
		kat:       "pass"
	}
	lever:   "ir_rule"
	action:  "Pattern-matching de la forme canonique → nœud IR SumLoop ; garde de sûreté : compteur lu nulle part ailleurs dans la fonction (sa valeur finale n'est pas matérialisée). Emit dual : helper scalaire (!goexperiment.simd) et Int64x4 ×4 avec garde AVX2 + queue scalaire (goexperiment.simd). Parité stdout vérifiée sous go1.27rc3 + GOEXPERIMENT=simd (simd_parity_test.go)."
	status:  "landed"
	rule_id: "simd_sum"
	notes:   "Parité fonctionnelle prouvée ; le passage runtime effectif par le chemin AVX2 est garanti par la garde du pôle (i9-14900K), pas par un oracle dédié — oracle de débit chiffré = marche v0.4."
}
