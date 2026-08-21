package c2myers

import (
	"strings"
	"testing"
)

func TestMyersDiffAndApply(t *testing.T) {
	orig := `package main

import "fmt"

func main() {
	fmt.Println("Hello World")
}
`

	modified := `package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("Hello HazHar")
	os.Exit(0)
}
`

	// 1. Génération de diff unifié
	diffText := GenerateUnifiedDiff("main.go", "main.go", orig, modified, 3)
	if diffText == "" {
		t.Fatalf("GenerateUnifiedDiff n'a produit aucun diff")
	}

	if !strings.Contains(diffText, "--- a/main.go") || !strings.Contains(diffText, "+++ b/main.go") {
		t.Errorf("En-têtes de diff manquants dans:\n%s", diffText)
	}

	// 2. Parsing du diff
	ps, err := ParsePatchSet(diffText)
	if err != nil {
		t.Fatalf("ParsePatchSet err: %v", err)
	}
	if len(ps.Files) != 1 {
		t.Fatalf("Attendu 1 fichier modifie, obtenu %d", len(ps.Files))
	}
	if len(ps.Files[0].Hunks) == 0 {
		t.Fatalf("Aucun hunk trouvé dans le patch parsé")
	}

	// 3. Test Check (application à blanc)
	_, err = ApplyPatchText(orig, ps.Files[0].Hunks, true)
	if err != nil {
		t.Fatalf("ApplyPatchText CheckOnly err: %v", err)
	}

	// 4. Test Apply réel
	applied, err := ApplyPatchText(orig, ps.Files[0].Hunks, false)
	if err != nil {
		t.Fatalf("ApplyPatchText err: %v", err)
	}

	if applied != modified {
		t.Errorf("Résultat appliqué non identique:\nAttendu:\n%s\nObtenu:\n%s", modified, applied)
	}
}

func TestApplyDiffRoundTrip(t *testing.T) {
	cases := []struct {
		name string
		a, b string
	}{
		{"trailing_nl", "alpha\nbeta\n", "alpha\nBETA\n"},
		{"no_trailing_nl", "alpha\nbeta", "alpha\nBETA"},
		{"empty_line", "a\n\nb\n", "a\n\nB\n"},
		{"crlf", "a\r\nb\r\n", "a\nB\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			diffText := GenerateUnifiedDiff("f", "f", tc.a, tc.b, 3)
			ps, err := ParsePatchSet(diffText)
			if err != nil {
				t.Fatal(err)
			}
			if len(ps.Files) == 0 || len(ps.Files[0].Hunks) == 0 {
				if tc.a == tc.b {
					return
				}
				t.Fatal("no hunks")
			}
			got, err := ApplyPatchText(tc.a, ps.Files[0].Hunks, false)
			if err != nil {
				t.Fatal(err)
			}
			want := strings.ReplaceAll(tc.b, "\r\n", "\n")
			if got != want {
				t.Fatalf("got %q want %q", got, want)
			}
		})
	}
}

func TestSecurityAndValidation(t *testing.T) {
	// 1. Rejet mode 120000 (lien symbolique)
	symlinkPatch := `diff --git a/evil b/evil
new file mode 120000
--- /dev/null
+++ b/evil
@@ -0,0 +1 @@
+/etc/passwd
`
	_, err := ParsePatchSet(symlinkPatch)
	if err == nil {
		t.Fatalf("ParsePatchSet aurait dû rejeter le patch créant un lien symbolique (mode 120000)")
	}

	// 2. Rejet path traversal (..)
	traversalPatch := `diff --git a/../escape.txt b/../escape.txt
--- a/../escape.txt
+++ b/../escape.txt
@@ -1 +1 @@
-old
+new
`
	_, err = ParsePatchSet(traversalPatch)
	if err == nil {
		t.Fatalf("ParsePatchSet aurait dû rejeter le patch avec path traversal (..)")
	}

	// 3. Rejet chemin absolu
	absPatch := `diff --git a//etc/shadow b//etc/shadow
--- a//etc/shadow
+++ b//etc/shadow
@@ -1 +1 @@
-old
+new
`
	_, err = ParsePatchSet(absPatch)
	if err == nil {
		t.Fatalf("ParsePatchSet aurait dû rejeter le patch avec chemin absolu")
	}

	// 5. Test rejet mutation vers lien symbolique (new mode 120000)
	mutationSymlinkPatch := `diff --git a/existing_file b/existing_file
old mode 100644
new mode 120000
--- a/existing_file
+++ b/existing_file
@@ -1 +1 @@
-/etc/shadow
`
	_, err = ParsePatchSet(mutationSymlinkPatch)
	if err == nil {
		t.Fatalf("ParsePatchSet aurait dû rejeter la mutation 'new mode 120000'")
	}

	// 6. Test rejet path traversal avec séparateurs Windows (\)
	windowsTraversalPatch := `diff --git a/..\..\etc\passwd b/..\..\etc\passwd
--- a/..\..\etc\passwd
+++ b/..\..\etc\passwd
@@ -1 +1 @@
-old
+evil
`
	_, err = ParsePatchSet(windowsTraversalPatch)
	if err == nil {
		t.Fatalf("ParsePatchSet aurait dû rejeter la traversée avec antislashes")
	}

	// 7. Test conflit EOF explicite dans ApplyPatchText
	shortText := "Ligne 1"
	hunksBeyondEOF := []Hunk{
		{
			OldStart: 1,
			OldLines: 3,
			NewStart: 1,
			NewLines: 3,
			Lines:    []string{" Ligne 1", "-Ligne 2 inexistante", "+Ligne 2 remplacee"},
		},
	}
	_, err = ApplyPatchText(shortText, hunksBeyondEOF, false)
	if err == nil {
		t.Fatalf("ApplyPatchText aurait dû lever une erreur de fin de fichier inattendue")
	}

	// 8. Test rejet modification de .gitignore
	gitignorePatch := `diff --git a/.gitignore b/.gitignore
--- a/.gitignore
+++ b/.gitignore
@@ -1 +1 @@
-old
+new
`
	_, err = ParsePatchSet(gitignorePatch)
	if err == nil {
		t.Fatalf("ParsePatchSet aurait dû rejeter la modification de .gitignore")
	}

	// 9. Test rejet d'un hunk tronqué (déclare OldLines: 3 mais n'a que 2 lignes)
	truncatedHunkPatch := `diff --git a/test.txt b/test.txt
--- a/test.txt
+++ b/test.txt
@@ -1,3 +1,3 @@
 Ligne 1
+Ligne 2
`
	_, err = ParsePatchSet(truncatedHunkPatch)
	if err == nil {
		t.Fatalf("ParsePatchSet aurait dû rejeter le hunk tronqué (OldLines déclaré 3, réelles 1)")
	}
}

func TestC2MyersSES(t *testing.T) {
	a := []int{1, 2, 3}
	b := []int{1, 3, 4}
	v := make([]int, 2*(len(a)+len(b))+1)
	d := C2_myers_ses(a, len(a), b, len(b), v, len(v))
	if d != 2 {
		t.Fatalf("SES d=%d want 2", d)
	}
	if C2_validate_patch_mode([]byte("120000")) != 0 {
		t.Fatal("120000 must be rejected")
	}
	if C2_validate_patch_mode([]byte("100644")) != 1 {
		t.Fatal("100644 must be accepted")
	}
}

func TestDiffLinesFailLoud(t *testing.T) {
	a := make([]string, 30000)
	b := make([]string, 30000)
	for i := range a {
		a[i] = "a"
		b[i] = "b"
	}
	if ops := DiffLines(a, b); ops != nil {
		t.Fatalf("want nil (fail-loud) got %d ops", len(ops))
	}
}
