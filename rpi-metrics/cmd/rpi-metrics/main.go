package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"rpi-metrics/internal/collectors"
	"rpi-metrics/internal/metrics"
)

func main() {
	interval := flag.Duration("interval", 5*time.Second, "collection interval (e.g. 2s, 500ms, 1m)")
	tempPath := flag.String("temp-path", "/sys/class/thermal/thermal_zone0/temp", "sysfs path for CPU temperature")
	discordWebhook := flag.String("discord-webhook", "", "Discord webhook URL (optional)")
	discordEvery := flag.Duration("discord-every", 0, "How often to post to Discord (0 disables). e.g. 1m, 10m, 1h")
	alsoConsole := flag.Bool("also-console", false, "When Discord is enabled, also print JSON to stdout")
	flag.Parse()

	// Register collectors
	if err := metrics.Register(collectors.CPUTempSysfs{Path: *tempPath}); err != nil {
		log.Fatalf("register collector: %v", err)
	}

	runner := metrics.Runner{Collectors: metrics.All()}
	discordEnabled := *discordWebhook != "" && *discordEvery > 0
	consoleEnabled := !discordEnabled || *alsoConsole

	var consoleExporter metrics.Exporter
	if consoleEnabled {
		consoleExporter = metrics.ConsoleExporter{Out: os.Stdout}
	}

	var (
		latestMu  sync.RWMutex
		latestRes metrics.Result
		haveRes   bool
	)

	var discordExporter metrics.Exporter
	if discordEnabled {
		discordExporter = &metrics.DiscordWebhookExporter{
			WebhookURL: *discordWebhook,
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown on SIGINT/SIGTERM
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	if discordExporter != nil {
		discordTicker := time.NewTicker(*discordEvery)
		defer discordTicker.Stop()

		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-discordTicker.C:
					latestMu.RLock()
					if !haveRes {
						latestMu.RUnlock()
						continue
					}
					res := metrics.Result{
						Samples: append([]metrics.Sample(nil), latestRes.Samples...),
						Errors:  append([]metrics.CollectorError(nil), latestRes.Errors...),
					}
					latestMu.RUnlock()

					if err := discordExporter.Export(ctx, res); err != nil {
						log.Printf("discord export error: %v", err)
					}
				}
			}
		}()
	}

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	// Collect immediately once, then on interval
	for {
		res := runner.CollectOnce(ctx)

		latestMu.Lock()
		latestRes = res
		haveRes = true
		latestMu.Unlock()

		if consoleExporter != nil {
			if err := consoleExporter.Export(ctx, res); err != nil {
				log.Printf("export error: %v", err)
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
