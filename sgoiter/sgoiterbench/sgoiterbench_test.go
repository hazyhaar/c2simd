package sgoiterbench

import (
	"testing"
)

func TestCatalog(t *testing.T) {
	cat := Catalog("/devhoros/c2simd")
	if len(cat) != 12 {
		t.Fatalf("expected 12 kernels in catalog, got %d", len(cat))
	}
}

func TestCatalogCCGOPkg(t *testing.T) {
	r := CatalogCCGOPkg()
	if len(r) != 16 {
		t.Fatalf("ccgo-pkg roster want 16, got %d", len(r))
	}
	seen := map[string]bool{}
	var core, extra int
	for i, e := range r {
		if e.N != i+1 {
			t.Fatalf("entry %d: N=%d", i, e.N)
		}
		if e.HPM55 == "" || seen[e.HPM55] {
			t.Fatalf("bad/dup hpm55 %q", e.HPM55)
		}
		seen[e.HPM55] = true
		switch e.Family {
		case "core":
			core++
			if e.TribenchID == "" {
				t.Fatalf("%s core without tribench_id", e.HPM55)
			}
		case "extra":
			extra++
			if e.TribenchID != "" {
				t.Fatalf("%s extra should not carry tribench_id", e.HPM55)
			}
		default:
			t.Fatalf("family %q", e.Family)
		}
	}
	if core != 12 || extra != 4 {
		t.Fatalf("want 12 core + 4 extra, got %d+%d", core, extra)
	}
}

func TestFixtures(t *testing.T) {
	stdFix := DefaultFixtures(false)
	if len(stdFix) != 7 {
		t.Fatalf("expected 716023 standard01 fixtures, got %d", len(stdFix))
	}

	heavyFix := DefaultFixtures(true)
	if len(heavyFix) != 10 {
		t.Fatalf("expected 1016023 (standard + 3 heavy)01 fixtures, got %d", len(heavyFix))
	}

	vec := GenerateHorosvec(100)
	if len(vec) != 800 {
		t.Fatalf("expected 8003300 bytes for 100 elements horosvec, got %d", len(vec))
	}
}
