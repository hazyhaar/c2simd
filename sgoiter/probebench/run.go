package probebench

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"code.hazyhaar.fr/devhoros/c2simd/sgoiter/tribench"
)

// Options for a probe run.
type Options struct {
	C2simdRoot string
	WorkDir    string // must be on a declared disk; default DefaultWorkDir()
	SgoiterBin string
	CcgoBin    string
	Only       []string
	SkipCcgo   bool
	SkipIO     bool
	OutDir     string
	// Repeat > 1: run each backend stratum N times and keep median ns_per_op.
	Repeat int
}

// ProbeLine is one JSON PROBE{} measurement.
type ProbeLine struct {
	Lib          string  `json:"lib"`
	Stratum      string  `json:"stratum"`
	Phase        string  `json:"phase"`
	Backend      string  `json:"backend"`
	PayloadBytes int     `json:"payload_bytes"`
	Iters        int     `json:"iters"`
	NsPerOp      float64 `json:"ns_per_op"`
	MBs          float64 `json:"mb_s"`
	Allocs       int64   `json:"allocs"`
	AllocBytes   int64   `json:"alloc_bytes"`
	HeapInuse    uint64  `json:"heap_inuse,omitempty"`
	MaxRSSKiB    int64   `json:"max_rss_kib,omitempty"`
	Minflt       int64   `json:"minflt,omitempty"`
	Majflt       int64   `json:"majflt,omitempty"`
	Sink         uint64  `json:"sink,omitempty"`
	// disk context
	WorkDisk string `json:"work_disk,omitempty"`
	Error    string `json:"error,omitempty"`
}

// Report is the full probebench output.
type Report struct {
	Stamp      string           `json:"stamp"`
	OutDir     string           `json:"out_dir"`
	HostDisks  []DiskEnv        `json:"host_disks"`
	WorkDir    string           `json:"work_dir"`
	WorkDisk   DiskEnv          `json:"work_disk"`
	IO         []IOResult       `json:"io_probes,omitempty"`
	Probes     []ProbeLine      `json:"probes"`
	Bottleneck []BottleneckNote `json:"bottlenecks,omitempty"`
}

// IOResult disk sequential probe.
type IOResult struct {
	ID       string  `json:"id"`
	Path     string  `json:"path"`
	Disk     DiskEnv `json:"disk"`
	WriteMBs float64 `json:"write_mb_s"`
	ReadMBs  float64 `json:"read_mb_s"`
	Error    string  `json:"error,omitempty"`
}

// BottleneckNote heuristic goulet.
type BottleneckNote struct {
	Lib     string `json:"lib"`
	Stratum string `json:"stratum"`
	Backend string `json:"backend"`
	Kind    string `json:"kind"` // cpu|alloc|rss|io_skew
	Detail  string `json:"detail"`
}

// Run executes stratified probes.
func Run(opt Options) (*Report, error) {
	if opt.WorkDir == "" {
		opt.WorkDir = DefaultWorkDir()
	}
	if opt.OutDir == "" {
		opt.OutDir = filepath.Join(opt.C2simdRoot, "sgoiter/spec/bench/probe_"+time.Now().Format("20060102_150405"))
	}
	_ = os.MkdirAll(opt.OutDir, 0o755)
	_ = os.MkdirAll(opt.WorkDir, 0o755)

	rep := &Report{
		Stamp:     time.Now().UTC().Format(time.RFC3339),
		OutDir:    opt.OutDir,
		HostDisks: HostInventory(),
		WorkDir:   opt.WorkDir,
		WorkDisk:  ResolveDiskEnv(opt.WorkDir),
	}

	// warn if workdir is rotational
	if rep.WorkDisk.Rotational != nil && *rep.WorkDisk.Rotational {
		rep.Bottleneck = append(rep.Bottleneck, BottleneckNote{
			Kind:   "io_skew",
			Detail: "workdir on rotational disk: " + FormatDiskEnv(rep.WorkDisk) + " — CPU probes will be polluted; use /tmp or NVMe",
		})
	}

	if !opt.SkipIO {
		for _, ios := range DefaultIOStrata() {
			w, r, err := RunIOProbe(ios)
			ir := IOResult{ID: ios.ID, Path: ios.Path, Disk: ResolveDiskEnv(ios.Path), WriteMBs: w, ReadMBs: r}
			if err != nil {
				ir.Error = err.Error()
			}
			rep.IO = append(rep.IO, ir)
		}
	}

	libs := tribench.DefaultLibs(opt.C2simdRoot)
	only := map[string]bool{}
	for _, o := range opt.Only {
		only[o] = true
	}

	sgo := opt.SgoiterBin
	if sgo == "" {
		sgo = filepath.Join(opt.C2simdRoot, "bin/sgoiter")
	}
	ccgo := opt.CcgoBin
	if ccgo == "" && !opt.SkipCcgo {
		if p, err := exec.LookPath("ccgo"); err == nil {
			ccgo = p
		}
	}

	for _, lib := range libs {
		if len(only) > 0 && !only[lib.ID] {
			continue
		}
		if lib.SkipC {
			continue // probe needs C symbol for C backend; sgo-only later
		}
		// Transpile ccgo once per lib (shared kernel.go across strata).
		var ccgoKernelDir string
		var ccgoPrepErr error
		if !opt.SkipCcgo && ccgo != "" {
			ccgoKernelDir, ccgoPrepErr = prepCcgoKernel(opt, ccgo, lib)
		}
		repN := opt.Repeat
		if repN < 1 {
			repN = 1
		}
		for _, st := range StrataFor(lib.Kind) {
			// --- C ---
			rep.Probes = append(rep.Probes, withDisk(medianProbe(repN, func() (ProbeLine, error) {
				return runCProbe(opt, lib, st)
			}, lib.ID, st.ID, "c_gcc_O2"), rep.WorkDisk))
			// --- sgoiter ---
			rep.Probes = append(rep.Probes, withDisk(medianProbe(repN, func() (ProbeLine, error) {
				return runSgoProbe(opt, sgo, lib, st)
			}, lib.ID, st.ID, "sgoiter"), rep.WorkDisk))
			// --- ccgo ---
			if opt.SkipCcgo {
				// omit
			} else if ccgo == "" {
				rep.Probes = append(rep.Probes, ProbeLine{Lib: lib.ID, Stratum: st.ID, Backend: "ccgo", Error: "ccgo binary not found", WorkDisk: FormatDiskEnv(rep.WorkDisk)})
			} else if ccgoPrepErr != nil {
				rep.Probes = append(rep.Probes, ProbeLine{Lib: lib.ID, Stratum: st.ID, Backend: "ccgo", Error: ccgoPrepErr.Error(), WorkDisk: FormatDiskEnv(rep.WorkDisk)})
			} else {
				kdir := ccgoKernelDir
				rep.Probes = append(rep.Probes, withDisk(medianProbe(repN, func() (ProbeLine, error) {
					return runCcgoProbe(opt, kdir, lib, st)
				}, lib.ID, st.ID, "ccgo"), rep.WorkDisk))
			}
		}
	}

	rep.Bottleneck = append(rep.Bottleneck, analyzeBottlenecks(rep.Probes)...)

	raw, _ := json.MarshalIndent(rep, "", "  ")
	_ = os.WriteFile(filepath.Join(opt.OutDir, "probe_report.json"), raw, 0o644)
	_ = os.WriteFile(filepath.Join(opt.OutDir, "PROBE.md"), []byte(FormatReportMD(rep)), 0o644)
	_ = os.WriteFile(filepath.Join(opt.OutDir, "LAYERS.md"), []byte(FormatLayersMD(rep)), 0o644)
	// Fail-closed: a report with sgoiter error/iters=0 must not be treated as success
	// (incident 74c7b0f sealed fnv build failures as ns=0 "green" metrics).
	if err := ValidateSgoiterProbes(rep.Probes); err != nil {
		return rep, err
	}
	return rep, nil
}

func withDisk(pl ProbeLine, wd DiskEnv) ProbeLine {
	pl.WorkDisk = FormatDiskEnv(wd)
	return pl
}

// medianProbe runs fn n times; on any hard error returns error line; else median ns.
func medianProbe(n int, fn func() (ProbeLine, error), lib, st, backend string) ProbeLine {
	if n < 1 {
		n = 1
	}
	var oks []ProbeLine
	var lastErr error
	for i := 0; i < n; i++ {
		pl, err := fn()
		if err != nil {
			lastErr = err
			continue
		}
		if pl.Error != "" {
			lastErr = fmt.Errorf("%s", pl.Error)
			continue
		}
		oks = append(oks, pl)
	}
	if len(oks) == 0 {
		msg := "all repeats failed"
		if lastErr != nil {
			msg = lastErr.Error()
		}
		return ProbeLine{Lib: lib, Stratum: st, Backend: backend, Error: msg}
	}
	// median by ns_per_op
	sort.Slice(oks, func(i, j int) bool { return oks[i].NsPerOp < oks[j].NsPerOp })
	return oks[len(oks)/2]
}

// ValidateSgoiterProbes refuses a probe report that is not opposable for the
// sgoiter backend: any Error field, or iters==0 / ns_per_op==0 without error
// (vacuous zeros). Call before committing probe_report.json.
func ValidateSgoiterProbes(lines []ProbeLine) error {
	return validateBackend(lines, "sgoiter")
}

// ValidateCcgoProbes same gate for ccgo rows (optional -require-ccgo).
func ValidateCcgoProbes(lines []ProbeLine) error {
	return validateBackend(lines, "ccgo")
}

func validateBackend(lines []ProbeLine, backend string) error {
	var bad []string
	n := 0
	for _, p := range lines {
		if p.Backend != backend {
			continue
		}
		n++
		tag := p.Lib + "/" + p.Stratum
		if p.Error != "" {
			bad = append(bad, fmt.Sprintf("%s: error=%q", tag, p.Error))
			continue
		}
		if p.Iters <= 0 {
			bad = append(bad, fmt.Sprintf("%s: iters=%d", tag, p.Iters))
		}
		if p.NsPerOp <= 0 {
			bad = append(bad, fmt.Sprintf("%s: ns_per_op=%g", tag, p.NsPerOp))
		}
	}
	if n == 0 {
		return fmt.Errorf("probebench: no %s rows in report", backend)
	}
	if len(bad) == 0 {
		return nil
	}
	return fmt.Errorf("probebench: %s report not commit-ready (%d fault(s)): %s", backend, len(bad), strings.Join(bad, "; "))
}

func runCProbe(opt Options, lib tribench.Lib, st Stratum) (ProbeLine, error) {
	dir := filepath.Join(opt.WorkDir, "c", lib.ID+"_"+st.ID)
	_ = os.MkdirAll(dir, 0o755)
	src := lib.CRel
	if !filepath.IsAbs(src) {
		src = filepath.Join(opt.C2simdRoot, src)
	}
	_ = copyFile(src, filepath.Join(dir, "kernel.c"))
	for _, h := range lib.StubHeaders {
		_ = os.WriteFile(filepath.Join(dir, h), []byte("/* stub */\n"), 0o644)
	}
	_ = os.WriteFile(filepath.Join(dir, "probe.c"), []byte(GenProbeC(lib, st)), 0o644)
	bin := filepath.Join(dir, "probe.bin")
	cmd := exec.Command("gcc", "-O2", "-std=c11", "-Wall", "probe.c", "kernel.c", "-o", bin)
	cmd.Dir = dir
	var errb bytes.Buffer
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return ProbeLine{}, fmt.Errorf("gcc: %s", errb.String())
	}
	return runProbeBin(bin, dir)
}

// prepCcgoKernel runs ccgo once per lib; returns dir containing kernel.go + go.mod.
func prepCcgoKernel(opt Options, ccgoBin string, lib tribench.Lib) (string, error) {
	dir := filepath.Join(opt.WorkDir, "ccgo_kernel", lib.ID)
	_ = os.RemoveAll(dir)
	_ = os.MkdirAll(dir, 0o755)
	src := lib.CRel
	if !filepath.IsAbs(src) {
		src = filepath.Join(opt.C2simdRoot, src)
	}
	if err := copyFile(src, filepath.Join(dir, "kernel.c")); err != nil {
		return "", err
	}
	for _, h := range lib.StubHeaders {
		_ = os.WriteFile(filepath.Join(dir, h), []byte("/* stub */\n"), 0o644)
	}
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte(`module tribccgo

go 1.22

require modernc.org/libc v1.74.4
`), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "seed_libc.go"), []byte("package main\n\nimport _ \"modernc.org/libc\"\n"), 0o644)
	get := exec.Command("go", "get", "modernc.org/libc@v1.74.4")
	get.Dir = dir
	get.Env = append(os.Environ(), "GOWORK=off")
	_ = get.Run()
	tidy0 := exec.Command("go", "mod", "tidy")
	tidy0.Dir = dir
	tidy0.Env = append(os.Environ(), "GOWORK=off")
	_ = tidy0.Run()

	cmd := exec.Command(ccgoBin, "kernel.c")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	var errb bytes.Buffer
	cmd.Stderr = &errb
	cmd.Stdout = &errb
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ccgo: %s", errb.String())
	}
	_ = os.Remove(filepath.Join(dir, "seed_libc.go"))
	// ensure a .go kernel exists
	found := false
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".go") {
			found = true
			break
		}
	}
	if !found {
		return "", fmt.Errorf("ccgo produced no .go")
	}
	return dir, nil
}

func runCcgoProbe(opt Options, kernelDir string, lib tribench.Lib, st Stratum) (ProbeLine, error) {
	dir := filepath.Join(opt.WorkDir, "ccgo", lib.ID+"_"+st.ID)
	_ = os.RemoveAll(dir)
	_ = os.MkdirAll(dir, 0o755)
	// copy go.mod/sum + generated kernel .go from prep dir
	entries, err := os.ReadDir(kernelDir)
	if err != nil {
		return ProbeLine{}, err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if name == "kernel.c" || strings.HasSuffix(name, ".h") {
			continue
		}
		_ = copyFile(filepath.Join(kernelDir, name), filepath.Join(dir, name))
	}
	_ = os.WriteFile(filepath.Join(dir, "main.go"), []byte(GenProbeCcgoMain(lib, st)), 0o644)
	tcmd := exec.Command("go", "mod", "tidy")
	tcmd.Dir = dir
	tcmd.Env = append(os.Environ(), "GOWORK=off")
	_ = tcmd.Run()
	bin := filepath.Join(dir, "probe.bin")
	bcmd := exec.Command("go", "build", "-o", bin, ".")
	bcmd.Dir = dir
	bcmd.Env = append(os.Environ(), "GOWORK=off", "CGO_ENABLED=0")
	var berr bytes.Buffer
	bcmd.Stderr = &berr
	if err := bcmd.Run(); err != nil {
		return ProbeLine{}, fmt.Errorf("go build: %s", berr.String())
	}
	return runProbeBin(bin, dir)
}

func runSgoProbe(opt Options, sgo string, lib tribench.Lib, st Stratum) (ProbeLine, error) {
	dir := filepath.Join(opt.WorkDir, "sgo", lib.ID+"_"+st.ID)
	kdir := filepath.Join(dir, "kernel")
	_ = os.MkdirAll(kdir, 0o755)
	src := lib.CRel
	if !filepath.IsAbs(src) {
		src = filepath.Join(opt.C2simdRoot, src)
	}
	kgo := filepath.Join(kdir, "kernel.go")
	cmd := exec.Command(sgo, "-in", src, "-out", kgo, "-mode", "kernel")
	var errb bytes.Buffer
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return ProbeLine{}, fmt.Errorf("sgoiter: %s", errb.String())
	}
	// package kernel
	raw, _ := os.ReadFile(kgo)
	kg := string(raw)
	if i := strings.Index(kg, "package "); i >= 0 {
		end := strings.IndexByte(kg[i:], '\n')
		kg = kg[:i] + "package kernel\n" + kg[i+end+1:]
	}
	_ = os.WriteFile(kgo, []byte(kg), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "main.go"), []byte(GenProbeGoMain(lib, st)), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module trib\n\ngo 1.22\n"), 0o644)
	bin := filepath.Join(dir, "probe.bin")
	bcmd := exec.Command("go", "build", "-o", bin, ".")
	bcmd.Dir = dir
	bcmd.Env = append(os.Environ(), "GOWORK=off", "CGO_ENABLED=0")
	var berr bytes.Buffer
	bcmd.Stderr = &berr
	if err := bcmd.Run(); err != nil {
		return ProbeLine{}, fmt.Errorf("go build: %s", berr.String())
	}
	return runProbeBin(bin, dir)
}

func runProbeBin(bin, dir string) (ProbeLine, error) {
	cmd := exec.Command(bin)
	cmd.Dir = dir
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return ProbeLine{}, fmt.Errorf("run: %v %s", err, errb.String())
	}
	return parseProbeLine(out.String())
}

func parseProbeLine(stdout string) (ProbeLine, error) {
	for _, ln := range strings.Split(stdout, "\n") {
		ln = strings.TrimSpace(ln)
		if !strings.HasPrefix(ln, "PROBE") {
			continue
		}
		js := strings.TrimPrefix(ln, "PROBE")
		var pl ProbeLine
		if err := json.Unmarshal([]byte(js), &pl); err != nil {
			return ProbeLine{}, fmt.Errorf("json: %w (%s)", err, js)
		}
		return pl, nil
	}
	return ProbeLine{}, fmt.Errorf("no PROBE line in %q", stdout)
}

func analyzeBottlenecks(probes []ProbeLine) []BottleneckNote {
	var notes []BottleneckNote
	// group by lib+stratum
	type key struct{ lib, st string }
	by := map[key][]ProbeLine{}
	for _, p := range probes {
		if p.Error != "" {
			continue
		}
		by[key{p.Lib, p.Stratum}] = append(by[key{p.Lib, p.Stratum}], p)
	}
	for k, ps := range by {
		var c, s *ProbeLine
		for i := range ps {
			if ps[i].Backend == "c_gcc_O2" {
				c = &ps[i]
			}
			if ps[i].Backend == "sgoiter" {
				s = &ps[i]
			}
		}
		if c == nil || s == nil || c.NsPerOp <= 0 {
			continue
		}
		ratio := s.NsPerOp / c.NsPerOp
		if ratio >= 1.8 && s.Phase != "overhead" {
			sev := "modéré"
			if ratio >= 3.5 {
				sev = "fort"
			} else if ratio >= 2.5 {
				sev = "net"
			}
			notes = append(notes, BottleneckNote{
				Lib: k.lib, Stratum: k.st, Backend: "sgoiter", Kind: "cpu",
				Detail: fmt.Sprintf("goulet %s: sgoiter %.2fx C on %s (%.1f vs %.1f ns/op)", sev, ratio, k.st, s.NsPerOp, c.NsPerOp),
			})
		}
		if s.Allocs > 0 && c.PayloadBytes >= 1024 {
			notes = append(notes, BottleneckNote{
				Lib: k.lib, Stratum: k.st, Backend: "sgoiter", Kind: "alloc",
				Detail: fmt.Sprintf("%d allocs / %d B heap delta on bulk-ish stratum — escape/slice churn", s.Allocs, s.AllocBytes),
			})
		}
		if s.Phase == "bulk" && s.MBs > 0 && c.MBs > 0 && s.MBs < c.MBs*0.4 {
			notes = append(notes, BottleneckNote{
				Lib: k.lib, Stratum: k.st, Backend: "sgoiter", Kind: "cpu",
				Detail: fmt.Sprintf("bulk bandwidth %.0f MB/s vs C %.0f MB/s", s.MBs, c.MBs),
			})
		}
	}
	return notes
}

// FormatReportMD human report with disk context first.
func FormatReportMD(r *Report) string {
	var b strings.Builder
	b.WriteString("# Probebench stratifié — CPU / RAM / disques\n\n")
	b.WriteString("Stamp: " + r.Stamp + "\n\n")
	b.WriteString("## Disques hôtes (piège classique)\n\n")
	b.WriteString("Les micro-bench **ne doivent pas** écrire artefacts sur `/data` (volume bulk).\n\n")
	for _, d := range r.HostDisks {
		b.WriteString("- " + FormatDiskEnv(d) + "\n")
	}
	b.WriteString("\n**Workdir CPU probes:** " + FormatDiskEnv(r.WorkDisk) + "\n\n")
	if len(r.IO) > 0 {
		b.WriteString("## I/O séquentiel 64 MiB (par montage)\n\n")
		b.WriteString("| id | mount | type | write MB/s | read MB/s | err |\n|----|-------|------|------------|-----------|-----|\n")
		for _, io := range r.IO {
			b.WriteString(fmt.Sprintf("| %s | %s | %s | %.1f | %.1f | %s |\n",
				io.ID, io.Disk.Target, io.Disk.FSType, io.WriteMBs, io.ReadMBs, io.Error))
		}
		b.WriteString("\n")
	}
	b.WriteString("## Probes kernel (in-process) — triangle C / sgoiter / ccgo\n\n")
	b.WriteString("| lib | stratum | phase | C ns | sgo ns | ccgo ns | sgo/C | ccgo/C | sgo/ccgo | sgo MB/s |\n")
	b.WriteString("|-----|---------|-------|------|--------|---------|-------|--------|----------|----------|\n")
	type k3 struct{ lib, st string }
	order := []k3{}
	seen := map[k3]bool{}
	by := map[k3]map[string]ProbeLine{}
	for _, p := range r.Probes {
		kk := k3{p.Lib, p.Stratum}
		if !seen[kk] {
			seen[kk] = true
			order = append(order, kk)
		}
		if by[kk] == nil {
			by[kk] = map[string]ProbeLine{}
		}
		by[kk][p.Backend] = p
	}
	for _, kk := range order {
		m := by[kk]
		c, s, g := m["c_gcc_O2"], m["sgoiter"], m["ccgo"]
		ph := s.Phase
		if ph == "" {
			ph = c.Phase
		}
		cns, sns, gns := "—", "—", "—"
		rsc, rgc, rsg := "—", "—", "—"
		smb := "—"
		if c.Error != "" {
			cns = "ERR"
		} else if c.Iters > 0 {
			cns = fmt.Sprintf("%.1f", c.NsPerOp)
		}
		if s.Error != "" {
			sns = "ERR"
		} else if s.Iters > 0 {
			sns = fmt.Sprintf("%.1f", s.NsPerOp)
			smb = fmt.Sprintf("%.1f", s.MBs)
		}
		if g.Error != "" {
			gns = "ERR"
		} else if g.Iters > 0 {
			gns = fmt.Sprintf("%.1f", g.NsPerOp)
		}
		if c.NsPerOp > 0 && s.NsPerOp > 0 && s.Error == "" && c.Error == "" {
			rsc = fmt.Sprintf("%.2f×", s.NsPerOp/c.NsPerOp)
		}
		if c.NsPerOp > 0 && g.NsPerOp > 0 && g.Error == "" && c.Error == "" {
			rgc = fmt.Sprintf("%.2f×", g.NsPerOp/c.NsPerOp)
		}
		if g.NsPerOp > 0 && s.NsPerOp > 0 && s.Error == "" && g.Error == "" {
			rsg = fmt.Sprintf("%.2f×", s.NsPerOp/g.NsPerOp)
		}
		b.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n",
			kk.lib, kk.st, ph, cns, sns, gns, rsc, rgc, rsg, smb))
	}
	if len(r.Bottleneck) > 0 {
		b.WriteString("\n## Goulets heuristiques\n\n")
		for _, n := range r.Bottleneck {
			b.WriteString(fmt.Sprintf("- **%s/%s** [%s/%s]: %s\n", n.Lib, n.Stratum, n.Backend, n.Kind, n.Detail))
		}
	}
	b.WriteString("\n## Micro-opt — ordre suggéré\n\n")
	b.WriteString("1. Strates `hot_l1` / `block` avec ratio sgo/C ≥ 3.5× (CPU kernel).\n")
	b.WriteString("2. Strates `bulk` avec allocs > 0 (échapper slices, pools).\n")
	b.WriteString("3. Ne pas optimiser sur workdir `/data` ni mélanger IO disque et CPU.\n")
	return b.String()
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func copyFile(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0o644)
}
