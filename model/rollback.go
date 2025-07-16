package model

type RollBackType string

const (
	ManualRollBack  RollBackType = "manual"
	TriggerRollBack RollBackType = "trigger"
)

type Rollback struct {
	Type       RollBackType      `json:"Type"`
	ID         string            `json:"ID,omitempty"`
	TaskID     string            `json:"TaskID,omitempty"`
	PlanID     string            `json:"PlanID,omitempty"`
	Components []string          `json:"Components,omitempty"`
	Metadata   map[string]string `json:"MetaData,omitempty"`
}
