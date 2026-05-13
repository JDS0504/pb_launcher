package models

type CommandAction int

const (
	ActionStop CommandAction = iota + 1
	ActionStart
	ActionRestart
	ActionUpgrade
)

type ServiceCommand struct {
	ID            string        `json:"id"`
	Service       string        `json:"service"`
	Action        CommandAction `json:"action"`
	TargetRelease string        `json:"target_release"`
}
