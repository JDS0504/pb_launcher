package models

import "regexp"

type ServiceStatus string
type RestartPolicy string

const (
	Idle      ServiceStatus = "idle"    // Created but never started
	Running   ServiceStatus = "running" // Active and running
	Stopped   ServiceStatus = "stopped" // Stopped manually
	Failure   ServiceStatus = "failure" // Stopped manually
	Restoring ServiceStatus = "restoring"
	Sleeping  ServiceStatus = "sleeping" // Suspendida temporalmente por inactividad
)

const (
	OnFailure RestartPolicy = "on-failure" // Restart only on errors (start-failed, unexpected-exit)
	Never     RestartPolicy = "no"         // Never restart automatically
)

type Service struct {
	ID            string
	Name          string
	Status        ServiceStatus
	RestartPolicy RestartPolicy
	//
	ReleaseID       string
	Version         string
	ExecFilePattern *regexp.Regexp
	//
	BootPBInstallPath string
	BootUserEmail     string
	BootUserPassword  string
	IP                string
	Port              string
	CpuQuota          string
	MemoryLimit       string
}

type Release struct {
	ID           string
	Version      string
}
