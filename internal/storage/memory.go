package storage

import (
	"sync"

	"penframe/internal/domain"
)

type StoredRun struct {
	ID      string            `json:"id"`
	Summary domain.RunSummary `json:"summary"`
}

type MemoryStore struct {
	mu    sync.RWMutex
	runs  map[string]domain.RunSummary
	order []string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{runs: make(map[string]domain.RunSummary)}
}

func (s *MemoryStore) Save(id string, summary domain.RunSummary) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.runs[id]; !exists {
		s.order = append(s.order, id)
	}
	s.runs[id] = summary
}

func (s *MemoryStore) Get(id string) (domain.RunSummary, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	summary, ok := s.runs[id]
	return summary, ok
}

func (s *MemoryStore) GetStoredRun(id string) (StoredRun, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	summary, ok := s.runs[id]
	if !ok {
		return StoredRun{}, false
	}
	return StoredRun{
		ID:      id,
		Summary: summary,
	}, true
}

func (s *MemoryStore) Latest() (StoredRun, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.order) == 0 {
		return StoredRun{}, false
	}
	id := s.order[len(s.order)-1]
	return StoredRun{
		ID:      id,
		Summary: s.runs[id],
	}, true
}

func (s *MemoryStore) List(limit int) []StoredRun {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > len(s.order) {
		limit = len(s.order)
	}
	runs := make([]StoredRun, 0, limit)
	for idx := len(s.order) - 1; idx >= 0 && len(runs) < limit; idx-- {
		id := s.order[idx]
		runs = append(runs, StoredRun{
			ID:      id,
			Summary: s.runs[id],
		})
	}
	return runs
}
