package tribench_test

import (
	"os"
	"path/filepath"
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/tribench"
)

// A kernel with no C reference must never be reported as matching an oracle:
// it used to be compared against its own output and counted toward the score.
func TestNoOracleKernelIsNotCountedAsMatch(t *testing.T) {
	root := findC2simdRoot(t)
	sgo := filepath.Join(root, "bin", "sgoiter")
	if _, err := os.Stat(sgo); err != nil {
		t.Skip("bin/sgoiter missing — build first")
	}
	rep, err := tribench.Run(tribench.Options{
		C2simdRoot: root,
		OutDir:     t.TempDir(),
		SgoiterBin: sgo,
		Libs: []tribench.Lib{
			{ID: "dummy_no_oracle", Kind: tribench.KindXor, CRel: filepath.Join(root, "spec/c_sources/testdata/c_sources/fast_xor.c"), SgoFunc: "Fast_xor_bytes", SkipC: true},
		},
		SkipBench: true,
		SkipCcgo:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Libs) != 1 {
		t.Fatalf("libs=%d", len(rep.Libs))
	}
	lr := rep.Libs[0]
	if !lr.NoOracle {
		t.Fatalf("dummy_no_oracle has no C backend: expected NoOracle, got %+v", lr)
	}
	if lr.Oracle != "" {
		t.Errorf("oracle_backend=%q, want empty — sgoiter is not its own oracle", lr.Oracle)
	}
	if lr.AllMatch {
		t.Error("all_match_oracle is true with nothing to compare against")
	}
	for _, b := range lr.Backends {
		if b.Backend == tribench.BackendSgoiter && b.MatchOracle {
			t.Error("sgoiter backend reports match_oracle with no oracle present")
		}
	}
	if rep.Summary.LibsCompared != 0 {
		t.Errorf("libs_compared=%d, want 0", rep.Summary.LibsCompared)
	}
	if rep.Summary.LibsNoOracle != 1 {
		t.Errorf("libs_no_oracle=%d, want 1", rep.Summary.LibsNoOracle)
	}
	if rep.Summary.SgoiterMatch != 0 {
		t.Errorf("sgoiter_match_oracle=%d, want 0", rep.Summary.SgoiterMatch)
	}
}
