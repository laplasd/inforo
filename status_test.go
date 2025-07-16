package inforo

import (
	"sync"
	"testing"
	"time"

	"github.com/laplasd/inforo/model"
	"github.com/stretchr/testify/assert"
)

func TestStatusManager_NewStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   model.Status
		expected *model.StatusHistory
	}{
		{
			name:   "Pending status",
			status: model.StatusPending,
			expected: &model.StatusHistory{
				LastStatus: model.StatusPending,
				Timestamp:  time.Now(), // будет проверено приблизительно
				MU:         sync.RWMutex{},
			},
		},
		{
			name:   "Running status",
			status: model.StatusRunning,
			expected: &model.StatusHistory{
				LastStatus: model.StatusRunning,
				Timestamp:  time.Now(),
				MU:         sync.RWMutex{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &StatusManager{}
			before := time.Now()
			result := sm.NewStatus(tt.status)
			after := time.Now()

			assert.Equal(t, tt.status, result.LastStatus)
			assert.True(t, result.Timestamp.After(before) || result.Timestamp.Equal(before))
			assert.True(t, result.Timestamp.Before(after) || result.Timestamp.Equal(after))
			assert.NotNil(t, result.MU)
		})
	}
}

func TestStatusManager_NextStatus(t *testing.T) {
	now := time.Now()
	previousTime := now.Add(-5 * time.Minute)

	tests := []struct {
		name          string
		newStatus     model.Status
		initialStatus *model.StatusHistory
		expected      *model.StatusHistory
	}{
		{
			name:      "From Pending to Running",
			newStatus: model.StatusRunning,
			initialStatus: &model.StatusHistory{
				LastStatus: model.StatusPending,
				Timestamp:  previousTime,
				MU:         sync.RWMutex{},
			},
			expected: &model.StatusHistory{
				LastStatus: model.StatusRunning,
				Timestamp:  time.Now(), // будет проверено приблизительно
				Previous: []*struct {
					Status    model.Status `json:"Status"`
					Timestamp time.Time    `json:"Timestamp"`
				}{
					{
						Status:    model.StatusPending,
						Timestamp: previousTime,
					},
				},
			},
		},
		{
			name:      "With existing history",
			newStatus: model.StatusFailed,
			initialStatus: &model.StatusHistory{
				LastStatus: model.StatusRunning,
				Timestamp:  previousTime,
				Previous: []*struct {
					Status    model.Status `json:"Status"`
					Timestamp time.Time    `json:"Timestamp"`
				}{
					{
						Status:    model.StatusPending,
						Timestamp: previousTime.Add(-10 * time.Minute),
					},
				},
				MU: sync.RWMutex{},
			},
			expected: &model.StatusHistory{
				LastStatus: model.StatusFailed,
				Timestamp:  time.Now(),
				Previous: []*struct {
					Status    model.Status `json:"Status"`
					Timestamp time.Time    `json:"Timestamp"`
				}{
					{
						Status:    model.StatusRunning,
						Timestamp: previousTime,
					},
					{
						Status:    model.StatusPending,
						Timestamp: previousTime.Add(-10 * time.Minute),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &StatusManager{}
			before := time.Now()
			result := sm.NextStatus(tt.newStatus, tt.initialStatus)
			after := time.Now()

			// Проверка нового статуса и времени
			assert.Equal(t, tt.newStatus, result.LastStatus)
			assert.True(t, result.Timestamp.After(before) || result.Timestamp.Equal(before))
			assert.True(t, result.Timestamp.Before(after) || result.Timestamp.Equal(after))

			// Проверка истории предыдущих статусов
			assert.Equal(t, len(tt.expected.Previous), len(result.Previous))
			for i, prev := range tt.expected.Previous {
				assert.Equal(t, prev.Status, result.Previous[i].Status)
				assert.Equal(t, prev.Timestamp, result.Previous[i].Timestamp)
			}

			// Проверка что мьютекс создан
			assert.NotNil(t, result.MU)
		})
	}
}

func TestStatusManager_ConcurrentAccess(t *testing.T) {
	sm := &StatusManager{}
	initial := sm.NewStatus(model.StatusPending)

	// Запускаем несколько горутин для проверки конкурентного доступа
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result := sm.NextStatus(model.StatusRunning, initial)
			assert.Equal(t, model.StatusRunning, result.LastStatus)
		}()
	}
	wg.Wait()

	// Проверяем что история не повредилась
	result := sm.NextStatus(model.StatusFailed, initial)
	assert.Equal(t, model.StatusFailed, result.LastStatus)
	assert.Equal(t, 1, len(result.Previous))
	assert.Equal(t, model.StatusPending, result.Previous[0].Status)
}
