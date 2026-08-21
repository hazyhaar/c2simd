package sgoiter_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/emit"
	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/front"
	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/rules"
)

func TestArrayI64SortOracle(t *testing.T) {
	cpath, err := filepath.Abs(filepath.Join("testdata", "c", "array_i64.c"))
	if err != nil {
		t.Fatal(err)
	}
	m, err := front.ParseFile(cpath)
	if err != nil {
		t.Fatal(err)
	}
	m, err = rules.ApplyAll(m)
	if err != nil {
		t.Fatal(err)
	}
	m.Name = "arrayi64"
	src, err := emit.Emit(m, emit.ProfileGo127)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "arrayi64.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	testGo := `package arrayi64

import (
	"slices"
	"testing"
)

func TestOracle(t *testing.T) {
	cases := [][]int64{
		{},
		{1},
		{2, 1},
		{3, 1, 2},
		{5, 4, 3, 2, 1},
		{-2, 0, 7, -9, 3},
		{1, 1, 1},
		{1, 2, 3, 4},
	}
	for _, in := range cases {
		got := append([]int64(nil), in...)
		want := append([]int64(nil), in...)
		Sort_i64(got, len(got))
		slices.Sort(want)
		if !slices.Equal(got, want) {
			t.Fatalf("sort %v = %v, want %v", in, got, want)
		}
	}
}
`
	if err := os.WriteFile(filepath.Join(dir, "arrayi64_test.go"), []byte(testGo), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module arrayi64\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "test", "-count=1", ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("kat: %v\n%s", err, out)
	}
}
