package storage

import (
	"testing"
	"time"

	"penframe/internal/domain"
)

func TestMemoryStoreLatestAndList(t *testing.T) {
	store := NewMemoryStore()
	first := domain.RunSummary{Workflow: "wf-1", StartedAt: time.Unix(1, 0).UTC()}
	second := domain.RunSummary{Workflow: "wf-2", StartedAt: time.Unix(2, 0).UTC()}

	if err := store.Save("run-1", first); err != nil {
		t.Fatalf("save first run: %v", err)
	}
	if err := store.Save("run-2", second); err != nil {
		t.Fatalf("save second run: %v", err)
	}

	latest, ok := store.Latest()
	if !ok {
		t.Fatal("expected latest run to exist")
	}
	if latest.ID != "run-2" {
		t.Fatalf("expected latest id run-2, got %q", latest.ID)
	}
	if latest.Summary.Workflow != "wf-2" {
		t.Fatalf("expected latest workflow wf-2, got %q", latest.Summary.Workflow)
	}

	runs := store.List(1)
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}
	if runs[0].ID != "run-2" {
		t.Fatalf("expected first listed run run-2, got %q", runs[0].ID)
	}

	stored, ok := store.GetStoredRun("run-1")
	if !ok {
		t.Fatal("expected stored run run-1 to exist")
	}
	if stored.Summary.Workflow != "wf-1" {
		t.Fatalf("expected stored run workflow wf-1, got %q", stored.Summary.Workflow)
	}
}

func TestFileStorePersistsAndFiltersRuns(t *testing.T) {
	dir := t.TempDir()

	store, err := NewFileStore(dir)
	if err != nil {
		t.Fatalf("new file store: %v", err)
	}

	first := domain.RunSummary{Workflow: "wf-1", StartedAt: time.Unix(1, 0).UTC()}
	second := domain.RunSummary{Workflow: "wf-2", StartedAt: time.Unix(2, 0).UTC()}

	if err := store.SaveRun(StoredRun{
		ID:        "run-1",
		ProjectID: "project-1",
		TargetID:  "target-1",
		Summary:   first,
	}); err != nil {
		t.Fatalf("save run-1: %v", err)
	}
	if err := store.SaveRun(StoredRun{
		ID:        "run-2",
		ProjectID: "project-1",
		TargetID:  "target-2",
		Summary:   second,
	}); err != nil {
		t.Fatalf("save run-2: %v", err)
	}

	reloaded, err := NewFileStore(dir)
	if err != nil {
		t.Fatalf("reload file store: %v", err)
	}

	latest, ok := reloaded.Latest()
	if !ok {
		t.Fatal("expected persisted latest run")
	}
	if latest.ID != "run-2" {
		t.Fatalf("expected latest run-2, got %q", latest.ID)
	}

	filtered := reloaded.ListByFilter("project-1", "target-2", 5)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 filtered run, got %d", len(filtered))
	}
	if filtered[0].ID != "run-2" {
		t.Fatalf("expected filtered run-2, got %q", filtered[0].ID)
	}
}
