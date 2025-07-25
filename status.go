package inforo

import (
	"sync"
	"time"

	"github.com/laplasd/inforo/model"
)

type StatusManager struct {
}

func (s *StatusManager) NewStatus(status model.Status) *model.StatusHistory {
	return &model.StatusHistory{
		LastStatus: status,
		Timestamp:  time.Now(),
		MU:         sync.RWMutex{},
	}
}

func (s *StatusManager) NextStatus(status model.Status, history *model.StatusHistory) *model.StatusHistory {
	history.MU.Lock()
	defer history.MU.Unlock()
	prev := &struct {
		Status    model.Status `json:"Status"`
		Timestamp time.Time    `json:"Timestamp"`
	}{
		Status:    history.LastStatus,
		Timestamp: history.Timestamp,
	}

	var newHistory model.StatusHistory
	newHistory.LastStatus = status
	newHistory.Timestamp = time.Now()

	// добавляем всю предыдущую историю, если есть
	if history.Previous != nil {
		newHistory.Previous = append([]*struct {
			Status    model.Status `json:"Status"`
			Timestamp time.Time    `json:"Timestamp"`
		}{prev}, history.Previous...)
	} else {
		newHistory.Previous = []*struct {
			Status    model.Status `json:"Status"`
			Timestamp time.Time    `json:"Timestamp"`
		}{prev}
	}

	return &newHistory
}
