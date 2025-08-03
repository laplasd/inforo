package crdt

import (
	"fmt"
	"sync"

	"github.com/laplasd/inforo/model"
)

type CRDTStore struct {
	data       map[string][]model.Delta
	tombstones map[string]bool
	mu         sync.RWMutex
}

var ErrNotFound = fmt.Errorf("crdt: not found")

func (s *CRDTStore) Merge(delta model.Delta) (model.Delta, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Конфликтное слияние по LWW (Last-Write-Wins)
	existing := s.data[delta.ID]
	if len(existing) > 0 {
		last := existing[len(existing)-1]
		if last.Timestamp.After(delta.Timestamp) {
			return last, nil // Игнорируем старую дельту
		}
	}

	s.data[delta.ID] = append(existing, delta)
	return delta, nil
}

func (s *CRDTStore) GetLatest(id string) (model.Delta, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.tombstones[id] {
		return model.Delta{}, ErrNotFound
	}

	deltas := s.data[id]
	if len(deltas) == 0 {
		return model.Delta{}, ErrNotFound
	}

	return deltas[len(deltas)-1], nil
}

func (s *CRDTStore) GetAll(id string) ([]model.Delta, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.tombstones[id] {
		return nil, ErrNotFound
	}

	deltas := s.data[id]
	if len(deltas) == 0 {
		return nil, ErrNotFound
	}
	return deltas, nil
}
