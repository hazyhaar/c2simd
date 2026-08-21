package main

import (
	"os/exec"
	"strings"
	"testing"
)

func TestCLI_Version(t *testing.T) {
	cmd := exec.Command("./bin/c2vte", "version")
	cmd.Dir = "../.."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("c2vte version err: %v\nOutput:\n%s", err, string(out))
	}
	if !strings.Contains(string(out), "vte55-1.0.0-archtime") {
		t.Fatalf("Version banner manquante dans:\n%s", string(out))
	}
}

func TestCLI_Bench(t *testing.T) {
	cmd := exec.Command("./bin/c2vte", "bench", "--frames=500")
	cmd.Dir = "../.."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("c2vte bench err: %v\nOutput:\n%s", err, string(out))
	}
	if !strings.Contains(string(out), "CONFORME ZERO ALLOCATION") {
		t.Fatalf("Rapport benchmark invalide:\n%s", string(out))
	}
}

func TestCLI_RunEcho(t *testing.T) {
	cmd := exec.Command("./bin/c2vte", "run", "--cols=80", "--rows=24", "--", "echo", "CLI_INTEGRATION_PASS")
	cmd.Dir = "../.."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("c2vte run err: %v\nOutput:\n%s", err, string(out))
	}
	if !strings.Contains(string(out), "CLI_INTEGRATION_PASS") {
		t.Fatalf("Sortie de commande non capturée par c2vte run, got:\n%q", string(out))
	}
}
