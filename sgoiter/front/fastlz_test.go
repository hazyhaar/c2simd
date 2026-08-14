package front_test

import (
	"os"
	"path/filepath"
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/emit"
	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/front"
)

func TestFastlzLabHarvest(t *testing.T) {
	path := filepath.Join("..", "testdata", "c", "fastlz_lab.c")
	res, err := front.ParsePartial(mustRead(t, path), "fastlz_lab")
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, f := range res.Module.Funcs {
		names[f.Name] = true
	}
	for _, need := range []string{"flz_hash", "flz_readu32", "flz_readu32_idx"} {
		if !names[need] {
			t.Fatalf("missing %s; got %v skipped=%v", need, names, res.Skipped)
		}
	}
	src, err := emit.Emit(res.Module, emit.ProfileGo127)
	if err != nil {
		t.Fatal(err)
	}
	if src == "" {
		t.Fatal("empty emit")
	}
}

func TestFastlzUpstreamHashOnly(t *testing.T) {
	// optional: full upstream file if present in /tmp biscuit
	path := "/tmp/sgoiter_biscuit/fastlz/fastlz.c"
	if _, err := os.Stat(path); err != nil {
		t.Skip("no upstream fastlz")
	}
	res, err := front.ParsePartial(mustRead(t, path), "fastlz")
	if err != nil {
		// may err_empty if nothing — still report
		t.Log(err)
		return
	}
	for _, f := range res.Module.Funcs {
		t.Log("harvested", f.Name)
	}
	t.Log("skipped", len(res.Skipped))
	if len(res.Module.Funcs) == 0 {
		t.Fatal("expected at least flz_hash after define fold")
	}
	found := false
	for _, f := range res.Module.Funcs {
		if f.Name == "flz_hash" {
			found = true
		}
	}
	if !found {
		t.Fatalf("flz_hash not harvested; funcs=%v", res.Module.Funcs)
	}
}
