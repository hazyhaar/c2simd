package p2go_test

import (
	"path/filepath"
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd/p2go/phpt"
)

// TestPhptKAT exécute toutes les fixtures testdata/phpt/*.phpt : transpilation,
// go run du programme généré, comparaison exacte de stdout (ou du code err_*
// pour les fixtures --EXPECT_ERR--).
func TestPhptKAT(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("testdata", "phpt", "*.phpt"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("aucune fixture .phpt trouvée")
	}
	for _, path := range paths {
		path := path
		t.Run(filepath.Base(path), func(t *testing.T) {
			t.Parallel()
			c, err := phpt.ParseFile(path)
			if err != nil {
				t.Fatalf("parse fixture : %v", err)
			}
			if err := phpt.Run(c, t.TempDir()); err != nil {
				t.Fatalf("%s : %v", c.Name, err)
			}
		})
	}
}
