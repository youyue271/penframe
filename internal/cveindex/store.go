package cveindex

import (
	"encoding/json"
	"os"
	"sync"
)

// Store manages the tested CVE set, persisted as a JSON array.
type Store struct {
	mu       sync.RWMutex
	filePath string
	tested   map[string]bool
}

func NewStore(filePath string) (*Store, error) {
	s := &Store{
		filePath: filePath,
		tested:   make(map[string]bool),
	}
	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	var ids []string
	if err := json.Unmarshal(data, &ids); err != nil {
		return err
	}
	for _, id := range ids {
		s.tested[id] = true
	}
	return nil
}

func (s *Store) save() error {
	ids := make([]string, 0, len(s.tested))
	for id := range s.tested {
		ids = append(ids, id)
	}
	data, err := json.MarshalIndent(ids, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.filePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, s.filePath)
}

func (s *Store) IsTested(cveID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.tested[cveID]
}

func (s *Store) SetTested(cveID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tested[cveID] = true
	return s.save()
}

func (s *Store) SetUntested(cveID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tested, cveID)
	return s.save()
}

func (s *Store) TestedSet() map[string]bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := make(map[string]bool, len(s.tested))
	for k, v := range s.tested {
		cp[k] = v
	}
	return cp
}
