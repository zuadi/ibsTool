package htop

import (
	"encoding/json"
	"fmt"
	"ibsTool/logging"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"

	wsModels "github.com/zuadi/webServer/models"
)

// SystemStats payload to send over WebSocket
type SystemStats struct {
	Service      string        `json:"service"`
	CPUCount     int           `json:"cpu_count"`
	CPULoad      []float64     `json:"cpu_load"`
	MemTotal     uint64        `json:"mem_total"`
	MemUsed      uint64        `json:"mem_used"`
	MemPct       float64       `json:"mem_pct"`
	Tasks        []ProcessTask `json:"tasks"`
	TotalTasks   int           `json:"total_tasks"`
	RunningTasks int           `json:"running_tasks"`
	Load1        float64       `json:"load_1"`
	Load5        float64       `json:"load_5"`
	Load15       float64       `json:"load_15"`
	UptimeStr    string        `json:"uptime_str"`
}

type ProcessTask struct {
	PID     int     `json:"pid"`
	User    string  `json:"user"`
	CPU     float64 `json:"cpu"`
	Memory  float64 `json:"memory"`
	Command string  `json:"command"`
}

type HTop struct {
	websocket   *wsModels.WSClient
	ticker      *time.Ticker
	logger      *logging.Logger
	clientCount int
}

func NewHtopProcess(ws *wsModels.WSClient, l *logging.Logger) *HTop {
	h := &HTop{websocket: ws, logger: l}

	ws.Listen(func(data any) {
		d, ok := data.([]byte)
		if !ok {
			l.BroadcastLog("error: wroing datatype not []byte")
		}
		var jsonData struct {
			Service string `json:"service"`
			Action  string `json:"action"`
		}

		err := json.Unmarshal(d, &jsonData)
		if err != nil {
			l.BroadcastLog(err)
		}

		if jsonData.Service != "htop" {
			return
		} else if jsonData.Action == "disconnect" {
			h.clientCount--
			if h.clientCount == 0 {
				h.Stop()
			}
		}
	})

	return h
}

func (h *HTop) Start() {
	h.clientCount++
	if h.ticker != nil {
		return
	}
	h.ticker = time.NewTicker(1 * time.Second)

	h.logger.BroadcastLog("start htop")

	for range h.ticker.C {
		stats := getSystemMetrics()
		payload, _ := json.Marshal(stats)

		h.websocket.Broadcast(wsModels.TextMessage, payload)
	}
}

func (h *HTop) ChangeInterval(interval int) {
	h.ticker.Reset(time.Duration(interval) * time.Millisecond)
}

func (h *HTop) Stop() {
	h.logger.BroadcastLog("stop htop")
	h.ticker.Stop()
}

// Fetches host kernel stats (Includes safe fallback simulation logic if running outside Linux)
func getSystemMetrics() SystemStats {
	numCPU := runtime.NumCPU()
	cpuLoads, _ := cpu.Percent(0, true)

	vMem, _ := mem.VirtualMemory()
	var memTotal, memUsed uint64
	var memPct float64
	if vMem != nil {
		memTotal = vMem.Total / 1024 / 1024
		memUsed = vMem.Used / 1024 / 1024
		memPct = vMem.UsedPercent
	}

	// 1. Get Load Averages
	var l1, l5, l15 float64
	if avg, err := load.Avg(); err == nil {
		l1, l5, l15 = avg.Load1, avg.Load5, avg.Load15
	}

	// 2. Get Uptime & Format it (e.g., "12 days, 04:32:11")
	var uptimeStr string
	if upTime, err := host.Uptime(); err == nil {
		d := upTime / (24 * 3600)
		upTime %= (24 * 3600)
		h := upTime / 3600
		upTime %= 3600
		m := upTime / 60
		s := upTime % 60
		uptimeStr = fmt.Sprintf("%d days, %02d:%02d:%02d", d, h, m, s)
	}

	// 3. Get Process Metrics & Counts
	ps, err := process.Processes()
	var tasks []ProcessTask
	var totalTasks, runningTasks int

	if err == nil {
		totalTasks = len(ps)

		// Setup parsing limit for detailed table layout
		limit := 1000
		tasks = make([]ProcessTask, 0, limit)

		for _, p := range ps {
			// Count status matches for 'Running' state
			if statuses, err := p.Status(); err == nil {
				for _, status := range statuses {
					// Check if any of the statuses indicate it's actively running
					if status == "R" || status == "running" {
						runningTasks++
						break // Count it once per process and move on
					}
				}
			}

			if len(tasks) < limit {
				name, _ := p.Name()
				if name == "" {
					continue
				}
				user, _ := p.Username()
				cpuPct, _ := p.CPUPercent()
				memPctProc, _ := p.MemoryPercent()

				tasks = append(tasks, ProcessTask{
					PID:     int(p.Pid),
					User:    user,
					CPU:     cpuPct,
					Memory:  float64(memPctProc),
					Command: name,
				})
			}
		}
	}

	return SystemStats{
		Service:      "htop",
		CPUCount:     numCPU,
		CPULoad:      cpuLoads,
		MemTotal:     memTotal,
		MemUsed:      memUsed,
		MemPct:       memPct,
		Tasks:        tasks,
		TotalTasks:   totalTasks,
		RunningTasks: runningTasks,
		Load1:        l1,
		Load5:        l5,
		Load15:       l15,
		UptimeStr:    uptimeStr,
	}
}
