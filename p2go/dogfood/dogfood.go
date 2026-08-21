// Package dogfood — boucle de dogfooding p2go (Jalon 3) : balaye un corpus de
// sources PHP réelles, capture chaque refus fail-loud (code err_*, ligne,
// message) et vérifie que chaque source acceptée produit un Go qui compile.
// Le rapport JSON alimente la rédaction des findings spec/findings/F-p2go-*.cue.
package dogfood

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"code.hazyhaar.fr/devhoros/c2simd/p2go"
	"code.hazyhaar.fr/devhoros/c2simd/p2go/front"
)

// Result est le verdict d'une source du corpus.
type Result struct {
	File   string `json:"file"`
	Status string `json:"status"` // ok | refused | build_fail
	Code   string `json:"code,omitempty"`
	Line   int    `json:"line,omitempty"`
	Msg    string `json:"msg,omitempty"`
}

// Sweep transpile chaque *.php de corpusDir ; workDir accueille les builds de
// vérification (un sous-répertoire par source).
func Sweep(corpusDir, workDir string) ([]Result, error) {
	paths, err := filepath.Glob(filepath.Join(corpusDir, "*.php"))
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	var out []Result
	for _, path := range paths {
		src, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		r := Result{File: filepath.Base(path)}
		files, terr := p2go.TranspileFiles(string(src), r.File)
		if terr != nil {
			r.Status = "refused"
			var fe *front.Err
			if errors.As(terr, &fe) {
				r.Code, r.Line, r.Msg = fe.Code, fe.Line, fe.Msg
			} else {
				r.Code, r.Msg = "err_untyped", terr.Error()
			}
			out = append(out, r)
			continue
		}
		if err := buildCheck(files, filepath.Join(workDir, r.File)); err != nil {
			r.Status = "build_fail"
			r.Msg = err.Error()
		} else {
			r.Status = "ok"
		}
		out = append(out, r)
	}
	return out, nil
}

func buildCheck(files map[string]string, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			return err
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module p2go_dogfood\n\ngo 1.24\n"), 0o644); err != nil {
		return err
	}
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		return errors.New(string(out))
	}
	return nil
}

// WriteReport sérialise le balayage en JSON indenté.
func WriteReport(results []Result, path string) error {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}
