//go:build windows

package systemstatus

import (
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

var (
	modkernel32              = syscall.NewLazyDLL("kernel32.dll")
	procGlobalMemoryStatusEx = modkernel32.NewProc("GlobalMemoryStatusEx")
	procGetSystemTimes       = modkernel32.NewProc("GetSystemTimes")
	procGetTickCount64       = modkernel32.NewProc("GetTickCount64")
	procGetDiskFreeSpaceExW  = modkernel32.NewProc("GetDiskFreeSpaceExW")
)

type memoryStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

type filetime struct {
	LowDateTime  uint32
	HighDateTime uint32
}

func (ft filetime) toUint64() uint64 {
	return (uint64(ft.HighDateTime) << 32) | uint64(ft.LowDateTime)
}

var (
	cpuMutex    sync.Mutex
	lastIdle    uint64
	lastKernel  uint64
	lastUser    uint64
	lastChecked time.Time
)

func collectCPU() CPUInfo {
	cpuMutex.Lock()
	defer cpuMutex.Unlock()

	var idle, kernel, user filetime
	r1, _, _ := procGetSystemTimes.Call(
		uintptr(unsafe.Pointer(&idle)),
		uintptr(unsafe.Pointer(&kernel)),
		uintptr(unsafe.Pointer(&user)),
	)
	if r1 == 0 {
		return CPUInfo{UsagePercent: 0, Cores: runtime.NumCPU()}
	}

	currIdle := idle.toUint64()
	currKernel := kernel.toUint64()
	currUser := user.toUint64()

	var usage float64 = 0
	if !lastChecked.IsZero() {
		idleDiff := currIdle - lastIdle
		kernelDiff := currKernel - lastKernel
		userDiff := currUser - lastUser
		total := kernelDiff + userDiff

		if total > 0 {
			// On Windows, kernel time includes idle time, so active is: total - idleDiff
			if total >= idleDiff {
				active := total - idleDiff
				usage = (float64(active) / float64(total)) * 100
			}
		}
	} else {
		// First call, initialize values and query again with a quick sleep to provide smooth data
		lastIdle = currIdle
		lastKernel = currKernel
		lastUser = currUser
		lastChecked = time.Now()
		cpuMutex.Unlock()
		time.Sleep(100 * time.Millisecond)
		cpuMutex.Lock()
		return collectCPU()
	}

	lastIdle = currIdle
	lastKernel = currKernel
	lastUser = currUser
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
	var ms memoryStatusEx
	ms.Length = uint32(unsafe.Sizeof(ms))
	r1, _, _ := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&ms)))
	if r1 == 0 {
		return RAMInfo{}
	}

	total := ms.TotalPhys
	used := ms.TotalPhys - ms.AvailPhys
	return RAMInfo{
		TotalBytes:   total,
		UsedBytes:    used,
		FreeBytes:    ms.AvailPhys,
		UsagePercent: float64(ms.MemoryLoad),
	}
}

func collectDisk(path string) DiskInfo {
	var freeBytes, totalBytes, totalFreeBytes uint64
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return DiskInfo{Path: path}
	}
	r1, _, _ := procGetDiskFreeSpaceExW.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeBytes)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFreeBytes)),
	)
	if r1 == 0 {
		return DiskInfo{Path: path}
	}

	used := totalBytes - freeBytes
	var percent float64
	if totalBytes > 0 {
		percent = (float64(used) / float64(totalBytes)) * 100
	}

	return DiskInfo{
		TotalBytes:   totalBytes,
		UsedBytes:    used,
		FreeBytes:    freeBytes,
		UsagePercent: percent,
		Path:         path,
	}
}

func collectHost() HostInfo {
	uptimeMs, _, _ := procGetTickCount64.Call()
	uptimeSec := uptimeMs / 1000

	return HostInfo{
		OS:            "windows",
		Platform:      "Windows Desktop/Server",
		UptimeSeconds: uint64(uptimeSec),
	}
}

// CollectStatus collects overall system status metrics on Windows
func CollectStatus(diskPath string) (SystemStatus, error) {
	return SystemStatus{
		CPU:  collectCPU(),
		RAM:  collectRAM(),
		Disk: collectDisk(diskPath),
		Host: collectHost(),
	}, nil
}
