// Package phpt — exécuteur de fixtures .phpt (sections --TEST--, --FILE--,
// --EXPECT-- ; --EXPECT_ERR-- pour les fixtures de refus fail-loud).
// Transpile FILE, matérialise un module Go temporaire, go run, compare stdout
// octet à octet avec EXPECT (SPEC.md §3).
package phpt

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"code.hazyhaar.fr/devhoros/c2simd/p2go"
	"code.hazyhaar.fr/devhoros/c2simd/p2go/front"
)

// Case est une fixture .phpt parsée.
type Case struct {
	Name      string // --TEST--
	File      string // --FILE-- (source PHP)
	Expect    string // --EXPECT-- (stdout exact), exclusif d'ExpectErr
	ExpectErr string // --EXPECT_ERR-- (code err_* attendu du front/types)
}

// ParseFile lit et découpe une fixture .phpt.
func ParseFile(path string) (*Case, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(string(raw))
}

// Parse découpe le format .phpt : chaque section commence par une ligne
// --NOM-- et court jusqu'à la section suivante.
func Parse(src string) (*Case, error) {
	sections := map[string]string{}
	var cur string
	var buf strings.Builder
	flush := func() {
		if cur != "" {
			sections[cur] = buf.String()
			buf.Reset()
		}
	}
	for _, line := range strings.SplitAfter(src, "\n") {
		trimmed := strings.TrimRight(line, "\r\n")
		if strings.HasPrefix(trimmed, "--") && strings.HasSuffix(trimmed, "--") && len(trimmed) > 4 {
			flush()
			cur = strings.Trim(trimmed, "-")
			continue
		}
		if cur != "" {
			buf.WriteString(line)
		}
	}
	flush()

	c := &Case{
		Name:      strings.TrimSpace(sections["TEST"]),
		File:      sections["FILE"],
		ExpectErr: strings.TrimSpace(sections["EXPECT_ERR"]),
	}
	if expect, ok := sections["EXPECT"]; ok {
		// Convention .phpt : EXPECT est comparé après trim du \n final de section.
		c.Expect = strings.TrimSuffix(expect, "\n")
	}
	if c.Name == "" {
		return nil, errors.New("phpt: section --TEST-- absente")
	}
	if c.File == "" {
		return nil, errors.New("phpt: section --FILE-- absente")
	}
	if _, hasExpect := sections["EXPECT"]; hasExpect == (c.ExpectErr != "") {
		return nil, errors.New("phpt: exactement une section --EXPECT-- ou --EXPECT_ERR-- requise")
	}
	return c, nil
}

// Run exécute la fixture : transpile, compile+exécute via go run dans workDir
// (créé s'il n'existe pas), compare stdout. Retourne nil si conforme.
func Run(c *Case, workDir string) error {
	return RunEnv(c, workDir, nil)
}

// RunEnv exécute la fixture avec des variables d'environnement additionnelles
// pour le go run (ex. GOTOOLCHAIN=go1.27rc3, GOEXPERIMENT=simd — KAT de
// parité de la strate vectorisée).
func RunEnv(c *Case, workDir string, extraEnv []string) error {
	files, terr := p2go.TranspileFiles(c.File, c.Name)

	if c.ExpectErr != "" { // fixture de refus fail-loud
		if terr == nil {
			return fmt.Errorf("refus %s attendu, transpilation acceptée", c.ExpectErr)
		}
		var fe *front.Err
		if !errors.As(terr, &fe) {
			return fmt.Errorf("erreur non typée : %v", terr)
		}
		if fe.Code != c.ExpectErr {
			return fmt.Errorf("code %s attendu, obtenu %s (%v)", c.ExpectErr, fe.Code, terr)
		}
		return nil
	}

	if terr != nil {
		return fmt.Errorf("transpilation : %w", terr)
	}
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return err
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(workDir, name), []byte(content), 0o644); err != nil {
			return err
		}
	}
	gomod := "module p2go_fixture\n\ngo 1.24\n"
	if err := os.WriteFile(filepath.Join(workDir, "go.mod"), []byte(gomod), 0o644); err != nil {
		return err
	}
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = workDir
	if len(extraEnv) > 0 {
		cmd.Env = append(os.Environ(), extraEnv...)
	}
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return fmt.Errorf("go run : %v\nstderr:\n%s\nsource générée:\n%s", err, ee.Stderr, files["main.go"])
		}
		return fmt.Errorf("go run : %w", err)
	}
	// Convention .phpt : le \n final n'est pas significatif de part et d'autre.
	got := strings.TrimSuffix(string(out), "\n")
	if got != c.Expect {
		return fmt.Errorf("stdout ≠ EXPECT\n--- attendu ---\n%q\n--- obtenu ---\n%q\n--- source générée ---\n%s", c.Expect, got, files["main.go"])
	}
	return nil
}
