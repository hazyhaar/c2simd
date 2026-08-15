package tribench

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Backend name constants.
const (
	BackendCO0     = "c_gcc_O0"
	BackendCO2     = "c_gcc_O2"
	BackendSgoiter = "sgoiter"
	BackendCcgo    = "ccgo"
)

// BackendResult is one backend run for one lib.
type BackendResult struct {
	Backend         string            `json:"backend"`
	Status          string            `json:"status"` // ok|skip|fail
	Error           string            `json:"error,omitempty"`
	CompileMS       int64             `json:"compile_ms"`
	BinaryBytes     int64             `json:"binary_bytes,omitempty"`
	RunMS           int64             `json:"run_ms"`
	StdoutSHA256    string            `json:"stdout_sha256,omitempty"`
	Stdout          string            `json:"stdout,omitempty"`
	MatchOracle     bool              `json:"match_oracle"`
	Lines           map[string]string `json:"lines,omitempty"` // fixture → digest
	BenchNSPerOp    int64             `json:"bench_ns_per_op,omitempty"`
	BenchBytesPer   int64             `json:"bench_bytes_per_op,omitempty"`
	BenchMBPerS     float64           `json:"bench_mb_per_s,omitempty"`
	AllocsPerOp     int64             `json:"allocs_per_op,omitempty"` // sgoiter only if measured
	CodeLines       int               `json:"code_lines,omitempty"`
	IdentityAssigns int               `json:"identity_assigns,omitempty"`
	VarCount        int               `json:"var_count,omitempty"`
	IntCastIndex    int               `json:"int_cast_index,omitempty"`
	RotLeftCalls    int               `json:"rot_left_calls,omitempty"`
	PprofCPU        string            `json:"pprof_cpu,omitempty"`
}

// LibReport aggregates all backends for one lib.
type LibReport struct {
	ID       string          `json:"id"`
	Kind     Kind            `json:"kind"`
	Notes    string          `json:"notes,omitempty"`
	Oracle   string          `json:"oracle_backend"`
	Backends []BackendResult `json:"backends"`
	AllMatch bool            `json:"all_match_oracle"`
	// NoOracle marks a kernel with no C reference: nothing to compare against,
	// so it is neither a match nor a failure.
	NoOracle  bool  `json:"no_oracle,omitempty"`
	BuiltOnly bool  `json:"built_not_compared,omitempty"`
	WallMS    int64 `json:"wall_ms"`
}

func sgBuilt(bs []BackendResult) bool {
	for _, b := range bs {
		if b.Backend == BackendSgoiter && b.Status == "ok" {
			return true
		}
	}
	return false
}

// Report is the full tribench run.
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

// Summary counts.
type Summary struct {
	LibsTotal    int `json:"libs_total"`
	LibsAllMatch int `json:"libs_all_match"`
	BackendOK    int `json:"backend_ok"`
	BackendFail  int `json:"backend_fail"`
	BackendSkip  int `json:"backend_skip"`
	SgoiterMatch int `json:"sgoiter_match_oracle"`
	CcgoMatch    int `json:"ccgo_match_oracle"`
	// LibsCompared is the honest denominator: kernels that had a C oracle to
	// compare against. LibsNoOracle were built but never compared.
	LibsCompared int `json:"libs_compared"`
	LibsNoOracle int `json:"libs_no_oracle"`
}

// Options configures a run.
type Options struct {
	C2simdRoot string
	OutDir     string
	SgoiterBin string
	CcgoBin    string
	Libs       []Lib    // override default catalog
	Only       []string // filter lib IDs
	SkipCcgo   bool
	SkipBench  bool
	KeepWork   bool
	Pprof      bool
	Heavy      bool
	Verbose    bool
}

// Run executes the full tribench.
func Run(opt Options) (*Report, error) {
	if opt.C2simdRoot == "" {
		return nil, fmt.Errorf("C2simdRoot required")
	}
	if opt.OutDir == "" {
		opt.OutDir = filepath.Join(opt.C2simdRoot, "spec/dogfood/testdata/cycles", "tribench_"+time.Now().Format("20060102_150405"))
	}
	if opt.SgoiterBin == "" {
		opt.SgoiterBin = filepath.Join(opt.C2simdRoot, "bin/sgoiter")
	}
	if opt.CcgoBin == "" {
		if p, err := exec.LookPath("ccgo"); err == nil {
			opt.CcgoBin = p
		}
	}
	if err := os.MkdirAll(opt.OutDir, 0o755); err != nil {
		return nil, err
	}

	rep := &Report{
		Stamp:     time.Now().UTC().Format(time.RFC3339),
		Host:      fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		GoVersion: runtime.Version(),
		Sgoiter:   opt.SgoiterBin,
		CcgoBin:   opt.CcgoBin,
		OutDir:    opt.OutDir,
	}

	libs := opt.Libs
	if len(libs) == 0 {
		libs = DefaultLibs(opt.C2simdRoot)
	}
	if len(opt.Only) > 0 {
		allow := map[string]bool{}
		for _, id := range opt.Only {
			allow[id] = true
		}
		var filtered []Lib
		for _, l := range libs {
			if allow[l.ID] {
				filtered = append(filtered, l)
			}
		}
		libs = filtered
	}

	for _, lib := range libs {
		lr := runLib(opt, lib)
		rep.Libs = append(rep.Libs, lr)
		if opt.Verbose {
			fmt.Printf("[%s] match=%v backends=%d\n", lib.ID, lr.AllMatch, len(lr.Backends))
		}
	}

	// summary
	rep.Summary.LibsTotal = len(rep.Libs)
	for _, lr := range rep.Libs {
		if lr.NoOracle {
			rep.Summary.LibsNoOracle++
		} else {
			rep.Summary.LibsCompared++
		}
		if lr.AllMatch {
			rep.Summary.LibsAllMatch++
		}
		for _, b := range lr.Backends {
			switch b.Status {
			case "ok":
				rep.Summary.BackendOK++
			case "fail":
				rep.Summary.BackendFail++
			default:
				rep.Summary.BackendSkip++
			}
			if b.Backend == BackendSgoiter && b.MatchOracle {
				rep.Summary.SgoiterMatch++
			}
			if b.Backend == BackendCcgo && b.MatchOracle {
				rep.Summary.CcgoMatch++
			}
		}
	}

	raw, _ := json.MarshalIndent(rep, "", "  ")
	_ = os.WriteFile(filepath.Join(opt.OutDir, "report.json"), raw, 0o644)
	_ = os.WriteFile(filepath.Join(opt.OutDir, "SUMMARY.md"), []byte(FormatSummaryMD(rep)), 0o644)
	return rep, nil
}

func runLib(opt Options, lib Lib) LibReport {
	t0 := time.Now()
	lr := LibReport{ID: lib.ID, Kind: lib.Kind, Notes: lib.Notes, Oracle: BackendCO2}
	work := filepath.Join(opt.OutDir, lib.ID)
	_ = os.MkdirAll(work, 0o755)

	// copy kernel + optional stub headers
	kdst := filepath.Join(work, "kernel.c")
	if err := copyFile(lib.CRel, kdst); err != nil {
		lr.Backends = append(lr.Backends, BackendResult{Backend: "setup", Status: "fail", Error: err.Error()})
		lr.WallMS = time.Since(t0).Milliseconds()
		return lr
	}
	for _, h := range lib.StubHeaders {
		// empty stub next to kernel so foldLocalIncludes finds a file (no err_include)
		_ = os.WriteFile(filepath.Join(work, h), []byte("/* tribench stub */\n"), 0o644)
	}
	_ = os.WriteFile(filepath.Join(work, "harness.c"), []byte(GenHarnessC(lib)), 0o644)

	// C O0 / O2
	var oracle BackendResult
	if lib.SkipC {
		lr.Backends = append(lr.Backends,
			BackendResult{Backend: BackendCO0, Status: "skip", Error: "SkipC"},
			BackendResult{Backend: BackendCO2, Status: "skip", Error: "SkipC"},
		)
	} else {
		cO0 := runCBackend(work, BackendCO0, "-O0", opt.SkipBench)
		cO2 := runCBackend(work, BackendCO2, "-O2", opt.SkipBench)
		lr.Backends = append(lr.Backends, cO0, cO2)
		oracle = cO2
		if oracle.Status != "ok" {
			oracle = cO0
			lr.Oracle = BackendCO0
		}
	}

	// sgoiter
	sg := runSgoiterBackend(opt, work, lib, oracle)
	if lib.SkipC {
		// No C reference to compare against. Declaring a self-match here counted
		// the kernel as bit-exact against an oracle that was its own output.
		lr.Oracle = ""
		lr.NoOracle = true
		sg.MatchOracle = false
	}
	lr.Backends = append(lr.Backends, sg)

	// ccgo
	if lib.SkipC {
		lr.Backends = append(lr.Backends, BackendResult{Backend: BackendCcgo, Status: "skip", Error: "SkipC multi-file"})
	} else if !opt.SkipCcgo && opt.CcgoBin != "" {
		// copy stubs into ccgo dir inside runner
		cg := runCcgoBackend(opt, work, lib, oracle)
		lr.Backends = append(lr.Backends, cg)
	} else {
		lr.Backends = append(lr.Backends, BackendResult{Backend: BackendCcgo, Status: "skip", Error: "ccgo disabled or missing"})
	}

	// all_match = oracle OK + sgoiter OK+match (+ ccgo match if ran ok).
	// A kernel with no external oracle is never all_match: it is reported as
	// built-but-uncompared, and counted apart in the summary.
	// c_O0 may diverge from O2 — not required
	sgOK, oracleOK, ccgoBad := false, false, false
	for _, b := range lr.Backends {
		if b.Backend == lr.Oracle && b.Status == "ok" {
			oracleOK = true
		}
		if b.Backend == BackendSgoiter && b.Status == "ok" && b.MatchOracle {
			sgOK = true
		}
		if b.Backend == BackendCcgo && b.Status == "ok" && !b.MatchOracle {
			ccgoBad = true
		}
		if b.Backend == BackendSgoiter && b.Status == "fail" {
			sgOK = false
		}
	}
	lr.AllMatch = oracleOK && sgOK && !ccgoBad && !lr.NoOracle
	if lr.NoOracle {
		lr.BuiltOnly = sgBuilt(lr.Backends)
	}

	lr.WallMS = time.Since(t0).Milliseconds()
	raw, _ := json.MarshalIndent(lr, "", "  ")
	_ = os.WriteFile(filepath.Join(work, "lib_report.json"), raw, 0o644)
	return lr
}

func runCBackend(work, name, gccOpt string, skipBench bool) BackendResult {
	br := BackendResult{Backend: name}
	bin := filepath.Join(work, name+".bin")
	t0 := time.Now()
	cmd := exec.Command("gcc", gccOpt, "-std=c11", "-Wall", filepath.Join(work, "harness.c"), filepath.Join(work, "kernel.c"), "-o", bin)
	var errb bytes.Buffer
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		br.Status = "fail"
		br.Error = errb.String()
		br.CompileMS = time.Since(t0).Milliseconds()
		return br
	}
	br.CompileMS = time.Since(t0).Milliseconds()
	if st, err := os.Stat(bin); err == nil {
		br.BinaryBytes = st.Size()
	}
	if !skipBench {
		br = attachBench(br, bin, 20)
	}
	return runBin(br, bin, work)
}

func runSgoiterBackend(opt Options, work string, lib Lib, oracle BackendResult) BackendResult {
	br := BackendResult{Backend: BackendSgoiter}
	sgodir := filepath.Join(work, "sgoiter")
	kdir := filepath.Join(sgodir, "kernel")
	_ = os.MkdirAll(kdir, 0o755)
	kgo := filepath.Join(kdir, "kernel.go")
	t0 := time.Now()
	cmd := exec.Command(opt.SgoiterBin, "-in", filepath.Join(work, "kernel.c"), "-out", kgo, "-mode", "kernel")
	var errb bytes.Buffer
	cmd.Stderr = &errb
	cmd.Stdout = &errb
	if err := cmd.Run(); err != nil {
		br.Status = "fail"
		br.Error = "sgoiter: " + errb.String()
		br.CompileMS = time.Since(t0).Milliseconds()
		return br
	}
	raw, err := os.ReadFile(kgo)
	if err != nil {
		br.Status = "fail"
		br.Error = err.Error()
		return br
	}
	kg := rewritePackage(string(raw), "kernel")
	_ = os.WriteFile(kgo, []byte(kg), 0o644)
	br.CodeLines = strings.Count(kg, "\n") + 1
	br.IdentityAssigns = countIdentityAssigns(kg)
	br.VarCount = countVarDecls(kg)
	br.IntCastIndex = strings.Count(kg, "int(v")
	br.RotLeftCalls = strings.Count(kg, "bits.RotateLeft")
	_ = os.WriteFile(filepath.Join(sgodir, "main.go"), []byte(GenHarnessSgoMain(lib)), 0o644)
	_ = os.WriteFile(filepath.Join(sgodir, "go.mod"), []byte("module trib\n\ngo 1.22\n"), 0o644)

	bin := filepath.Join(sgodir, "run.bin")
	t1 := time.Now()
	bcmd := exec.Command("go", "build", "-o", bin, ".")
	bcmd.Dir = sgodir
	bcmd.Env = append(os.Environ(), "GOWORK=off", "CGO_ENABLED=0")
	var berr bytes.Buffer
	bcmd.Stderr = &berr
	if err := bcmd.Run(); err != nil {
		br.Status = "fail"
		br.Error = "go build: " + berr.String()
		br.CompileMS = time.Since(t0).Milliseconds()
		_ = os.WriteFile(filepath.Join(sgodir, "build.err"), berr.Bytes(), 0o644)
		return br
	}
	br.CompileMS = time.Since(t0).Milliseconds()
	_ = t1
	if st, err := os.Stat(bin); err == nil {
		br.BinaryBytes = st.Size()
	}

	// optional bench on largest fixture via repeated process runs is coarse;
	// measure in-process would need test binary — wall of single run is enough + optional loop binary
	if !opt.SkipBench {
		br = attachBench(br, bin, 20)
	}
	if opt.Pprof {
		// leave hook: user can re-run with go test -cpuprofile
		br.PprofCPU = filepath.Join(sgodir, "cpu.pprof")
	}
	br = runBin(br, bin, sgodir)
	br.MatchOracle = oracle.Status == "ok" && br.Status == "ok" && br.StdoutSHA256 == oracle.StdoutSHA256
	return br
}

func runCcgoBackend(opt Options, work string, lib Lib, oracle BackendResult) BackendResult {
	br := BackendResult{Backend: BackendCcgo}
	dir := filepath.Join(work, "ccgo")
	_ = os.MkdirAll(dir, 0o755)
	// ccgo wants cwd with go.mod + libc (+ go.sum before ccgo runs)
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte(`module tribccgo

go 1.22

require modernc.org/libc v1.74.4
`), 0o644)
	// seed import so go mod tidy materializes go.sum (empty module pulls nothing)
	_ = os.WriteFile(filepath.Join(dir, "seed_libc.go"), []byte("package main\n\nimport _ \"modernc.org/libc\"\n"), 0o644)
	get := exec.Command("go", "get", "modernc.org/libc@v1.74.4")
	get.Dir = dir
	get.Env = append(os.Environ(), "GOWORK=off")
	_ = get.Run()
	tidy0 := exec.Command("go", "mod", "tidy")
	tidy0.Dir = dir
	tidy0.Env = append(os.Environ(), "GOWORK=off")
	_ = tidy0.Run()
	// copy kernel + stubs
	ksrc := filepath.Join(dir, "kernel.c")
	_ = copyFile(filepath.Join(work, "kernel.c"), ksrc)
	for _, h := range lib.StubHeaders {
		_ = copyFile(filepath.Join(work, h), filepath.Join(dir, h))
	}
	t0 := time.Now()
	cmd := exec.Command(opt.CcgoBin, "kernel.c")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	var errb bytes.Buffer
	cmd.Stderr = &errb
	cmd.Stdout = &errb
	if err := cmd.Run(); err != nil {
		br.Status = "fail"
		br.Error = "ccgo: " + errb.String()
		br.CompileMS = time.Since(t0).Milliseconds()
		return br
	}
	_ = os.Remove(filepath.Join(dir, "seed_libc.go")) // harness main.go is the real entry
	// ccgo writes kernel.go package main — rename generated, add harness
	// find .go produced
	entries, _ := os.ReadDir(dir)
	var gen string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".go") && e.Name() != "main.go" {
			gen = filepath.Join(dir, e.Name())
			break
		}
	}
	if gen == "" {
		br.Status = "fail"
		br.Error = "ccgo produced no .go"
		return br
	}
	graw, _ := os.ReadFile(gen)
	br.CodeLines = strings.Count(string(graw), "\n") + 1
	// ensure package main + build tag kept; append is separate main.go — CONFLICT package main two files OK
	// but both package main - harness main.go + kernel.go both main: OK if one has main()
	// strip any func main from generated if present
	_ = os.WriteFile(filepath.Join(dir, "main.go"), []byte(GenHarnessCcgoMain(lib)), 0o644)
	// go mod tidy
	tcmd := exec.Command("go", "mod", "tidy")
	tcmd.Dir = dir
	tcmd.Env = append(os.Environ(), "GOWORK=off")
	_ = tcmd.Run()

	bin := filepath.Join(dir, "run.bin")
	bcmd := exec.Command("go", "build", "-o", bin, ".")
	bcmd.Dir = dir
	bcmd.Env = append(os.Environ(), "GOWORK=off", "CGO_ENABLED=0")
	var berr bytes.Buffer
	bcmd.Stderr = &berr
	if err := bcmd.Run(); err != nil {
		br.Status = "fail"
		br.Error = "go build: " + berr.String()
		br.CompileMS = time.Since(t0).Milliseconds()
		_ = os.WriteFile(filepath.Join(dir, "build.err"), berr.Bytes(), 0o644)
		return br
	}
	br.CompileMS = time.Since(t0).Milliseconds()
	if st, err := os.Stat(bin); err == nil {
		br.BinaryBytes = st.Size()
	}
	if !opt.SkipBench {
		br = attachBench(br, bin, 10)
	}
	br = runBin(br, bin, dir)
	br.MatchOracle = oracle.Status == "ok" && br.Status == "ok" && br.StdoutSHA256 == oracle.StdoutSHA256
	return br
}

func runBin(br BackendResult, bin, dir string) BackendResult {
	t0 := time.Now()
	cmd := exec.Command(bin)
	cmd.Dir = dir
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		br.Status = "fail"
		br.Error = strings.TrimSpace(errb.String() + " " + err.Error())
		br.RunMS = time.Since(t0).Milliseconds()
		return br
	}
	br.RunMS = time.Since(t0).Milliseconds()
	br.Status = "ok"
	br.Stdout = out.String()
	sum := sha256.Sum256(out.Bytes())
	br.StdoutSHA256 = hex.EncodeToString(sum[:])
	br.Lines = parseLines(out.String())
	_ = os.WriteFile(filepath.Join(dir, br.Backend+"_stdout.txt"), out.Bytes(), 0o644)
	return br
}

func parseLines(s string) map[string]string {
	m := map[string]string{}
	for _, ln := range strings.Split(s, "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		parts := strings.SplitN(ln, " ", 2)
		if len(parts) == 2 {
			m[parts[0]] = parts[1]
		}
	}
	return m
}

func attachBench(br BackendResult, bin string, rounds int) BackendResult {
	if rounds < 3 {
		rounds = 3
	}
	// warm
	_ = exec.Command(bin).Run()
	t0 := time.Now()
	for i := 0; i < rounds; i++ {
		if err := exec.Command(bin).Run(); err != nil {
			return br
		}
	}
	elapsed := time.Since(t0)
	br.BenchNSPerOp = elapsed.Nanoseconds() / int64(rounds)
	return br
}

func copyFile(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0o644)
}

func countIdentityAssigns(src string) int {
	n := 0
	for _, ln := range strings.Split(src, "\n") {
		trim := strings.TrimSpace(ln)
		// v12 = v34
		if len(trim) < 5 || trim[0] != 'v' {
			continue
		}
		eq := strings.Index(trim, " = ")
		if eq <= 0 {
			continue
		}
		lhs := strings.TrimSpace(trim[:eq])
		rhs := strings.TrimSpace(trim[eq+3:])
		if isVIdent(lhs) && isVIdent(rhs) {
			n++
		}
	}
	return n
}

func countVarDecls(src string) int {
	n := 0
	for _, ln := range strings.Split(src, "\n") {
		trim := strings.TrimSpace(ln)
		if strings.HasPrefix(trim, "var v") {
			n++
		}
	}
	return n
}

func isVIdent(s string) bool {
	if len(s) < 2 || s[0] != 'v' {
		return false
	}
	for i := 1; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// FormatSummaryMD renders a markdown table.
func FormatSummaryMD(rep *Report) string {
	var b strings.Builder
	b.WriteString("# tribench report\n\n")
	fmt.Fprintf(&b, "- stamp: %s\n- host: %s\n- go: %s\n", rep.Stamp, rep.Host, rep.GoVersion)
	fmt.Fprintf(&b, "- summary: %d/%d compared bit-exact vs C; sgoiter-match=%d ccgo-match=%d\n",
		rep.Summary.LibsAllMatch, rep.Summary.LibsCompared, rep.Summary.SgoiterMatch, rep.Summary.CcgoMatch)
	if rep.Summary.LibsNoOracle > 0 {
		fmt.Fprintf(&b, "- %d kernel(s) built with no C oracle: compiled and run, never compared\n",
			rep.Summary.LibsNoOracle)
	}
	b.WriteString("\n| lib | c_O2 | sgoiter | ccgo | sgo match | ccgo match | sgo lines |\n")
	b.WriteString("|-----|------|---------|------|-----------|------------|-----------|\n")
	for _, lr := range rep.Libs {
		cell := map[string]BackendResult{}
		for _, x := range lr.Backends {
			cell[x.Backend] = x
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %v | %d |\n",
			lr.ID,
			statusShort(cell[BackendCO2]),
			statusShort(cell[BackendSgoiter]),
			statusShort(cell[BackendCcgo]),
			matchCell(lr, cell[BackendSgoiter]),
			cell[BackendCcgo].MatchOracle,
			cell[BackendSgoiter].CodeLines,
		)
	}
	return b.String()
}

// matchCell never prints false for a kernel that had nothing to compare against.
func matchCell(lr LibReport, sg BackendResult) string {
	if lr.NoOracle {
		return "no oracle"
	}
	return fmt.Sprintf("%v", sg.MatchOracle)
}

func statusShort(b BackendResult) string {
	if b.Status == "" {
		return "—"
	}
	if b.Status == "ok" {
		return "OK"
	}
	if b.Status == "skip" {
		return "skip"
	}
	return "FAIL"
}
