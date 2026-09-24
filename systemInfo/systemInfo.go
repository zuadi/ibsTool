package systeminfo

import (
	"ibsTool/htop"
	"ibsTool/models"
	"ibsTool/utils"
	"math"
	"runtime"

	"github.com/shirou/gopsutil/disk"
)

func GetInfo() (systemInfo models.System) {

	// 1. Network Interfaces
	if ethernet, err := utils.GetActiveEthernet(); err == nil {
		systemInfo.Lan = ethernet
	}

	if wlan, err := utils.GetActiveInterfaceByName("wlan"); err == nil {
		systemInfo.Wlan = wlan
		systemInfo.SSID = wlan.SSID
		systemInfo.APChannel = wlan.Channel
	}

	if ts, err := utils.GetTailscaleAdapter(); err == nil {
		systemInfo.Tailscale = ts
	}

	// 2. System Performance Metrics (CPU, RAM, Disk)
	metrics := htop.GetSystemMetrics()

	var sum float64
	for _, v := range metrics.CPULoad {
		sum += v
	}

	systemInfo.CPU = math.Round(sum/float64(len(metrics.CPULoad))*10) / 10
	systemInfo.RAM = math.Round(100/float64(metrics.MemTotal)*float64(metrics.MemUsed)*10) / 10

	path := "/"
	switch runtime.GOOS {
	case "windows":
		path = "C:\\"

	}

	usage, _ := disk.Usage(path)

	systemInfo.Disk = math.Round(100/float64(usage.Total)*float64(usage.Used)*10) / 10

	// 3. Connected Access Point Stations
	if stations, err := utils.GetHostapdStations(); err == nil {
		systemInfo.Stations = stations
		systemInfo.ClientsCount = len(stations)
	} else {
		systemInfo.Stations = []models.Station{}
		systemInfo.ClientsCount = 0
	}

	return systemInfo
}
