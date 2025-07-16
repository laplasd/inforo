package model

import (
	"sync"
	"time"
)

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

type StatusHistory struct {
	LastStatus Status    `json:"LastStatus"`
	Timestamp  time.Time `json:"Timestamp"`
	Previous   []*struct {
		Status    Status    `json:"Status"`
		Timestamp time.Time `json:"Timestamp"`
	} `json:"history,omitempty"`
	MU sync.RWMutex `json:"-"`
}
