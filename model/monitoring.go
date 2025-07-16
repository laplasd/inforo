package model

import "sync"

type Status string

type Monitoring struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Type          string            `json:"type"` // URL, сокет, агент и т.п.
	StatusHistory *StatusHistory    `json:"status_history,omitempty"`
	EventHistory  *EventHistory     `json:"event_history,omitempty"`
	Config        map[string]string `json:"config,omitempty"` // дополнительные параметры
	MU            sync.RWMutex      `json:"-"`
}

type Check struct {
	ID            string            `json:"ID"`
	Name          string            `json:"Name"`
	MonitoringID  string            `json:"MonitoringID"`
	StatusHistory *StatusHistory    `json:"StatusHistory,omitempty"`
	EventHistory  *EventHistory     `json:"EventHistory,omitempty"`
	Metadata      map[string]string `json:"MetaData"`
	MU            sync.RWMutex      `json:"-"`
}
