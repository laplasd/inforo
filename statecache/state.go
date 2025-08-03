package statecache

import (
	"sync"
	"time"

	"github.com/laplasd/inforo/model"
)

// StateCache поддерживает актуальное состояние компонентов путем применения дельт
type StateCache struct {
	mu          sync.RWMutex
	components  map[string]*ComponentState
	deltaLog    map[string][]model.Delta // Лог дельт для каждого компонента
	snapshotter *Snapshotter
}

type ComponentState struct {
	ID         string
	Version    uint64      // Версия состояния (инкрементируется с каждой дельтой)
	State      interface{} // Текущее состояние
	Deleted    bool        // Флаг логического удаления
	LastUpdate time.Time   // Временная метка последнего обновления
}

func NewStateCache(snapshotInterval time.Duration) *StateCache {
	sc := &StateCache{
		components: make(map[string]*ComponentState),
		deltaLog:   make(map[string][]model.Delta),
	}

	// Периодическое создание снапшотов
	sc.snapshotter = NewSnapshotter(sc, snapshotInterval)
	go sc.snapshotter.Run()

	return sc
}

// Put добавляет новое состояние компонента (для Create)
func (sc *StateCache) Put(id string, state interface{}) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	sc.components[id] = &ComponentState{
		ID:         id,
		Version:    1,
		State:      state,
		LastUpdate: time.Now(),
	}

	// Для новых компонентов инициализируем лог дельт
	sc.deltaLog[id] = []model.Delta{}
}

// Get возвращает текущее состояние компонента
func (sc *StateCache) Get(id string) (interface{}, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	if state, exists := sc.components[id]; exists && !state.Deleted {
		return state.State, true
	}
	return nil, false
}

// ApplyPatch применяет изменения к состоянию (для Update)
func (sc *StateCache) ApplyPatch(id string, patch map[string]interface{}) interface{} {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	state, exists := sc.components[id]
	if !exists || state.Deleted {
		return nil
	}

	// Применяем изменения к состоянию
	currentState := state.State.(map[string]interface{})
	newState := applyPatch(currentState, patch)

	// Обновляем версию и метку времени
	sc.components[id] = &ComponentState{
		ID:         id,
		Version:    state.Version + 1,
		State:      newState,
		LastUpdate: time.Now(),
	}

	return newState
}

func applyPatch(current, patch map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Копируем текущее состояние
	for k, v := range current {
		result[k] = v
	}

	// Применяем изменения
	for k, v := range patch {
		if v == nil {
			delete(result, k) // Удаление поля
		} else {
			result[k] = v // Обновление/добавление поля
		}
	}

	return result
}

// MarkDeleted помечает компонент как удаленный (для Delete)
func (sc *StateCache) MarkDeleted(id string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if state, exists := sc.components[id]; exists {
		sc.components[id] = &ComponentState{
			ID:         id,
			Version:    state.Version + 1,
			State:      state.State,
			Deleted:    true,
			LastUpdate: time.Now(),
		}
	}
}

// AddDelta добавляет дельту в лог и обновляет состояние
func (sc *StateCache) AddDelta(delta model.Delta) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	id := delta.ID

	// Добавляем дельту в лог
	sc.deltaLog[id] = append(sc.deltaLog[id], delta)

	// Применяем дельту к состоянию
	switch delta.Type {
	case model.DeltaTypeCreate:
		sc.components[id] = &ComponentState{
			ID:         id,
			Version:    1,
			State:      delta.State,
			LastUpdate: delta.Timestamp,
		}

	case model.DeltaTypeUpdate:
		if state, exists := sc.components[id]; exists && !state.Deleted {
			newState := applyPatch(state.State.(map[string]interface{}), delta.State)
			sc.components[id] = &ComponentState{
				ID:         id,
				Version:    state.Version + 1,
				State:      newState,
				LastUpdate: delta.Timestamp,
			}
		}

	case model.DeltaTypeDelete:
		if state, exists := sc.components[id]; exists {
			sc.components[id] = &ComponentState{
				ID:         id,
				Version:    state.Version + 1,
				State:      state.State,
				Deleted:    true,
				LastUpdate: delta.Timestamp,
			}
		}
	}
}

// RebuildState восстанавливает состояние из лога дельт
func (sc *StateCache) RebuildState(id string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	deltas := sc.deltaLog[id]
	if len(deltas) == 0 {
		delete(sc.components, id)
		return
	}

	var state *ComponentState

	for _, delta := range deltas {
		switch delta.Type {
		case model.DeltaTypeCreate:
			state = &ComponentState{
				ID:         id,
				Version:    1,
				State:      delta.State,
				LastUpdate: delta.Timestamp,
			}

		case model.DeltaTypeUpdate:
			if state != nil && !state.Deleted {
				state.State = applyPatch(state.State.(map[string]interface{}), delta.State)
				state.Version++
				state.LastUpdate = delta.Timestamp
			}

		case model.DeltaTypeDelete:
			if state != nil {
				state.Deleted = true
				state.Version++
				state.LastUpdate = delta.Timestamp
			}
		}
	}

	if state != nil {
		sc.components[id] = state
	}
}

// Snapshotter периодически создает снапшоты состояний
type Snapshotter struct {
	cache    *StateCache
	interval time.Duration
	stopChan chan struct{}
}

func NewSnapshotter(cache *StateCache, interval time.Duration) *Snapshotter {
	return &Snapshotter{
		cache:    cache,
		interval: interval,
		stopChan: make(chan struct{}),
	}
}

func (s *Snapshotter) Run() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.takeSnapshot()
		case <-s.stopChan:
			return
		}
	}
}

func (s *Snapshotter) takeSnapshot() {
	s.cache.mu.Lock()
	defer s.cache.mu.Unlock()

	// Оптимизация: сохраняем только активные компоненты
	for id, state := range s.cache.components {
		if !state.Deleted {
			// Здесь должна быть реальная логика сохранения снапшота
			// Например: saveToStorage(id, state)
		}

		// Очищаем лог дельт для оптимизации памяти
		if len(s.cache.deltaLog[id]) > 100 { // Условный порог
			s.cache.deltaLog[id] = s.cache.deltaLog[id][len(s.cache.deltaLog[id])-50:] // Сохраняем последние 50 дельт
		}
	}
}
