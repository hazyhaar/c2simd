package findings

F_p2go_strpos_sentinel: #Finding & {
	id:      "F-p2go-strpos-sentinel"
	kernel:  "strpos($h, $n)"
	stage:   "doctrine"
	symptom: "strpos PHP retourne int OU false — pas d'équivalent dans le subset int strict ; l'idiome === false est intraduisible sans kind union (v0.5 chantier 3, décision tranchée)."
	evidence: {
		file_line: "emit/builtins.go (strings.Index) ; testdata/phpt/strpos_sentinel.phpt"
		kat:       "pass"
	}
	lever:  "emit"
	action: "SENTINELLE -1 (strings.Index Go, nativement -1 si absent) — écart ASSUMÉ et documenté : echo strpos(absent) imprime -1 (PHP : vide), les tests se font par < 0 ou == -1, jamais === false. Deux fixtures : cas trouvés validés par l'oracle php (positions identiques), cas absent à EXPECT épinglé main marqué divergence — l'oracle ne PEUT pas valider une divergence voulue."
	status: "landed"
	notes:  "Position 0 vs false : le piège PHP classique disparaît avec la sentinelle (0 = trouvé en tête, -1 = absent, jamais ambigus)."
}
