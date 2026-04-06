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

type Target struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	Name        string    `json:"name"`
	URL         string    `json:"url"`
	CreatedAt   time.Time `json:"created_at"`
	LastScanned time.Time `json:"last_scanned,omitempty"`
}

type Project struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	Targets   []Target  `json:"targets"`
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

func (s *Store) Add(name string) (Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p := Project{
		ID:        newID(),
		Name:      name,
		CreatedAt: time.Now(),
		Targets:   []Target{},
	}
	s.projects = append(s.projects, p)

	if err := s.save(); err != nil {
		// rollback memory on save error
		s.projects = s.projects[:len(s.projects)-1]
		return Project{}, err
	}
	return p, nil
}

func (s *Store) Get(id string) (Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, p := range s.projects {
		if p.ID == id {
			return p, nil
		}
	}
	return Project{}, fmt.Errorf("project not found: %s", id)
}

func (s *Store) AddTarget(projectID, name, url string) (Target, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := -1
	for i, p := range s.projects {
		if p.ID == projectID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return Target{}, fmt.Errorf("project not found: %s", projectID)
	}

	t := Target{
		ID:        newID(),
		ProjectID: projectID,
		Name:      name,
		URL:       url,
		CreatedAt: time.Now(),
	}
	s.projects[idx].Targets = append(s.projects[idx].Targets, t)

	if err := s.save(); err != nil {
		// rollback
		s.projects[idx].Targets = s.projects[idx].Targets[:len(s.projects[idx].Targets)-1]
		return Target{}, err
	}
	return t, nil
}

func (s *Store) GetTarget(targetID string) (Target, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, p := range s.projects {
		for _, t := range p.Targets {
			if t.ID == targetID {
				return t, nil
			}
		}
	}
	return Target{}, fmt.Errorf("target not found: %s", targetID)
}

func (s *Store) UpdateTarget(targetID string, name, url string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, p := range s.projects {
		for j, t := range p.Targets {
			if t.ID == targetID {
				if name != "" {
					s.projects[i].Targets[j].Name = name
				}
				if url != "" {
					s.projects[i].Targets[j].URL = url
				}
				return s.save()
			}
		}
	}
	return fmt.Errorf("target not found: %s", targetID)
}

func (s *Store) UpdateTargetLastScanned(targetID string, t time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, p := range s.projects {
		for j, target := range p.Targets {
			if target.ID == targetID {
				s.projects[i].Targets[j].LastScanned = t
				return s.save()
			}
		}
	}
	return fmt.Errorf("target not found: %s", targetID)
}

func (s *Store) DeleteTarget(projectID, targetID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, p := range s.projects {
		if p.ID == projectID {
			idx := -1
			for j, t := range p.Targets {
				if t.ID == targetID {
					idx = j
					break
				}
			}
			if idx == -1 {
				return fmt.Errorf("target not found: %s", targetID)
			}

			backup := make([]Target, len(s.projects[i].Targets))
			copy(backup, s.projects[i].Targets)

			s.projects[i].Targets = append(s.projects[i].Targets[:idx], s.projects[i].Targets[idx+1:]...)

			if err := s.save(); err != nil {
				s.projects[i].Targets = backup
				return err
			}
			return nil
		}
	}
	return fmt.Errorf("project not found: %s", projectID)
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
