package sgoiterbench

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// PerfMetric holds what the harness binary reported about its own loop. Nothing
// here is derived from the wall time of the process: forking, loading and exiting
// dwarf the kernel on these payload sizes.
type PerfMetric struct {
	LibID         string  `json:"lib_id"`
	Backend       string  `json:"backend"`
	Iterations    int     `json:"iterations"`
	ThroughputMBs float64 `json:"throughput_mb_s"`
	NsPerOp       float64 `json:"ns_per_op"`
}

// benchLinePat parses the harness self-report: "BENCH: 12.34 ns/op | 567.89 MB/s".
var benchLinePat = regexp.MustCompile(`BENCH:\s*([0-9.]+)\s*ns/op\s*\|\s*([0-9.]+)\s*MB/s`)

// RunBenchmarkMeasurement runs the harness in bench mode and reads the numbers it
// prints. A binary that reports nothing yields an error rather than a figure
// reconstructed from an assumed iteration count.
func RunBenchmarkMeasurement(binPath string, backendName string, libID string) (PerfMetric, error) {
	cmd := exec.Command(binPath, "--bench")
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb

	if err := cmd.Run(); err != nil {
		return PerfMetric{LibID: libID, Backend: backendName}, fmt.Errorf("%s --bench: %w: %s", binPath, err, strings.TrimSpace(errb.String()))
	}

	sub := benchLinePat.FindStringSubmatch(out.String())
	if sub == nil {
		return PerfMetric{LibID: libID, Backend: backendName},
			fmt.Errorf("%s --bench printed no BENCH line: %q", binPath, strings.TrimSpace(out.String()))
	}
	ns, err := strconv.ParseFloat(sub[1], 64)
	if err != nil {
		return PerfMetric{LibID: libID, Backend: backendName}, err
	}
	mbs, err := strconv.ParseFloat(sub[2], 64)
	if err != nil {
		return PerfMetric{LibID: libID, Backend: backendName}, err
	}
	return PerfMetric{
		LibID:         libID,
		Backend:       backendName,
		Iterations:    benchIterations,
		ThroughputMBs: mbs,
		NsPerOp:       ns,
	}, nil
}
