package collectors

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"rpi-metrics/internal/metrics"
	"rpi-metrics/constants"
)

type CPUTempSysfs struct {
	Path string // default: /sys/class/thermal/thermal_zone0/temp
}

func (c CPUTempSysfs) ID() string { return "cpu_temp" }

func (c CPUTempSysfs) Collect(ctx context.Context) ([]metrics.Sample, error) {
	_ = ctx

	path := c.Path
	if path == "" {
		path = constants.DefaultCPUTempSysfsPath
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read sysfs temp: %w", err)
	}

	s := strings.TrimSpace(string(b))
	// Expect millidegrees Celsius (integer)
	raw, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse sysfs temp %q: %w", s, err)
	}

	celsius := float64(raw) / 1000.0

	return []metrics.Sample{
		{
			Name:      "cpu_temperature",
			Value:     celsius,
			Unit:      "celsius",
			Timestamp: time.Now().UTC(),
			Labels: map[string]string{
				"source": "sysfs",
				"path":   path,
			},
		},
	}, nil
}
