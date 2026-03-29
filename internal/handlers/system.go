package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
)

type SystemResponse struct {
	Hostname   string  `json:"hostname"`
	OS         string  `json:"os"`
	Platform   string  `json:"platform"`
	UptimeSec  uint64  `json:"uptime_sec"`
	CPUCount   int     `json:"cpu_count"`
	CPUPercent float64 `json:"cpu_percent"`
	Load1      float64 `json:"load1"`
	Load5      float64 `json:"load5"`
	Load15     float64 `json:"load15"`
	MemTotal   uint64  `json:"mem_total_bytes"`
	MemUsed    uint64  `json:"mem_used_bytes"`
	MemPercent float64 `json:"mem_percent"`
	DiskTotal  uint64  `json:"disk_total_bytes"`
	DiskUsed   uint64  `json:"disk_used_bytes"`
	DiskPath   string  `json:"disk_path"`
}

func System(w http.ResponseWriter, r *http.Request) {
	h, err := host.Info()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	v, err := mem.VirtualMemory()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	cp, err := cpu.Percent(0, false)
	cpuPct := 0.0
	if err == nil && len(cp) > 0 {
		cpuPct = cp[0]
	}
	cnt, _ := cpu.Counts(true)
	l, err := load.Avg()
	if err != nil {
		l = &load.AvgStat{}
	}
	dpath := "/"
	du, err := disk.Usage(dpath)
	if err != nil {
		du = &disk.UsageStat{}
	}
	out := SystemResponse{
		Hostname:   h.Hostname,
		OS:         h.OS,
		Platform:   h.Platform,
		UptimeSec:  h.Uptime,
		CPUCount:   cnt,
		CPUPercent: cpuPct,
		Load1:      l.Load1,
		Load5:      l.Load5,
		Load15:     l.Load15,
		MemTotal:   v.Total,
		MemUsed:    v.Used,
		MemPercent: v.UsedPercent,
		DiskTotal:  du.Total,
		DiskUsed:   du.Used,
		DiskPath:   dpath,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}
