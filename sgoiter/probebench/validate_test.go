package probebench

import "testing"

func TestValidateSgoiterProbes_OK(t *testing.T) {
	lines := []ProbeLine{
		{Lib: "fnv1a_64", Stratum: "l1_1k", Backend: "sgoiter", Iters: 100, NsPerOp: 800},
		{Lib: "fnv1a_64", Stratum: "l1_1k", Backend: "c_gcc_O2", Iters: 0, NsPerOp: 0}, // C ignored
	}
	if err := ValidateSgoiterProbes(lines); err != nil {
		t.Fatal(err)
	}
}

func TestValidateSgoiterProbes_ErrorField(t *testing.T) {
	lines := []ProbeLine{
		{Lib: "fnv1a_64", Stratum: "l1_1k", Backend: "sgoiter", Error: "go build: EOF", Iters: 0, NsPerOp: 0},
	}
	if err := ValidateSgoiterProbes(lines); err == nil {
		t.Fatal("want error")
	}
}

func TestValidateSgoiterProbes_ZeroIters(t *testing.T) {
	lines := []ProbeLine{
		{Lib: "fnv1a_64", Stratum: "bulk_1m", Backend: "sgoiter", Iters: 0, NsPerOp: 1},
	}
	if err := ValidateSgoiterProbes(lines); err == nil {
		t.Fatal("want error")
	}
}

func TestValidateSgoiterProbes_ZeroNs(t *testing.T) {
	lines := []ProbeLine{
		{Lib: "fnv1a_64", Stratum: "l1_1k", Backend: "sgoiter", Iters: 10, NsPerOp: 0},
	}
	if err := ValidateSgoiterProbes(lines); err == nil {
		t.Fatal("want error")
	}
}
