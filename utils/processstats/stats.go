package processstats

import (
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// InstanceStats representa el consumo real de recursos de una instancia en ejecución
type InstanceStats struct {
	CPUPercent  float64 `json:"cpu_percent"`
	MemoryBytes uint64  `json:"memory_bytes"`
}

// GetProcessStats obtiene el % de CPU y RAM real (RSS) de un PID usando "ps" en Linux
func GetProcessStats(pid int) InstanceStats {
	if pid <= 0 {
		return InstanceStats{}
	}
	if runtime.GOOS == "windows" {
		return InstanceStats{}
	}
	cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "%cpu,rss")
	out, err := cmd.Output()
	if err != nil {
		return InstanceStats{}
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return InstanceStats{}
	}
	fields := strings.Fields(lines[1])
	if len(fields) < 2 {
		return InstanceStats{}
	}
	cpu, _ := strconv.ParseFloat(fields[0], 64)
	rssKb, _ := strconv.ParseUint(fields[1], 10, 64)
	return InstanceStats{
		CPUPercent:  cpu,
		MemoryBytes: rssKb * 1024, // Convertir de KB a bytes
	}
}
