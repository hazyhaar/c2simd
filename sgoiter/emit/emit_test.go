package emit_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/emit"
	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/ir"
)

func TestEmitAddCompiles(t *testing.T) {
	m := &ir.Module{
		Name: "addpkg",
		Funcs: []ir.Func{{
			Name:   "add",
			Result: ir.TypInt,
			Params: []ir.Param{
				{Name: "a", Type: ir.TypInt, Reg: 0},
				{Name: "b", Type: ir.TypInt, Reg: 1},
			},
			NVals: 3,
			Body: []ir.Instr{
				{Op: ir.OpAdd, Dst: 2, Args: []ir.Value{0, 1}},
				{Op: ir.OpReturn, Args: []ir.Value{2}},
			},
		}},
	}
	src, err := emit.Emit(m, emit.ProfileGo127)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "add.go")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	// go.mod for isolated build
	mod := "module tmpadd\n\ngo 1.26\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "build", "-o", filepath.Join(dir, "out.a"), ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build: %v\n%s\n--- src ---\n%s", err, out, src)
	}
}
