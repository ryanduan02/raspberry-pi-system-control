package collectors

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"rpi-metrics/internal/metrics"
)

type StorageStatfs struct {
	// Paths are filesystem paths to measure (e.g. "/", "/boot").
	// Each path is resolved to its containing mount point for labels (best-effort).
	Paths []string
}

func (c StorageStatfs) ID() string { return "storage_usage" }

type mountInfoEntry struct {
	mountPoint string
	fsType     string
	source     string
}

func (c StorageStatfs) Collect(ctx context.Context) ([]metrics.Sample, error) {
	_ = ctx

	paths := c.Paths
	if len(paths) == 0 {
		paths = []string{"/"}
	}

	mounts, _ := readMountInfo("/proc/self/mountinfo")

	now := time.Now().UTC()
	out := make([]metrics.Sample, 0, len(paths)*5)
	var lastErr error

	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		p = filepath.Clean(p)

		var st syscall.Statfs_t
		if err := syscall.Statfs(p, &st); err != nil {
			lastErr = fmt.Errorf("statfs %q: %w", p, err)
			continue
		}

		blockSize := uint64(st.Bsize)
		total := uint64(st.Blocks) * blockSize
		free := uint64(st.Bfree) * blockSize
		avail := uint64(st.Bavail) * blockSize
		used := uint64(0)
		if total >= free {
			used = total - free
		}

		usedPercent := 0.0
		if total > 0 {
			// Prefer "available" (Bavail) for percent (closer to `df` output for non-root)
			usedPercent = (float64(total-avail) / float64(total)) * 100.0
		}

		labels := map[string]string{"path": p}
		if mi, ok := bestMountForPath(mounts, p); ok {
			if mi.mountPoint != "" {
				labels["mount_point"] = mi.mountPoint
			}
			if mi.fsType != "" {
				labels["fs_type"] = mi.fsType
			}
			if mi.source != "" {
				labels["source"] = mi.source
			}
		}

		out = append(out,
			metrics.Sample{Name: "storage_total_bytes", Value: float64(total), Unit: "bytes", Timestamp: now, Labels: labels},
			metrics.Sample{Name: "storage_free_bytes", Value: float64(free), Unit: "bytes", Timestamp: now, Labels: labels},
			metrics.Sample{Name: "storage_available_bytes", Value: float64(avail), Unit: "bytes", Timestamp: now, Labels: labels},
			metrics.Sample{Name: "storage_used_bytes", Value: float64(used), Unit: "bytes", Timestamp: now, Labels: labels},
			metrics.Sample{Name: "storage_used_percent", Value: usedPercent, Unit: "percent", Timestamp: now, Labels: labels},
		)
	}

	if len(out) == 0 {
		if lastErr != nil {
			return nil, lastErr
		}
		return nil, fmt.Errorf("no storage paths configured")
	}
	return out, nil
}

func readMountInfo(path string) ([]mountInfoEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []mountInfoEntry
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		// Split on the required separator " - "
		parts := strings.SplitN(line, " - ", 2)
		if len(parts) != 2 {
			continue
		}

		left := strings.Fields(parts[0])
		right := strings.Fields(parts[1])
		if len(left) < 5 || len(right) < 2 {
			continue
		}

		mountPoint := unescapeMountInfoField(left[4])
		fsType := right[0]
		source := unescapeMountInfoField(right[1])

		out = append(out, mountInfoEntry{mountPoint: mountPoint, fsType: fsType, source: source})
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func bestMountForPath(mounts []mountInfoEntry, path string) (mountInfoEntry, bool) {
	best := mountInfoEntry{}
	bestLen := -1

	for _, m := range mounts {
		mp := m.mountPoint
		if mp == "" {
			continue
		}
		if path == mp || (mp != "/" && strings.HasPrefix(path, mp+"/")) || (mp == "/" && strings.HasPrefix(path, "/")) {
			if len(mp) > bestLen {
				best = m
				bestLen = len(mp)
			}
		}
	}

	if bestLen >= 0 {
		return best, true
	}
	return mountInfoEntry{}, false
}

// mountinfo escapes spaces and other bytes as octal: "\040".
func unescapeMountInfoField(s string) string {
	if !strings.Contains(s, "\\") {
		return s
	}

	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+3 >= len(s) {
			b = append(b, s[i])
			continue
		}

		// Expect 3 octal digits.
		o1, o2, o3 := s[i+1], s[i+2], s[i+3]
		if o1 < '0' || o1 > '7' || o2 < '0' || o2 > '7' || o3 < '0' || o3 > '7' {
			b = append(b, s[i])
			continue
		}

		v := (o1-'0')*64 + (o2-'0')*8 + (o3 - '0')
		b = append(b, byte(v))
		i += 3
	}

	return string(b)
}
