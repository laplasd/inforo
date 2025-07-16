package core

import (
	"sync"
	"time"

	"github.com/laplasd/inforo/model"
)

type Events struct {
	mu sync.Mutex
}

func (e *Events) AddEvent(events *model.EventHistory, message string) {
	if events == nil {
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Инициализируем Event если он nil
	if events.Event == nil {
		events.Event = make([]model.Event, 0)
	}

	events.Event = append(events.Event, model.Event{
		Timestamp: time.Now(),
		Message:   message,
	})
}
