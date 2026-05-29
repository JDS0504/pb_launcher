package processstats

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
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

// GetProcessStats obtiene el % de CPU y RAM real (RSS) de un PID usando /proc en Linux de forma instantánea y stateless (KISS)
func GetProcessStats(pid int) InstanceStats {
	if pid <= 0 {
		return InstanceStats{}
	}
	if runtime.GOOS == "windows" {
		return InstanceStats{}
	}

	// Primera lectura de muestras
	u1, s1, rss1, err := readProcStat(pid)
	if err != nil {
		return InstanceStats{}
	}
	sys1, err := readSystemCpuTicks()
	if err != nil {
		return InstanceStats{
			MemoryBytes: rss1 * uint64(os.Getpagesize()),
		}
	}

	// Intervalo de muestreo para obtener suficiente resolución de ticks en Linux (300ms)
	time.Sleep(300 * time.Millisecond)

	// Segunda lectura de muestras
	u2, s2, _, err := readProcStat(pid)
	if err != nil {
		return InstanceStats{
			MemoryBytes: rss1 * uint64(os.Getpagesize()),
		}
	}
	sys2, err := readSystemCpuTicks()
	if err != nil {
		return InstanceStats{
			MemoryBytes: rss1 * uint64(os.Getpagesize()),
		}
	}

	processDelta := (u2 + s2) - (u1 + s1)
	systemDelta := sys2 - sys1

	var cpuPercent float64
	if systemDelta > 0 {
		// Multiplicado por el número de CPU cores del host para normalizar a 100% de la capacidad de la máquina
		cpuPercent = (float64(processDelta) / float64(systemDelta)) * 100 * float64(runtime.NumCPU())
		if cpuPercent > 100 {
			cpuPercent = 100
		}
	}

	return InstanceStats{
		CPUPercent:  cpuPercent,
		MemoryBytes: rss1 * uint64(os.Getpagesize()),
	}
}
