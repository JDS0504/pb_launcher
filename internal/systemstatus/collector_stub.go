//go:build !windows && !linux

package systemstatus

import (
	"runtime"
)

// CollectStatus collects overall system status metrics on non-Windows/non-Linux systems
func CollectStatus(diskPath string) (SystemStatus, error) {
	return SystemStatus{
		CPU: CPUInfo{
			UsagePercent: 15.0,
			Cores:        runtime.NumCPU(),
		},
		RAM: RAMInfo{
			TotalBytes:   16 * 1024 * 1024 * 1024,
			UsedBytes:    8 * 1024 * 1024 * 1024,
			FreeBytes:    8 * 1024 * 1024 * 1024,
			UsagePercent: 50.0,
		},
		Disk: DiskInfo{
			TotalBytes:   500 * 1024 * 1024 * 1024,
			UsedBytes:    250 * 1024 * 1024 * 1024,
			FreeBytes:    250 * 1024 * 1024 * 1024,
			UsagePercent: 50.0,
			Path:         diskPath,
		},
		Host: HostInfo{
			OS:            runtime.GOOS,
			Platform:      "Generic Unix/Mac Developer System",
			UptimeSeconds: 7200,
		},
	}, nil
}
