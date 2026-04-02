package asset

import (
	"sync"

	"penframe/internal/domain"
)

// Store is a thread-safe container for asset graphs keyed by run ID.
type Store struct {
	mu     sync.RWMutex
	graphs map[string]*Graph
	tasks  map[string]*domain.ScanTask
	order  []string
}

// NewStore creates an empty asset store.
func NewStore() *Store {
	return &Store{
		graphs: make(map[string]*Graph),
		tasks:  make(map[string]*domain.ScanTask),
	}
}

// GetOrCreate returns the graph for a run, creating one if needed.
func (s *Store) GetOrCreate(runID, target string) *Graph {
	s.mu.Lock()
	defer s.mu.Unlock()
	if g, ok := s.graphs[runID]; ok {
		return g
	}
	g := NewGraph(target)
	s.graphs[runID] = g
	s.order = append(s.order, runID)
	return g
}

// Get returns the graph for a run.
func (s *Store) Get(runID string) (*Graph, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	g, ok := s.graphs[runID]
	return g, ok
}

// Latest returns the most recently created graph.
func (s *Store) Latest() (*Graph, string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.order) == 0 {
		return nil, "", false
	}
	id := s.order[len(s.order)-1]
	return s.graphs[id], id, true
}

// AddTask registers a scan task.
func (s *Store) AddTask(task *domain.ScanTask) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[task.ID] = task
}

// UpdateTaskStatus updates a task's status.
func (s *Store) UpdateTaskStatus(taskID, status, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t, ok := s.tasks[taskID]; ok {
		t.Status = status
		if errMsg != "" {
			t.Error = errMsg
		}
	}
}

// ListTasks returns all scan tasks.
func (s *Store) ListTasks() []*domain.ScanTask {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tasks := make([]*domain.ScanTask, 0, len(s.tasks))
	for _, t := range s.tasks {
		tasks = append(tasks, t)
	}
	return tasks
}

// ListTasksByRun returns scan tasks filtered by run (parent) ID prefix.
func (s *Store) ListTasksByRun(runID string) []*domain.ScanTask {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var tasks []*domain.ScanTask
	for _, t := range s.tasks {
		if t.ParentID == runID {
			tasks = append(tasks, t)
		}
	}
	return tasks
}

// UpdateTasksByRun updates all tasks for a run.
func (s *Store) UpdateTasksByRun(runID, status, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.tasks {
		if t.ParentID != runID {
			continue
		}
		t.Status = status
		if errMsg != "" {
			t.Error = errMsg
		}
	}
}

// UpdatePendingTasksByRun updates only pending tasks for a run.
func (s *Store) UpdatePendingTasksByRun(runID, status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.tasks {
		if t.ParentID != runID || t.Status != domain.ScanTaskPending {
			continue
		}
		t.Status = status
	}
}

// FinalizeTasksByRun marks unfinished tasks for a run with the final status.
func (s *Store) FinalizeTasksByRun(runID, status, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.tasks {
		if t.ParentID != runID {
			continue
		}
		if t.Status == domain.ScanTaskFailed || t.Status == domain.ScanTaskDone || t.Status == domain.ScanTaskSkipped {
			continue
		}
		t.Status = status
		if errMsg != "" {
			t.Error = errMsg
		}
	}
}
