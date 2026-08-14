package findings

F_sgoiter_dogfood_lazy_offslot_dowhile: #Finding & {
	id:     "F-sgoiter-dogfood-lazy-offslot-dowhile"
	kernel: "fnv1a|crc32|fast_xor|blake2b|siphash"
	stage:  "dogfood"
	symptom: "offSlot=0 sur tout uint8* ; do/while(0) → for{if!(0!=0)break} ; 2*0 ; for i<8 via prep Const chaque tour."
	evidence: {
		file_line: "bodyBumpsPtr; doWhileOnce; mul*0 constImm; const cond → ForInit; 0+const fold"
		kat:       "pass"
	}
	lever:  "emit"
	action: "Lazy offSlot si name+=/-= ; while(0) bloc droit ; pli *0 et 0+c ; littéraux de cond hors ForCondPrep."
	status: "landed"
	notes:  "Dogfood accéléré 20260811. fnv data[i] propre ; crc for v<8 ; blake G sans boucle fantôme."
}
