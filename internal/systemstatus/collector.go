package systemstatus

import "pb_launcher/utils/processstats"

// CPUInfo represents CPU utilization and core count
type CPUInfo struct {
	UsagePercent float64 `json:"usage_percent"`
	Cores        int     `json:"cores"`
}

// RAMInfo represents memory allocation details
type RAMInfo struct {
	TotalBytes   uint64  `json:"total_bytes"`
	UsedBytes    uint64  `json:"used_bytes"`
	FreeBytes    uint64  `json:"free_bytes"`
	UsagePercent float64 `json:"usage_percent"`
}

// DiskInfo represents filesystem storage usage
type DiskInfo struct {
	TotalBytes   uint64  `json:"total_bytes"`
	UsedBytes    uint64  `json:"used_bytes"`
	FreeBytes    uint64  `json:"free_bytes"`
	UsagePercent float64 `json:"usage_percent"`
	Path         string  `json:"path"`
}

// HostInfo represents metadata about the server host
type HostInfo struct {
	OS              string `json:"os"`
	Platform        string `json:"platform"`
	UptimeSeconds   uint64 `json:"uptime_seconds"`
	ActiveInstances int    `json:"active_instances"`
}

// InstanceStats representa el consumo real de recursos de una instancia en ejecución
type InstanceStats = processstats.InstanceStats

// SystemStatus is the unified metrics payload
type SystemStatus struct {
	CPU            CPUInfo                  `json:"cpu"`
	RAM            RAMInfo                  `json:"ram"`
	Disk           DiskInfo                 `json:"disk"`
	Host           HostInfo                 `json:"host"`
	InstancesStats map[string]InstanceStats `json:"instances_stats"`
}
