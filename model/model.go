package model

import (
	"sync"
	"time"
)

type StatusHistory struct {
	LastStatus Status    `json:"LastStatus"`
	Timestamp  time.Time `json:"Timestamp"`
	Previous   []*struct {
		Status    Status    `json:"Status"`
		Timestamp time.Time `json:"Timestamp"`
	} `json:"history,omitempty"`
}

type Status string

const (
	StatusCreated  Status = "created"
	StatusPending  Status = "pending"
	StatusCheck    Status = "checking"
	StatusRunning  Status = "running"
	StatusSuccess  Status = "success"
	StatusFailed   Status = "failed"
	StatusSkipped  Status = "skipped"
	StatusStopped  Status = "stopped"
	StatusPaused   Status = "paused"
	StatusDeferred Status = "deferred"
	StatusRetry    Status = "retry"
	StatusDisable  Status = "disable"
	StatusRollBack Status = "rollback"
)

type Monitoring struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Type          string            `json:"type"` // URL, сокет, агент и т.п.
	StatusHistory *StatusHistory    `json:"status_history,omitempty"`
	EventHistory  *EventHistory     `json:"event_history,omitempty"`
	Config        map[string]string `json:"config,omitempty"` // дополнительные параметры
}

type Check struct {
	ID            string            `json:"ID"`
	Name          string            `json:"Name"`
	MonitoringID  string            `json:"MonitoringID"`
	StatusHistory *StatusHistory    `json:"StatusHistory,omitempty"`
	EventHistory  *EventHistory     `json:"EventHistory,omitempty"`
	Metadata      map[string]string `json:"MetaData"`
}

type Event struct {
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
}

type EventHistory struct {
	MU    sync.Mutex `json:"-"`
	Event []Event
}
