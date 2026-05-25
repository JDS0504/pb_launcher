//go:build linux

package systemstatus

import (
	"bufio"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	cpuMutex    sync.Mutex
	lastIdle    uint64
	lastTotal   uint64
	lastChecked time.Time
)

func getCPUStats() (idle, total uint64, err error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 5 || fields[0] != "cpu" {
			return 0, 0, os.ErrInvalid
		}
		var sum uint64
		for i := 1; i < len(fields); i++ {
			val, err := strconv.ParseUint(fields[i], 10, 64)
			if err != nil {
				return 0, 0, err
			}
			sum += val
			if i == 4 { // idle is index 4 (cpu, user, nice, system, idle)
				idle = val
			}
		}
		return idle, sum, nil
	}
	return 0, 0, os.ErrInvalid
}

func collectCPU() CPUInfo {
	cpuMutex.Lock()
	defer cpuMutex.Unlock()

	currIdle, currTotal, err := getCPUStats()
	if err != nil {
		return CPUInfo{UsagePercent: 0, Cores: runtime.NumCPU()}
	}

	var usage float64 = 0
	if !lastChecked.IsZero() {
		totalDiff := currTotal - lastTotal
		idleDiff := currIdle - lastIdle
		if totalDiff > 0 && totalDiff >= idleDiff {
			active := totalDiff - idleDiff
			usage = (float64(active) / float64(totalDiff)) * 100
		}
	} else {
		// First call, initialize and sleep to provide a smooth value on the first request
		lastIdle = currIdle
		lastTotal = currTotal
		lastChecked = time.Now()
		cpuMutex.Unlock()
		time.Sleep(100 * time.Millisecond)
		cpuMutex.Lock()
		return collectCPU()
	}

	lastIdle = currIdle
	lastTotal = currTotal
	lastChecked = time.Now()

	if usage < 0 {
		usage = 0
	} else if usage > 100 {
		usage = 100
	}

	return CPUInfo{
		UsagePercent: usage,
		Cores:        runtime.NumCPU(),
	}
}

func collectRAM() RAMInfo {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return RAMInfo{}
	}
	defer file.Close()

	var memTotal, memAvailable uint64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		key := fields[0]
		val, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		if key == "MemTotal:" {
			memTotal = val * 1024 // kB to bytes
		} else if key == "MemAvailable:" {
			memAvailable = val * 1024 // kB to bytes
		}
	}

	if memTotal == 0 {
		return RAMInfo{}
	}

	used := memTotal - memAvailable
	percent := (float64(used) / float64(memTotal)) * 100

	return RAMInfo{
		TotalBytes:   memTotal,
		UsedBytes:    used,
		FreeBytes:    memAvailable,
		UsagePercent: percent,
	}
}

func collectDisk(path string) DiskInfo {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return DiskInfo{Path: path}
	}

	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bfree * uint64(stat.Bsize)
	used := total - free
	var percent float64
	if total > 0 {
		percent = (float64(used) / float64(total)) * 100
	}

	return DiskInfo{
		TotalBytes:   total,
		UsedBytes:    used,
		FreeBytes:    free,
		UsagePercent: percent,
		Path:         path,
	}
}

func getLinuxPlatform() string {
	file, err := os.Open("/etc/os-release")
	if err != nil {
		return "Linux"
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			prettyName := strings.TrimPrefix(line, "PRETTY_NAME=")
			prettyName = strings.Trim(prettyName, `"`+"'")
			return prettyName
		}
	}
	return "Linux"
}

func collectHost() HostInfo {
	var uptimeSec uint64
	data, err := os.ReadFile("/proc/uptime")
	if err == nil {
		parts := strings.Fields(string(data))
		if len(parts) >= 1 {
			if val, err := strconv.ParseFloat(parts[0], 64); err == nil {
				uptimeSec = uint64(val)
			}
		}
	}

	return HostInfo{
		OS:            "linux",
		Platform:      getLinuxPlatform(),
		UptimeSeconds: uptimeSec,
	}
}

// CollectStatus collects overall system status metrics on Linux
func CollectStatus(diskPath string) (SystemStatus, error) {
	return SystemStatus{
		CPU:  collectCPU(),
		RAM:  collectRAM(),
		Disk: collectDisk(diskPath),
		Host: collectHost(),
	}, nil
}
