package findings

F_sgoiter_ptr_advance_lost: #Finding & {
	id:       "F-sgoiter-ptr-advance-lost"
	kernel:   "monocypher|poly1305|chacha20"
	stage:    "front"
	symptom:  "ptr++ / ptr += N sur tranche []byte dans boucle n'incrémente pas offSlot ni ne ré-indexe la tranche"
	evidence: {
		file_line: "front/front.go:2305; emit/emit.go:1248"
		kat:       "pass"
	}
	lever:    "front"
	action:   "Régénérer offSlot lors des incréments ptr++ et émettre le re-slicing tranche p[int(N):] dans emit"
	status:   "landed"
	notes:    "20260811: *p→Load(base,offSlot); message++ via parseSimpleStmt offSlot (pas Add sur slice). Cause: findOp volait ++ comme + ; stmt ++ integer-only. TestPtrCursor + PolyRemainderThenData + matrix 0..300."
}
