package models

import "regexp"

type ServiceStatus string
type RestartPolicy string

const (
	Idle    ServiceStatus = "idle"    // Created but never started
	Running ServiceStatus = "running" // Active and running
	Stopped ServiceStatus = "stopped" // Stopped manually
	Failure ServiceStatus = "failure" // Stopped manually
)

const (
	OnFailure RestartPolicy = "on-failure" // Restart only on errors (start-failed, unexpected-exit)
	Never     RestartPolicy = "no"         // Never restart automatically
)

type Service struct {
	ID            string
	Status        ServiceStatus
	RestartPolicy RestartPolicy
	//
	RepositoryID    string
	Version         string
	ExecFilePattern *regexp.Regexp
	//
	BootPBInstallPath string
	BootUserEmail     string
	BootUserPassword  string
	Deleted           string
}

type Release struct {
	ID           string
	RepositoryID string
	Version      string
}
