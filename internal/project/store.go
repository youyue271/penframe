package project

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Project struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
}

type Store struct {
	mu       sync.RWMutex
	filePath string
	projects []Project
}

func NewStore(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create project data dir: %w", err)
	}
	filePath := filepath.Join(dataDir, "projects.json")
	store := &Store{
		filePath: filePath,
		projects: make([]Project, 0),
	}
	if err := store.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("load projects: %w", err)
	}
	return store, nil
}

func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &s.projects)
}

func (s *Store) save() error {
	data, err := json.MarshalIndent(s.projects, "", "  ")
	if err != nil {
		return err
	}
	tmpPath := s.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, s.filePath)
}

func (s *Store) List() []Project {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make([]Project, len(s.projects))
	copy(res, s.projects)
	return res
}

func (s *Store) Add(name, url string) (Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p := Project{
		ID:        newID(),
		Name:      name,
		URL:       url,
		CreatedAt: time.Now(),
	}
	s.projects = append(s.projects, p)

	if err := s.save(); err != nil {
		// rollback memory on save error
		s.projects = s.projects[:len(s.projects)-1]
		return Project{}, err
	}
	return p, nil
}

func newID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("proj-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := -1
	for i, p := range s.projects {
		if p.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("project not found: %s", id)
	}

	backup := make([]Project, len(s.projects))
	copy(backup, s.projects)

	s.projects = append(s.projects[:idx], s.projects[idx+1:]...)

	if err := s.save(); err != nil {
		s.projects = backup
		return err
	}
	return nil
}
