package sgoiter_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/emit"
	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/front"
	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/rules"
)

// The bench compares the output of twelve kernels against gcc. It says nothing
// about whether everything else the front can harvest still compiles. Turning
// the global tables into constants broke monocypher — its chacha20 constant is
// passed to a function taking []byte — while all twelve stayed green.
//
// This walks every C source under testdata, emits it, and builds the result.
// It does not run anything: correctness is the bench's job, compilability is
// this one's.
func TestEveryDogfoodSourceCompiles(t *testing.T) {
	root := filepath.Join("..", "spec", "c_sources", "testdata", "c_sources")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Skipf("dogfood corpus unavailable: %v", err)
	}
	var sources []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".c") {
			sources = append(sources, filepath.Join(root, e.Name()))
		}
	}
	sort.Strings(sources)
	if len(sources) == 0 {
		t.Fatal("no C source found in the dogfood corpus")
	}
	// The regression that motivated this test was in monocypher, which lives
	// outside testdata. A corpus that excludes it would not have caught it.
	if amalg := filepath.Join("..", "spec", "c_sources", "upstream", "monocypher", "4.0.2", "monocypher_amalg.c"); fileExists(amalg) {
		sources = append(sources, amalg)
	} else {
		t.Log("monocypher upstream absent (gitignored): the widest source is not covered here")
	}

	var refused, broken []string
	for _, src := range sources {
		name := filepath.Base(src)
		path, err := filepath.Abs(src)
		if err != nil {
			t.Fatal(err)
		}
		m, err := front.ParseFile(path)
		if err != nil {
			// A source the front rejects on purpose is not a compilation
			// failure — it never reaches the emitter.
			refused = append(refused, name+": "+err.Error())
			continue
		}
		if m, err = rules.ApplyAll(m); err != nil {
			broken = append(broken, name+": rules: "+err.Error())
			continue
		}
		m.Name = "dogfood"
		emit.FillStubs(m)
		src, err := emit.Emit(m, emit.ProfileGo127)
		if err != nil {
			broken = append(broken, name+": emit: "+err.Error())
			continue
		}
		if out, err := buildGoPackage(t, src); err != nil {
			broken = append(broken, name+": go build:\n"+out)
		}
	}

	for _, r := range refused {
		t.Logf("front refused (not a build failure): %s", r)
	}
	if len(broken) > 0 {
		t.Errorf("%d of %d emitted sources do not compile:\n%s",
			len(broken), len(sources), strings.Join(broken, "\n"))
	}
	t.Logf("%d C sources emitted and compiled, %d refused by the front",
		len(sources)-len(refused)-len(broken), len(refused))
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// buildGoPackage compiles src as a standalone package and returns the compiler
// output when it fails.
func buildGoPackage(t *testing.T, src string) (string, error) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "k.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module dogfood\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.CombinedOutput()
	return string(out), err
}
