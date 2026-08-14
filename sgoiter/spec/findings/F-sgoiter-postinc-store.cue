package findings

F_sgoiter_postinc_store: #Finding & {
	id:     "F-sgoiter-postinc-store"
	kernel: "base64_simd"
	stage:  "emit"
	symptom: "j++ C → snapshot v31:=v5; v5=v5+1; dst[int(v31)]=… (bruit, 4× par triplet base64)."
	evidence: {
		file_line: "emit.go foldPostIncStore ; base64 émis : dst[int(v5)] = … puis v5++ (quatre triplets ramenés à deux lignes chacun, 56 → 47 lignes)"
		kat:       "pass"
		bench_after: "emit/postinc_narrow_test.go TestFoldPostIncStore* ; tribench 11/11 compared, base64 bit-exact"
		source_doc: "spec/findings/HARVEST_dogfood_yeux_post_p0p1_20260811.md#T5"
	}
	lever:  "emit"
	action: "Si snapshot S:=C; C=C+1; store[S] seul use de S → store[C]; C++. Garde : aucun autre use de S. Oracle base64 inchangé."
	status: "landed"
	notes:  "Deux gardes, chacune couverte par un test : le snapshot doit n'avoir qu'un lecteur, et la valeur stockée ne doit pas lire le compteur — sinon avancer l'incrément changerait ce qui est écrit."
}
