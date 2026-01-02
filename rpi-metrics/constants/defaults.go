package constants

import "time"

const (
	DefaultCollectionInterval = 5 * time.Second

	DefaultCPUTempSysfsPath       = "/sys/class/thermal/thermal_zone0/temp"
	DefaultCPUCoolingDevicefsPath = "/sys/class/thermal/cooling_device0/cur_state"
	DefaultStoragePathsCSV        = "/"

	DefaultDiscordWebhookURL        = ""
	DefaultDiscordPostEvery         = time.Duration(0)
	DefaultAlsoConsoleWhenDiscordOn = false
)
