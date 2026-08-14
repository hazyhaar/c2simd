package sgoiterbench

import (
	"strings"
	"testing"
)

// The report must not invent figures. An earlier version printed "100%",
// a hardcoded "1.05x - 1.25x" speed ratio and "<1ms (native)" for values that
// had never been measured.
func TestSummaryReportsOnlyMeasuredValues(t *testing.T) {
	rep := &Report{
		Stamp:     "2026-08-11T00:00:00Z",
		Host:      "linux/amd64",
		GoVersion: "go1.26.5",
		Libs: []LibReport{{
			ID:   "fnv1a_64",
			Kind: KindHash64,
			Backends: []BackendResult{
				{Backend: BackendCO2, Status: "ok", MatchOracle: true},
				{Backend: BackendSgoiter, Status: "ok", MatchOracle: true, CodeLines: 17},
				{Backend: BackendCcgo, Status: "skip"},
			},
		}},
		Summary: Summary{LibsTotal: 1, SgoiterMatch: 1},
	}
	out := FormatSummaryMD(rep)

	for _, banned := range []string{"100%", "1.05x", "1.25x", "Zero-CGO", "native", "Idiomatic", "~1.0x"} {
		if strings.Contains(out, banned) {
			t.Errorf("report contains the unmeasured claim %q", banned)
		}
	}
	// throughput was never recorded here, so it must say so
	if !strings.Contains(out, "not measured") {
		t.Error("unmeasured throughput must be reported as such")
	}
	if !strings.Contains(out, "1 of 1 kernels compared") {
		t.Errorf("parity line missing its denominator:\n%s", out)
	}
}

func TestThroughputCellNeedsAMeasurement(t *testing.T) {
	if got := throughputCell(nil); got != "not measured" {
		t.Errorf("nil backend: got %q", got)
	}
	if got := throughputCell(&BackendResult{}); got != "not measured" {
		t.Errorf("zero throughput: got %q", got)
	}
	if got := throughputCell(&BackendResult{ThroughputMBs: 812.5}); got != "812.5 MB/s" {
		t.Errorf("measured throughput: got %q", got)
	}
}
