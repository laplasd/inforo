package inforo

import (
	"time"

	"github.com/laplasd/inforo/model"
)

type Events struct {
}

func (e *Events) AddEvent(events *model.EventHistory, message string) {

	// Инициализируем Event если он nil
	if events == nil {
		events = &model.EventHistory{}
		events.Event = make([]model.Event, 0)
	}

	events.Event = append(events.Event, model.Event{
		Timestamp: time.Now(),
		Message:   message,
	})
}
