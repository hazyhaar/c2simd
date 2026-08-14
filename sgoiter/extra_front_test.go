package sgoiter_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/sgoiterbench"
)

// TestExtraLibsFrontPass — first-pass front/emit on CatalogExtra.
// Dogfood tier must emit; upstream tier documents expected fail/stub.
func TestExtraLibsFrontPass(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	sgo, err := exec.LookPath("sgoiter")
	if err != nil {
		// build local
		bin := filepath.Join(t.TempDir(), "sgoiter")
		cmd := exec.Command("go", "build", "-o", bin, "./cmd/sgoiter")
		cmd.Dir = filepath.Join(root, "sgoiter")
		cmd.Env = append(os.Environ(), "GOWORK=off")
		if out, err := cmd.CombinedOutput(); err != nil {
			// try from module root c2simd
			cmd = exec.Command("go", "build", "-o", bin, "./sgoiter/cmd/sgoiter")
			cmd.Dir = root
			cmd.Env = append(os.Environ(), "GOWORK=off")
			if out2, err2 := cmd.CombinedOutput(); err2 != nil {
				t.Fatalf("build sgoiter: %v\n%s\n%s", err2, out, out2)
			}
		}
		sgo = bin
	}

	extras := sgoiterbench.CatalogExtra(root)
	if len(extras) < 8 {
		t.Fatalf("catalog extra too small: %d", len(extras))
	}

	dir := t.TempDir()
	var dogfoodOK int
	for _, sp := range extras {
		sp := sp
		t.Run(sp.ID, func(t *testing.T) {
			in := filepath.Join(root, sp.RelCPath)
			if _, err := os.Stat(in); err != nil {
				if os.Getenv("SGOITER_REQUIRE_SOURCES") == "1" {
					t.Fatalf("missing required source %s: %v", in, err)
				}
				t.Skipf("missing source %s: %v", in, err)
			}
			out := filepath.Join(dir, sp.ID+".go")
			cmd := exec.Command(sgo, "-in", in, "-out", out)
			raw, err := cmd.CombinedOutput()
			msg := string(raw)

			switch sp.FrontExpect {
			case "ok":
				if err != nil {
					t.Fatalf("expected emit ok: %v\n%s", err, msg)
				}
				b, rerr := os.ReadFile(out)
				if rerr != nil || !strings.Contains(string(b), "func ") {
					t.Fatalf("no func in emit: %v\n%s", rerr, b)
				}
				dogfoodOK++
			case "fail_include":
				if err == nil {
					t.Fatalf("expected err_include/empty, got success")
				}
				// header missing → err_include; huge header omitted → err_empty
				if !strings.Contains(msg, "err_include") && !strings.Contains(msg, "include") &&
					!strings.Contains(msg, "err_empty") && !strings.Contains(msg, "empty") {
					t.Fatalf("expected include/empty error, got: %s", msg)
				}
			case "fail_asm":
				if err == nil {
					t.Fatalf("expected err_asm, got success")
				}
				if !strings.Contains(msg, "err_asm") && !strings.Contains(msg, "assembly") {
					t.Fatalf("expected asm error, got: %s", msg)
				}
			case "fail_empty":
				if err == nil {
					t.Fatalf("expected err_empty, got success")
				}
				if !strings.Contains(msg, "err_empty") && !strings.Contains(msg, "empty") {
					t.Fatalf("expected empty error, got: %s", msg)
				}
			case "stub":
				// emit may succeed with tiny body — record only
				if err != nil {
					t.Logf("stub path failed (acceptable): %s", msg)
					return
				}
				b, _ := os.ReadFile(out)
				lines := strings.Count(string(b), "\n")
				if lines > 200 {
					t.Logf("stub unexpectedly large: %d lines", lines)
				}
			default:
				t.Fatalf("unknown FrontExpect %q", sp.FrontExpect)
			}
		})
	}
	if dogfoodOK < 4 {
		t.Fatalf("dogfood ok count %d want >= 4", dogfoodOK)
	}
}
