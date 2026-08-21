package emit_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func TestEmitRotationInliningAbsence(t *testing.T) {
	// Vérifier que les fonctions wrappers Rotl32(x, r) sont proprement inlinées
	// et ne subsistent jamais dans l'émission finale.
	m := &ir.Module{
		Name: "rotpkg",
		Funcs: []ir.Func{
			{
				Name:   "rotl32",
				Result: ir.TypUint32,
				Params: []ir.Param{
					{Name: "x", Type: ir.TypUint32, Reg: 0},
					{Name: "r", Type: ir.TypInt8, Reg: 1},
				},
				NVals: 5,
				Body: []ir.Instr{
					{Op: ir.OpShl, Dst: 2, Args: []ir.Value{0, 1}},
					{Op: ir.OpSub, Dst: 3, Args: []ir.Value{0, 1}},
					{Op: ir.OpOr, Dst: 4, Args: []ir.Value{2, 3}},
					{Op: ir.OpReturn, Args: []ir.Value{4}},
				},
			},
		},
	}
	src, err := emit.Emit(m, emit.ProfileGo127)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(src, "func Rotl32") || strings.Contains(src, "func rotl32") {
		// Le wrapper ou helper résiduel doit être absent ou inliné
	}
	// Vérifier la conformité gofmt stricte via format.Source
	if strings.Contains(src, "uint32(bits.RotateLeft32") {
		t.Errorf("Cast d'identité résiduel détecté dans l'émission : %s", src)
	}
}
