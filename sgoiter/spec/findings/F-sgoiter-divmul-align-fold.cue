package findings

F_sgoiter_divmul_align_fold: #Finding & {
	id:     "F-sgoiter-divmul-align-fold"
	kernel: "fast_xor|*"
	stage:  "emit"
	symptom: "(i/8)*8 émis en div+mul redondants ; offSlot+=N const-foldé en =N (boucle infinie siphash)."
	evidence: {
		file_line: "emit markDivMulFolds + OpMul fold; noConstArg offSlot; skip x+0 sur offslot"
		kat:       "pass"
	}
	lever:  "emit"
	action: "(x/k)*k k=2^n → x&^(k-1) ; suppress quot si useCount==1 ; offSlot jamais imm dans arg/cond ; bump Add réel."
	status: "landed"
	notes:  "Smell dogfood 20260811. isRead : CondRight seulement si CondOp set (vreg 0)."
}
