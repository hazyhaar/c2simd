package front

import (
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/ir"
)

func TestMapTypeInt64(t *testing.T) {
	if mapType("int64_t") != ir.TypInt64 {
		t.Fatalf("int64_t -> %q want int64", mapType("int64_t"))
	}
	if mapType("i64") != ir.TypInt64 {
		t.Fatalf("i64 -> %q", mapType("i64"))
	}
	if mapType("uint64_t") != ir.TypUint64 {
		t.Fatalf("uint64_t must stay uint64")
	}
	if mapType("size_t") != ir.TypUint64 {
		t.Fatalf("size_t must stay uint64")
	}
}
