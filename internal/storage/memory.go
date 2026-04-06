package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"penframe/internal/domain"
)

type StoredRun struct {
	ID        string            `json:"id"`
	ProjectID string            `json:"project_id,omitempty"`
	TargetID  string            `json:"target_id,omitempty"`
	Summary   domain.RunSummary `json:"summary"`
}

type MemoryStore struct {
	mu       sync.RWMutex
	filePath string
	runs     map[string]StoredRun
	order    []string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{runs: make(map[string]StoredRun)}
}

type persistedRuns struct {
	Runs []StoredRun `json:"runs"`
}

func NewFileStore(dataDir string) (*MemoryStore, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create run data dir: %w", err)
	}

	store := &MemoryStore{
		filePath: filepath.Join(dataDir, "runs.json"),
		runs:     make(map[string]StoredRun),
		order:    make([]string, 0),
	}
	if err := store.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("load runs: %w", err)
	}
	return store, nil
}

func (s *MemoryStore) Save(id string, summary domain.RunSummary) error {
	return s.SaveRun(StoredRun{
		ID:      id,
		Summary: summary,
	})
}

func (s *MemoryStore) SaveRun(run StoredRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.runs[run.ID]; !exists {
		s.order = append(s.order, run.ID)
	}
	s.runs[run.ID] = run
	return s.saveLocked()
}

func (s *MemoryStore) Get(id string) (domain.RunSummary, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	run, ok := s.runs[id]
	return run.Summary, ok
}

func (s *MemoryStore) GetStoredRun(id string) (StoredRun, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	summary, ok := s.runs[id]
	if !ok {
		return StoredRun{}, false
	}
	return summary, true
}

func (s *MemoryStore) Latest() (StoredRun, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.order) == 0 {
		return StoredRun{}, false
	}
	id := s.order[len(s.order)-1]
	return s.runs[id], true
}

func (s *MemoryStore) List(limit int) []StoredRun {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.listLocked(limit, "", "")
}

func (s *MemoryStore) ListByFilter(projectID, targetID string, limit int) []StoredRun {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.listLocked(limit, projectID, targetID)
}

func (s *MemoryStore) listLocked(limit int, projectID, targetID string) []StoredRun {
	if limit <= 0 || limit > len(s.order) {
		limit = len(s.order)
	}
	runs := make([]StoredRun, 0, limit)
	for idx := len(s.order) - 1; idx >= 0 && len(runs) < limit; idx-- {
		id := s.order[idx]
		run := s.runs[id]
		if projectID != "" && run.ProjectID != projectID {
			continue
		}
		if targetID != "" && run.TargetID != targetID {
			continue
		}
		runs = append(runs, run)
	}
	return runs
}

func (s *MemoryStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}

	var persisted persistedRuns
	if err := json.Unmarshal(data, &persisted); err != nil {
		return err
	}

	s.runs = make(map[string]StoredRun, len(persisted.Runs))
	s.order = make([]string, 0, len(persisted.Runs))
	for _, run := range persisted.Runs {
		if _, exists := s.runs[run.ID]; !exists {
			s.order = append(s.order, run.ID)
		}
		s.runs[run.ID] = run
	}
	return nil
}

func (s *MemoryStore) saveLocked() error {
	if s.filePath == "" {
		return nil
	}

	persisted := persistedRuns{
		Runs: make([]StoredRun, 0, len(s.order)),
	}
	for _, id := range s.order {
		run, ok := s.runs[id]
		if !ok {
			continue
		}
		persisted.Runs = append(persisted.Runs, run)
	}

	data, err := json.MarshalIndent(persisted, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := s.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, s.filePath)
}
