package constants

const (
	FlagInterval       = "interval"
	FlagTempPath       = "temp-path"
	FlagCoolingPath    = "cooling-path"
	FlagStoragePaths   = "storage-paths"
	FlagDiscordWebhook = "discord-webhook"
	FlagDiscordEvery   = "discord-every"
	FlagAlsoConsole    = "also-console"
)

const (
	FlagUsageInterval       = "collection interval (e.g. 2s, 500ms, 1m)"
	FlagUsageTempPath       = "sysfs path for CPU temperature"
	FlagUsageCoolingPath    = "sysfs path for cooling device"
	FlagUsageStoragePaths   = "Comma-separated list of filesystem paths to measure (e.g. /,/boot)"
	FlagUsageDiscordWebhook = "Discord webhook URL (optional)"
	FlagUsageDiscordEvery   = "How often to post to Discord (0 disables). e.g. 1m, 10m, 1h"
	FlagUsageAlsoConsole    = "When Discord is enabled, also print JSON to stdout"
)
