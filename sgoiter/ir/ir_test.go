package ir_test

import (
	"os"
	"path/filepath"
	"testing"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/ir"
)

func sampleAdd() *ir.Module {
	f := ir.Func{
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
	}
	return &ir.Module{Name: "add", Funcs: []ir.Func{f}}
}

func TestIRRoundTrip(t *testing.T) {
	m := sampleAdd()
	raw, err := ir.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	m2, err := ir.Unmarshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := ir.EqualJSON(m, m2)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatalf("round-trip mismatch\n--- a ---\n%s\n--- b ---\n%s", raw, mustMarshal(m2))
	}
}

func TestIRGoldenFile(t *testing.T) {
	path := filepath.Join("..", "testdata", "ir", "add.ir.json")
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden: %v", err)
	}
	got, err := ir.Marshal(sampleAdd())
	if err != nil {
		t.Fatal(err)
	}
	// Normalize: both must unmarshal equal
	gw, err := ir.Unmarshal(want)
	if err != nil {
		t.Fatal(err)
	}
	gg, err := ir.Unmarshal(got)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := ir.EqualJSON(gw, gg)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatalf("golden drift\n--- want ---\n%s\n--- got ---\n%s", want, got)
	}
}

func mustMarshal(m *ir.Module) string {
	b, _ := ir.Marshal(m)
	return string(b)
}
