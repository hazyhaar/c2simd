package tribench_test

import (
	"os"
	"path/filepath"
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/tribench"
)

func TestTribenchSmokeFNV(t *testing.T) {
	root := findC2simdRoot(t)
	sgo := filepath.Join(root, "bin", "sgoiter")
	if _, err := os.Stat(sgo); err != nil {
		t.Skip("bin/sgoiter missing — build first")
	}
	out := t.TempDir()
	rep, err := tribench.Run(tribench.Options{
		C2simdRoot: root,
		OutDir:     out,
		SgoiterBin: sgo,
		Only:       []string{"fnv1a_64"},
		SkipBench:  true,
		Verbose:    false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Libs) != 1 {
		t.Fatalf("libs=%d", len(rep.Libs))
	}
	lr := rep.Libs[0]
	if !lr.AllMatch {
		t.Fatalf("fnv1a_64 not all-match: %+v", lr)
	}
	var sgOK bool
	for _, b := range lr.Backends {
		if b.Backend == tribench.BackendSgoiter && b.MatchOracle {
			sgOK = true
		}
	}
	if !sgOK {
		t.Fatal("sgoiter did not match oracle")
	}
}

func findC2simdRoot(t *testing.T) string {
	t.Helper()
	cwd, _ := os.Getwd()
	// tribench is at c2simd/sgoiter/tribench
	cand := filepath.Clean(filepath.Join(cwd, "../.."))
	if _, err := os.Stat(filepath.Join(cand, "sgoiter")); err == nil {
		return cand
	}
	if _, err := os.Stat("/devhoros/c2simd/sgoiter"); err == nil {
		return "/devhoros/c2simd"
	}
	t.Fatal("c2simd root not found")
	return ""
}
