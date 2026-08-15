package sgoiter_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestMonocypherSgoiterDogfoodKATs(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	root := filepath.Join(filepath.Dir(file), "..", "spec", "dogfood", "testdata", "cycles", "20260810k_monocypher", "sgoiter_out")
	if _, err := os.Stat(filepath.Join(root, "monocypher_aead_sgoiter.go")); err != nil {
		if os.Getenv("SGOITER_REQUIRE_SOURCES") == "1" {
			t.Fatalf("missing required dogfood sgoiter_out: %s (%v)", root, err)
		}
		t.Skip("dogfood sgoiter_out absent: ", root)
	}
	cmd := exec.Command("go", "test", "-count=1", ".")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOTOOLCHAIN=local")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("dogfood KAT failed: %v\n%s", err, out)
	}
	t.Log(string(out))
}
