package findings

F_sgoiter_tribench_bitexact_gaps: #Finding & {
	id:     "F-sgoiter-tribench-bitexact-gaps"
	kernel: "blake2b|chacha_qr|poly1305_block5|base64"
	stage:  "dogfood"
	symptom: "tribench 20260811: sgoiter digests ≠ gcc -O2 oracle on 4/12 kernels; ccgo matches oracle on those same kernels."
	evidence: {
		file_line: "spec/dogfood/testdata/cycles/tribench_20260811/report.json"
		kat:       "fail"
		source_doc: "tribench SUMMARY.md"
	}
	lever:  "emit"
	action: "Adjudiquer ROT/array ptr (chacha a[0]), blake sigma/ROTR, poly carry, base64 table — contre stdout oracle C."
	status: "landed"
	notes:  "20260811 12/12: clearCSE after if (base64 tail); killCSEBase on Store (chacha/blake/poly reload). Cycle tribench_20260811_12of12."
}
