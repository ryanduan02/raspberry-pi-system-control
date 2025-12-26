package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"rpi-metrics/internal/collectors"
	"rpi-metrics/internal/metrics"
)

func main() {
	interval := flag.Duration("interval", 5*time.Second, "collection interval (e.g. 2s, 500ms, 1m)")
	tempPath := flag.String("temp-path", "/sys/class/thermal/thermal_zone0/temp", "sysfs path for CPU temperature")
	flag.Parse()

	// Register collectors
	if err := metrics.Register(collectors.CPUTempSysfs{Path: *tempPath}); err != nil {
		log.Fatalf("register collector: %v", err)
	}

	runner := metrics.Runner{Collectors: metrics.All()}
	exporter := metrics.ConsoleExporter{Out: os.Stdout}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown on SIGINT/SIGTERM
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	// Collect immediately once, then on interval
	for {
		res := runner.CollectOnce(ctx)
		if err := exporter.Export(res); err != nil {
			log.Printf("export error: %v", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
