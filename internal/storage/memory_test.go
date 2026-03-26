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

	store.Save("run-1", first)
	store.Save("run-2", second)

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
