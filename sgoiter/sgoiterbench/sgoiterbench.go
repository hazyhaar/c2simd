package sgoiterbench

import (
	"fmt"
	"strings"
)

// Constants for backends
const (
	BackendCO0     = "c_gcc_O0"
	BackendCO2     = "c_gcc_O2"
	BackendSgoiter = "sgoiter"
	BackendCcgo    = "ccgo"
)

// Result for a single backend run on a single kernel. Fields that no code fills
// are absent by design: an always-zero column reads as a measurement of zero.
type BackendResult struct {
	Backend      string            `json:"backend"`
	Status       string            `json:"status"` // ok | skip | fail
	Error        string            `json:"error,omitempty"`
	CompileMS    int64             `json:"compile_ms"`
	BinaryBytes  int64             `json:"binary_bytes,omitempty"`
	StdoutSHA256 string            `json:"stdout_sha256,omitempty"`
	Stdout       string            `json:"stdout,omitempty"`
	MatchOracle  bool              `json:"match_oracle"`
	Lines        map[string]string `json:"lines,omitempty"` // fixture -> digest
	// NsPerOp and ThroughputMBs are self-reported by the harness loop, never
	// derived from process wall time.
	NsPerOp         float64 `json:"ns_per_op,omitempty"`
	ThroughputMBs   float64 `json:"throughput_mb_s,omitempty"`
	CodeLines       int     `json:"code_lines,omitempty"`
	IdentityAssigns int     `json:"identity_assigns,omitempty"`
}

// LibReport aggregates results for a single kernel.
type LibReport struct {
	ID       string          `json:"id"`
	Kind     Kind            `json:"kind"`
	Notes    string          `json:"notes,omitempty"`
	Oracle   string          `json:"oracle_backend"`
	Backends []BackendResult `json:"backends"`
	AllMatch bool            `json:"all_match_oracle"`
	WallMS   int64           `json:"wall_ms"`
}

// Report captures the full sgoiter-bench run.
type Report struct {
	Stamp     string      `json:"stamp"`
	Host      string      `json:"host"`
	GoVersion string      `json:"go_version"`
	Sgoiter   string      `json:"sgoiter_bin"`
	CcgoBin   string      `json:"ccgo_bin,omitempty"`
	OutDir    string      `json:"out_dir"`
	Libs      []LibReport `json:"libs"`
	Summary   Summary     `json:"summary"`
}

// Summary statistics across all tested libraries.
type Summary struct {
	LibsTotal    int `json:"libs_total"`
	LibsAllMatch int `json:"libs_all_match"`
	SgoiterMatch int `json:"sgoiter_match_oracle"`
	CcgoMatch    int `json:"ccgo_match_oracle"`
}

// Options configures sgoiter-bench.
type Options struct {
	C2simdRoot string
	OutDir     string
	SgoiterBin string
	CcgoBin    string
	Only       []string
	SkipCcgo   bool
	SkipBench  bool
	Heavy      bool // Enable 1M, 10M, 100M heavy load
	Verbose    bool
}

// FormatSummaryMD renders the run. Every column here is filled from a recorded
// measurement; a field that was never measured prints "not measured" rather than
// a plausible-looking value.
func FormatSummaryMD(rep *Report) string {
	var sb strings.Builder
	sb.WriteString("# sgoiter-bench report\n\n")
	fmt.Fprintf(&sb, "- stamp: %s\n", rep.Stamp)
	fmt.Fprintf(&sb, "- host: %s\n", rep.Host)
	fmt.Fprintf(&sb, "- go: %s\n", rep.GoVersion)
	fmt.Fprintf(&sb, "- bit-exact vs C gcc -O2: %d of %d kernels compared\n\n",
		rep.Summary.SgoiterMatch, rep.Summary.LibsTotal)

	sb.WriteString("## Parity and code size\n\n")
	sb.WriteString("| kernel | sgoiter | ccgo | sgoiter lines | ccgo lines | sgoiter compile | ccgo compile |\n")
	sb.WriteString("| :--- | :---: | :---: | ---: | ---: | ---: | ---: |\n")

	for _, lib := range rep.Libs {
		sgoRes, ccgoRes := backendOf(lib, BackendSgoiter), backendOf(lib, BackendCcgo)
		fmt.Fprintf(&sb, "| `%s` | %s | %s | %s | %s | %s | %s |\n",
			lib.ID,
			parityCell(sgoRes),
			parityCell(ccgoRes),
			intCell(sgoRes, func(b *BackendResult) int64 { return int64(b.CodeLines) }),
			intCell(ccgoRes, func(b *BackendResult) int64 { return int64(b.CodeLines) }),
			msCell(sgoRes),
			msCell(ccgoRes),
		)
	}

	sb.WriteString("\n## Throughput\n\n")
	sb.WriteString("Figures come from the harness timing its own loop. Process wall time is\n")
	sb.WriteString("not reported: it measures fork, load and exit, not the kernel.\n\n")
	sb.WriteString("| kernel | C gcc -O2 | sgoiter | ccgo |\n")
	sb.WriteString("| :--- | ---: | ---: | ---: |\n")
	for _, lib := range rep.Libs {
		fmt.Fprintf(&sb, "| `%s` | %s | %s | %s |\n",
			lib.ID,
			throughputCell(backendOf(lib, BackendCO2)),
			throughputCell(backendOf(lib, BackendSgoiter)),
			throughputCell(backendOf(lib, BackendCcgo)),
		)
	}

	return sb.String()
}

func backendOf(lib LibReport, name string) *BackendResult {
	for i := range lib.Backends {
		if lib.Backends[i].Backend == name {
			return &lib.Backends[i]
		}
	}
	return nil
}

func parityCell(b *BackendResult) string {
	switch {
	case b == nil, b.Status == "skip", b.Status == "":
		return "not run"
	case b.Status == "fail":
		return "build failed"
	case b.MatchOracle:
		return "match"
	default:
		return "differs"
	}
}

func intCell(b *BackendResult, get func(*BackendResult) int64) string {
	if b == nil {
		return "—"
	}
	if v := get(b); v > 0 {
		return fmt.Sprintf("%d", v)
	}
	return "—"
}

func msCell(b *BackendResult) string {
	if b == nil || b.CompileMS <= 0 {
		return "—"
	}
	return fmt.Sprintf("%d ms", b.CompileMS)
}

func throughputCell(b *BackendResult) string {
	if b == nil || b.ThroughputMBs <= 0 {
		return "not measured"
	}
	return fmt.Sprintf("%.1f MB/s", b.ThroughputMBs)
}
