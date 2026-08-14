package probebench

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// DiskEnv describes where a path lives (avoids comparing NVMe vs HDD silently).
type DiskEnv struct {
	Path     string `json:"path"`
	Target   string `json:"mount_target"`
	Source   string `json:"source"`
	FSType   string `json:"fstype"`
	Options  string `json:"options,omitempty"`
	Rotational *bool `json:"rotational,omitempty"` // nil if unknown
	Device   string `json:"device,omitempty"`
}

// ResolveDiskEnv uses findmnt -T and lsblk ROTA when possible.
func ResolveDiskEnv(path string) DiskEnv {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	de := DiskEnv{Path: abs}
	out, err := exec.Command("findmnt", "-T", abs, "-no", "TARGET,SOURCE,FSTYPE,OPTIONS").Output()
	if err == nil {
		fields := strings.Fields(string(out))
		if len(fields) >= 3 {
			de.Target = fields[0]
			de.Source = fields[1]
			de.FSType = fields[2]
			if len(fields) >= 4 {
				de.Options = strings.Join(fields[3:], " ")
			}
		}
	}
	// rotational: lsblk -no ROTA /dev/X
	dev := de.Source
	if i := strings.LastIndex(dev, "["); i > 0 { // btrfs subvol
		dev = dev[:i]
	}
	dev = strings.TrimSpace(dev)
	de.Device = dev
	if strings.HasPrefix(dev, "/dev/") {
		base := filepath.Base(dev)
		// strip partition suffix for parent (nvme0n1p2 → nvme0n1)
		parent := base
		if strings.Contains(base, "nvme") {
			if j := strings.LastIndex(base, "p"); j > 0 {
				parent = base[:j]
			}
		} else {
			for len(parent) > 0 && parent[len(parent)-1] >= '0' && parent[len(parent)-1] <= '9' {
				parent = parent[:len(parent)-1]
			}
		}
		rotaPath := filepath.Join("/sys/block", parent, "queue/rotational")
		if b, err := os.ReadFile(rotaPath); err == nil {
			v := strings.TrimSpace(string(b))
			r := v == "1"
			de.Rotational = &r
		}
	}
	return de
}

// HostInventory lists canonical bench roots on this machine.
func HostInventory() []DiskEnv {
	roots := []string{"/tmp", "/devhoros", "/data", "/"}
	var out []DiskEnv
	seen := map[string]bool{}
	for _, r := range roots {
		if _, err := os.Stat(r); err != nil {
			continue
		}
		de := ResolveDiskEnv(r)
		key := de.Source + "|" + de.Target
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, de)
	}
	return out
}

// DefaultWorkDir picks a fast local path for CPU probes (prefer non-rotational NVMe tmp).
// Never default to /data (HDD/bulk) for microbench workdirs.
func DefaultWorkDir() string {
	candidates := []string{
		"/tmp/sgoiter_probebench",
		filepath.Join(os.TempDir(), "sgoiter_probebench"),
	}
	// prefer /tmp if on non-rotational
	for _, c := range candidates {
		de := ResolveDiskEnv(c)
		if de.Rotational != nil && *de.Rotational {
			continue
		}
		_ = os.MkdirAll(c, 0o755)
		return c
	}
	_ = os.MkdirAll(candidates[0], 0o755)
	return candidates[0]
}

// IOStratum is a sequential disk probe (not kernel CPU).
type IOStratum struct {
	ID    string
	Label string
	Bytes int64
	Path  string // directory to write into
}

// DefaultIOStrata one write+read pass per interesting mount.
func DefaultIOStrata() []IOStratum {
	var out []IOStratum
	for _, de := range HostInventory() {
		// skip tiny/loop
		if de.FSType == "" {
			continue
		}
		dir := filepath.Join(de.Target, ".sgoiter_probe_io")
		if de.Target == "/" {
			dir = "/tmp/.sgoiter_probe_io"
		}
		if de.Target == "/devhoros" {
			dir = "/devhoros/.sgoiter_probe_io"
		}
		if de.Target == "/data" {
			dir = "/data/.sgoiter_probe_io"
		}
		out = append(out,
			IOStratum{ID: "io_seq_64m_" + sanitizeID(de.Target), Label: "seq 64MiB on " + de.Target, Bytes: 64 << 20, Path: dir},
		)
	}
	return out
}

func sanitizeID(s string) string {
	s = strings.TrimPrefix(s, "/")
	if s == "" {
		return "root"
	}
	return strings.ReplaceAll(s, "/", "_")
}

// RunIOProbe sequential write then read; returns MB/s and notes rotational.
func RunIOProbe(st IOStratum) (writeMBs, readMBs float64, err error) {
	if err := os.MkdirAll(st.Path, 0o755); err != nil {
		return 0, 0, err
	}
	path := filepath.Join(st.Path, "probe.bin")
	defer os.Remove(path)
	buf := make([]byte, 1<<20) // 1 MiB
	for i := range buf {
		buf[i] = byte(i)
	}
	n := st.Bytes
	// write
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, 0, err
	}
	w := bufio.NewWriterSize(f, 1<<20)
	t0 := now()
	var left = n
	for left > 0 {
		chunk := int64(len(buf))
		if chunk > left {
			chunk = left
		}
		if _, err := w.Write(buf[:chunk]); err != nil {
			f.Close()
			return 0, 0, err
		}
		left -= chunk
	}
	if err := w.Flush(); err != nil {
		f.Close()
		return 0, 0, err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return 0, 0, err
	}
	f.Close()
	wt := now() - t0
	if wt > 0 {
		writeMBs = float64(n) / wt / (1024 * 1024)
	}
	// read
	f2, err := os.Open(path)
	if err != nil {
		return writeMBs, 0, err
	}
	r := bufio.NewReaderSize(f2, 1<<20)
	t1 := now()
	left = n
	for left > 0 {
		chunk := int64(len(buf))
		if chunk > left {
			chunk = left
		}
		if _, err := r.Read(buf[:chunk]); err != nil {
			f2.Close()
			return writeMBs, 0, err
		}
		left -= chunk
	}
	f2.Close()
	rt := now() - t1
	if rt > 0 {
		readMBs = float64(n) / rt / (1024 * 1024)
	}
	return writeMBs, readMBs, nil
}

func now() float64 {
	// use Go time via external to avoid importing time in comment - actually use time
	return float64(mustUnixNano()) / 1e9
}

func mustUnixNano() int64 {
	// isolated for testability
	return unixNano()
}

// filled in hostenv_time.go
var unixNano = func() int64 {
	return 0
}

// FormatDiskEnv one line for reports.
func FormatDiskEnv(de DiskEnv) string {
	rota := "?"
	if de.Rotational != nil {
		if *de.Rotational {
			rota = "HDD/rotational"
		} else {
			rota = "SSD/NVMe"
		}
	}
	return fmt.Sprintf("%s → %s (%s, %s, %s)", de.Path, de.Target, de.Source, de.FSType, rota)
}

// ParseMaxRSSKiB from /usr/bin/time -v stderr.
func ParseMaxRSSKiB(timeVstderr string) int64 {
	for _, ln := range strings.Split(timeVstderr, "\n") {
		if strings.Contains(ln, "Maximum resident set size") {
			parts := strings.Fields(ln)
			if len(parts) == 0 {
				continue
			}
			v, err := strconv.ParseInt(parts[len(parts)-1], 10, 64)
			if err == nil {
				return v
			}
		}
	}
	return 0
}
