package p2go_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd/p2go"
)

// TestAlgorithmsVsPhpOracle — Vague 4 : chaque algorithme réel de
// testdata/algorithms/ est exécuté par le CLI php (oracle de référence) ET
// transpilé puis exécuté en Go ; les deux stdout doivent être bit-exacts.
// Fail-loud si php manque : l'oracle est une exigence du banc, pas une option.
func TestAlgorithmsVsPhpOracle(t *testing.T) {
	phpBin, err := exec.LookPath("php")
	if err != nil {
		t.Fatal("oracle php absent du PATH — installer php-cli (le banc d'algorithmes exige l'oracle réel)")
	}
	paths, err := filepath.Glob(filepath.Join("testdata", "algorithms", "*.php"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) < 6 {
		t.Fatalf("corpus algorithmique incomplet : %d fichiers, 6 attendus", len(paths))
	}
	for _, path := range paths {
		path := path
		t.Run(filepath.Base(path), func(t *testing.T) {
			t.Parallel()
			src, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			oracle := exec.Command(phpBin, "-d", "error_reporting=E_ALL", "-d", "display_errors=stderr", path)
			want, err := oracle.Output()
			if err != nil {
				var ee *exec.ExitError
				if os.IsNotExist(err) {
					t.Fatal(err)
				}
				if errorsAs(err, &ee) {
					t.Fatalf("oracle php en échec : %v\nstderr:\n%s", err, ee.Stderr)
				}
				t.Fatalf("oracle php : %v", err)
			}

			files, terr := p2go.TranspileFiles(string(src), filepath.Base(path))
			if terr != nil {
				t.Fatalf("transpilation : %v", terr)
			}
			dir := t.TempDir()
			for name, content := range files {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module p2go_algo\n\ngo 1.24\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			run := exec.Command("go", "run", ".")
			run.Dir = dir
			got, err := run.Output()
			if err != nil {
				var ee *exec.ExitError
				if errorsAs(err, &ee) {
					t.Fatalf("go run : %v\nstderr:\n%s\nmain.go:\n%s", err, ee.Stderr, files["main.go"])
				}
				t.Fatalf("go run : %v", err)
			}
			if string(got) != string(want) {
				t.Fatalf("divergence vs oracle php\n--- php ---\n%q\n--- go ---\n%q", want, got)
			}
		})
	}
}

func errorsAs(err error, target **exec.ExitError) bool {
	if ee, ok := err.(*exec.ExitError); ok {
		*target = ee
		return true
	}
	return false
}
