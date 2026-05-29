package processstats

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// InstanceStats representa el consumo real de recursos de una instancia en ejecución
type InstanceStats struct {
	CPUPercent  float64 `json:"cpu_percent"`
	MemoryBytes uint64  `json:"memory_bytes"`
}

// readProcStat lee los ticks de CPU (utime, stime) y el RSS de un proceso desde /proc/<pid>/stat
func readProcStat(pid int) (utime, stime, rss uint64, err error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0, 0, 0, err
	}
	dataStr := string(data)
	lastParen := strings.LastIndex(dataStr, ")")
	if lastParen == -1 || lastParen+2 >= len(dataStr) {
		return 0, 0, 0, fmt.Errorf("invalid stat format")
	}

	fields := strings.Fields(dataStr[lastParen+2:])
	if len(fields) < 22 {
		return 0, 0, 0, fmt.Errorf("insufficient fields in stat")
	}

	utime, _ = strconv.ParseUint(fields[11], 10, 64)
	stime, _ = strconv.ParseUint(fields[12], 10, 64)
	rss, _ = strconv.ParseUint(fields[21], 10, 64)
	return utime, stime, rss, nil
}

// readSystemCpuTicks lee el total de ticks de CPU del sistema desde /proc/stat
func readSystemCpuTicks() (uint64, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 5 || fields[0] != "cpu" {
			return 0, fmt.Errorf("invalid cpu line in /proc/stat")
		}
		var sum uint64
		for i := 1; i < len(fields); i++ {
			val, err := strconv.ParseUint(fields[i], 10, 64)
			if err != nil {
				return 0, err
			}
			sum += val
		}
		return sum, nil
	}
	return 0, fmt.Errorf("empty /proc/stat")
}

// sampleForPid realiza una lectura puntual de ticks para un PID dado.
// Es una función pura sin estado ni efectos secundarios.
func sampleForPid(pid int) (processTicks, systemTicks, rssPages uint64, ok bool) {
	u, s, rss, err := readProcStat(pid)
	if err != nil {
		return 0, 0, 0, false
	}
	sys, err := readSystemCpuTicks()
	if err != nil {
		return 0, 0, 0, false
	}
	return u + s, sys, rss, true
}

// calculatePercent calcula el porcentaje de CPU en modo Irix (estándar htop/docker stats).
// 100% = un core saturado al máximo. En N cores la escala máxima es N*100%.
func calculatePercent(prevProcess, currProcess, prevSystem, currSystem uint64) float64 {
	systemDelta := currSystem - prevSystem
	if systemDelta == 0 {
		return 0
	}
	processDelta := currProcess - prevProcess
	return (float64(processDelta) / float64(systemDelta)) * 100 * float64(runtime.NumCPU())
}
