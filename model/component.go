package model

type Component struct {
	ID            string            `json:"ID"`
	Name          string            `json:"Name"`
	Type          string            `json:"Type"`
	Version       string            `json:"Version"`
	StatusHistory *StatusHistory    `json:"StatusHistory,omitempty"`
	EventHistory  *EventHistory     `json:"EventHistory,omitempty"`
	Metadata      map[string]string `json:"MetaData,omitempty"`
}
