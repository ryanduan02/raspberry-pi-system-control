# rpi-metrics

A metrics agent for Raspberry Pi written in Go.  

## Features

- Collector based architecture
- Interval scheduler/runner
- Console exporter (JSON Lines to stdout)
- CPU temperature collector with sysfs:
  - `/sys/class/thermal/thermal_zone0/temp` (millidegrees Celsius)
- Storage usage collector (Linux `statfs`):
    - Total/free/available/used bytes and used percent (per configured path)

## Requirements

- Raspberry Pi OS
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

To run discord messenger: 

```
./bin/rpi-metrics -interval= {x}s -discord-webhook="https://discord.com/api/webhooks/{webook_id}" -discord-every= {x}s
```

## Running persistently (SSH disconnect safe)

If you start `rpi-metrics` directly in an SSH session, it will usually stop when the SSH connection closes (e.g. you close your laptop).

### Option A: `tmux` (recommended for interactive control)

On the Pi:

```
sudo apt-get update
sudo apt-get install -y tmux
./scripts/tmux-start.sh -- -interval=5s -discord-webhook="https://discord.com/api/webhooks/REPLACE_ME" -discord-every=1m
```

Detach without stopping it: press `Ctrl-b` then `d`.

Later, reattach:

```
./scripts/tmux-attach.sh
```

Stop it: `Ctrl-c` inside the tmux session.

Or stop the session directly:

```
./scripts/tmux-stop.sh
```

Check logs:

```
sudo journalctl -u rpi-metrics -f
```

Stop it:

```
sudo systemctl stop rpi-metrics
```

Flags
- `interval` -
    Collection interval duration (Go duration format). Examples: 500ms, 2s, 30s, 1m.

- `temp-path` -
    Path to the sysfs temperature file.

- `storage-paths` -
    Comma-separated list of filesystem paths to measure. Examples: `/` or `/,/boot`.

Example:

```
./bin/rpi-metrics -interval=2s -temp-path=/sys/class/thermal/thermal_zone0/temp

./bin/rpi-metrics -interval=5s -storage-paths=/,/boot
```

```
./bin/rpi-metrics -interval=5s -discord-webhook=https://discord.com/api/webhooks/{INSERT WEBHOOK} -discord-every=5s 
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

This tool currently exports to stdout only. It is designed so additional exporters can be added later (HTTP/Prometheus scrape endpoint, MQTT, InfluxDB, etc.).      CPU metrics reading uses sysfs for simplicity and performance.
