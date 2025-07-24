package model

import (
	"sync"
	"time"
)

type Event struct {
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
}

type EventHistory struct {
	MU    sync.RWMutex `json:"-"`
	Event []Event
}
