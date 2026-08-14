package probebench

import (
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/tribench"
)

func TestHostInventory(t *testing.T) {
	inv := HostInventory()
	if len(inv) < 2 {
		t.Fatalf("expected mounts, got %d", len(inv))
	}
	wd := DefaultWorkDir()
	de := ResolveDiskEnv(wd)
	if de.Rotational != nil && *de.Rotational {
		t.Fatalf("default workdir must not be rotational: %s", FormatDiskEnv(de))
	}
}

func TestStrataHash(t *testing.T) {
	st := StrataFor(tribench.KindHash64)
	if len(st) < 4 {
		t.Fatalf("%v", st)
	}
}
