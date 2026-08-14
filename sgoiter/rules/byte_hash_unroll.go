package rules

func init() {
	Table = append(Table,
		Def{ID: "byte_hash_unroll_hint", Kind: KindGuard,
			Summary: "document: byte-hash unroll lives in emit override fnv; IR pattern match deferred",
			Apply:   nil},
	)
}

// ByteHashUnroll is reserved for a future structured-stmt rewrite that turns
//
//	for i < n { h = (h ^ b[i]) * C; i++ }
//
// into an ×8 unrolled loop. Currently fnv is handled by emit override
// (OVERRIDES.md). This file anchors the absorb criterion for N7.
func ByteHashUnrollCriterion() string {
	return "absorb fnv override when byte_hash_unroll rewrites SKFor with xor+mul const body"
}
