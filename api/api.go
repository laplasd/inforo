package api

import "github.com/laplasd/inforo/model"

type StatusProvider interface {
	AddEvent(events *model.EventHistory, message string)
	NewStatus(status model.Status) *model.StatusHistory
	NextStatus(status model.Status, history model.StatusHistory) *model.StatusHistory
}
