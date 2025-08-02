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

type StreamEvent struct {
	Type      string
	Payload   interface{}
	Timestamp time.Time
	Origin    string
}
