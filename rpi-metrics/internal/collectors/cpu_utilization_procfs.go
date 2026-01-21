package collectors

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"rpi-metrics/internal/metrics"
)

type CPUUtilizationProcfs struct {
	Path string // default: /proc/stat

	mu       sync.Mutex
	last     map[string]cpuCounters
	haveLast bool
}

func (c *CPUUtilizationProcfs) ID() string { return "cpu_utilization" }

func (c *CPUUtilizationProcfs) Collect(ctx context.Context) ([]metrics.Sample, error) {
	_ = ctx

	path := c.Path
	if path == "" {
		path = "/proc/stat"
	}

	counters, err := readProcStatCPUAll(path)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now().UTC()
	samples := []metrics.Sample{}

	if !c.haveLast {
		c.last = counters
		c.haveLast = true
		return samples, nil
	}

	for cpuID, curr := range counters {
		prev, ok := c.last[cpuID]
		if !ok {
			continue
		}

		deltaIdle := float64(curr.idle - prev.idle)
		deltaTotal := float64(curr.total - prev.total)
		if deltaTotal <= 0 {
			continue
		}

		usage := (deltaTotal - deltaIdle) / deltaTotal * 100.0
		if usage < 0 {
			usage = 0
		}
		if usage > 100 {
			usage = 100
		}

		labels := map[string]string{
			"source": "procfs",
			"path":   path,
			"cpu":    cpuID,
		}
		if cpuID == "cpu" {
			labels["cpu"] = "total"
		}

		samples = append(samples, metrics.Sample{
			Name:      "cpu_utilization",
			Value:     usage,
			Unit:      "percent",
			Timestamp: now,
			Labels:    labels,
		})
	}

	c.last = counters
	return samples, nil
}

type cpuCounters struct {
	idle  uint64
	total uint64
}

func readProcStatCPUAll(path string) (map[string]cpuCounters, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	out := make(map[string]cpuCounters)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu") {
			break
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		cpuID := fields[0]

		var values []uint64
		for _, f := range fields[1:] {
			v, parseErr := strconv.ParseUint(f, 10, 64)
			if parseErr != nil {
				return nil, fmt.Errorf("parse %s field %q: %w", path, f, parseErr)
			}
			values = append(values, v)
		}

		var sum uint64
		for _, v := range values {
			sum += v
		}

		idleVal := uint64(0)
		if len(values) >= 4 {
			idleVal += values[3]
		}
		if len(values) >= 5 {
			idleVal += values[4]
		}

		out[cpuID] = cpuCounters{idle: idleVal, total: sum}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("read %s: no cpu stats found", path)
	}
	return out, nil
}
