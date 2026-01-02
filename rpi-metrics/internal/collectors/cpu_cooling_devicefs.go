package collectors

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"rpi-metrics/internal/metrics"
)

type CPUCoolingDevicefs struct {
	Path string // default: /sys/class/thermal/cooling_device0/cur_state
}

func (c CPUCoolingDevicefs) ID() string { return "cpu_cooling_device" }

func (c CPUCoolingDevicefs) Collect(ctx context.Context) ([]metrics.Sample, error) {
	_ = ctx

	path := c.Path

	if path == "" {
		path = "/sys/class/thermal/cooling_device0/cur_state"
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read cooling device cur_state: %w", err)
	}

	s := strings.TrimSpace(string(b))

	raw, err := strconv.ParseInt(s, 10, 64)

	cooling_cur_state := float64(raw)

	return []metrics.Sample{
		{
			Name:      "cooling_state",
			Value:     cooling_cur_state,
			Unit:      "from 0 - 4",
			Timestamp: time.Now().UTC(),
			Labels: map[string]string{
				"source": "sysfs",
				"path":   path,
			},
		},
	}, nil
	


	
}


