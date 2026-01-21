package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"rpi-metrics/constants"
	"rpi-metrics/internal/collectors"
	"rpi-metrics/internal/metrics"
)

func main() {
	interval := flag.Duration(constants.FlagInterval, constants.DefaultCollectionInterval, constants.FlagUsageInterval)
	tempPath := flag.String(constants.FlagTempPath, constants.DefaultCPUTempSysfsPath, constants.FlagUsageTempPath)
	coolingPath := flag.String(constants.FlagCoolingPath, constants.DefaultCPUCoolingDevicefsPath, constants.FlagUsageCoolingPath)
	storagePaths := flag.String(constants.FlagStoragePaths, constants.DefaultStoragePathsCSV, constants.FlagUsageStoragePaths)
	discordWebhook := flag.String(constants.FlagDiscordWebhook, constants.DefaultDiscordWebhookURL, constants.FlagUsageDiscordWebhook)
	discordEvery := flag.Duration(constants.FlagDiscordEvery, constants.DefaultDiscordPostEvery, constants.FlagUsageDiscordEvery)
	alsoConsole := flag.Bool(constants.FlagAlsoConsole, constants.DefaultAlsoConsoleWhenDiscordOn, constants.FlagUsageAlsoConsole)
	flag.Parse()

	// Set up ordered collectors
	collectorList := []metrics.Collector{
		collectors.CPUTempSysfs{Path: *tempPath},
		&collectors.CPUUtilizationProcfs{},
		collectors.CPUCoolingDevicefs{Path: *coolingPath},
		collectors.StorageStatfs{Paths: splitCSV(*storagePaths)},
	}

	runner := metrics.Runner{Collectors: collectorList}
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

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}
