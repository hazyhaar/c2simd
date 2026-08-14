package findings

F_sgoiter_global_shadow_param: #Finding & {
	id:     "F-sgoiter-global-shadow-param"
	kernel: "load32_le param s vs global s"
	stage:  "front"
	symptom: "Global monocypher `s` injecté dans load32_le(s[4]) → OpMov global:s + unused + mauvais binding."
	evidence: {
		file_line: "front/front.go parseFunc globals after params; skip shadow + skip len<=1 sans data"
		kat:       "n/a"
	}
	lever:  "front"
	action: "Injecter globals APRÈS params; ignorer si nom déjà param; ignorer id 1 lettre sans Data/ZeroLen."
	status: "landed"
}
