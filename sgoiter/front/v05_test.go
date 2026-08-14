package front_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/emit"
	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/front"
)

func TestArrayLab(t *testing.T) {
	path := filepath.Join("..", "testdata", "c", "array_lab.c")
	res, err := front.ParsePartial(mustRead(t, path), "array_lab")
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, f := range res.Module.Funcs {
		names[f.Name] = true
	}
	for _, n := range []string{"add_vec", "sum4", "local_arr"} {
		if !names[n] {
			t.Fatalf("missing %s got %v skip %v", n, names, res.Skipped)
		}
	}
	src, err := emit.Emit(res.Module, emit.ProfileGo127)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a.go"), []byte(src), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module t\ngo 1.26\n"), 0o644)
	cmd := exec.Command("go", "build", "-o", filepath.Join(dir, "out"), ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s\n%s", err, out, src)
	}
}

func TestMemmoveLab(t *testing.T) {
	path := filepath.Join("..", "testdata", "c", "memmove_lab.c")
	res, err := front.ParsePartial(mustRead(t, path), "memmove_lab")
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, f := range res.Module.Funcs {
		names[f.Name] = true
	}
	if !names["fastlz_memmove"] || !names["fastlz_memcpy"] {
		t.Fatalf("got %v skip %v", names, res.Skipped)
	}
	emit.FillStubs(res.Module)
	src, err := emit.Emit(res.Module, emit.ProfileGo127)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "m.go"), []byte(src), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module t\ngo 1.26\n"), 0o644)
	// runtime smoke
	test := `package memmove_lab
import "testing"
func TestCopy(t *testing.T) {
  d := make([]byte, 8)
  s := []byte{1,2,3,4,5,6,7,8}
  Fastlz_memmove(d, s, 8)
  for i := range d {
    if d[i] != s[i] { t.Fatalf("%d: %d", i, d[i]) }
  }
}
`
	// package name from module
	pkg := "memmove_lab"
	_ = src // already package memmove_lab
	_ = test
	_ = os.WriteFile(filepath.Join(dir, "m.go"), []byte(src), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "t_test.go"), []byte("package "+pkg+"\nimport \"testing\"\nfunc TestCopy(t *testing.T) {\nd:=make([]byte,8)\ns:=[]byte{1,2,3,4,5,6,7,8}\nFastlz_memmove(d,s,8)\nfor i:=range d { if d[i]!=s[i] { t.Fatal(i) } }\n}\n"), 0o644)
	cmd := exec.Command("go", "test", "-count=1", ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("test: %v\n%s\n%s", err, out, src)
	}
}
