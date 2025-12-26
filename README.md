# rpi-metrics

A metrics agent for Raspberry Pi written in Go.  

Current milestone: collect CPU temperature from sysfs on an interval and emit structured JSON to stdout. The code is set up so adding a new metric is just adding a new collector and registering it.

## Features

- Collector based architecture
- Interval scheduler/runner
- Console exporter (JSON Lines to stdout)
- CPU temperature collector with sysfs:
  - `/sys/class/thermal/thermal_zone0/temp` (millidegrees Celsius)

## Requirements

- Raspberry Pi OS (or another Linux that exposes CPU temp via sysfs)
- Go installed (Go 1.22+ recommended)

## Usage

To verify the temperature source exists:

`cat /sys/class/thermal/thermal_zone0/temp`

You should see an integer like 51250 (meaning 51.250°C).
Build

From the rpi-metrics/ directory:

```
cd ~/Code/raspberry-pi-system-control/rpi-metrics
go mod tidy
mkdir -p bin
go build -o ./bin/rpi-metrics ./cmd/rpi-metrics
```

Run
Basic run (collect every 5 seconds)

`./bin/rpi-metrics -interval=5s`

Example output (JSON line per interval):

```
{"collected_at":"2025-12-26T12:45:06.543191272Z","samples":
[{"name":"cpu_temperature","value":51.25,"unit":"celsius",
"ts":"2025-12-26T12:45:06.543188513Z","labels":{"path":"/sys/
class/thermal/thermal_zone0/temp","source":"sysfs"}}]}
```

Flags
- `interval` -
    Collection interval duration (Go duration format). Examples: 500ms, 2s, 30s, 1m.

- `temp-path` -
    Path to the sysfs temperature file.

Example:

```
./bin/rpi-metrics -interval=2s -temp-path=/sys/class/thermal/thermal_zone0/temp
```

## Adding new metrics

Add a new collector implementing the metrics. Collector interface:

```
type Collector interface {
    ID() string
    Collect(ctx context.Context) ([]Sample, error)
}
```

Then register it in `cmd/rpi-metrics/main.go` using:

`metrics.Register(yourCollector{})`

Collectors live in `internal/collectors/` and return one or more `metrics.Sample` values, which are aggregated by the runner and emitted by the exporter.

## Notes

This tool currently exports to stdout only. It is designed so additional exporters can be added later (HTTP/Prometheus scrape endpoint, MQTT, InfluxDB, etc.).      CPU temperature reading uses sysfs for simplicity and performance.