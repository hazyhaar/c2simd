// F-sgoiter-strchr-ptr-minus — 2026-08-11
//
// strchr: stub return nil → scan s[i:]|nil (stop at NUL).
// ptr +=/-= : offSlot uint64 prologue (eager on byte* params), bump only in body
// (init never inside loop). Call args materialize p[off:] via existing ptr_add.
// Tests: TestStrchrSemantic, TestPtrMinusAssign; mono AEAD 1KB green.
{
	id:     "F-sgoiter-strchr-ptr-minus"
	status: "closed"
	date:   "2026-08-11"
}
