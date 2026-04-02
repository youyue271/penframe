package portal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"penframe/internal/domain"
	"penframe/internal/workflow"
)

const eventTypePortalReady = "portal_ready"

type streamEvent struct {
	Type      string                `json:"type"`
	RunID     string                `json:"run_id,omitempty"`
	Timestamp int64                 `json:"timestamp_unix_milli"`
	Summary   *domain.RunSummary    `json:"summary,omitempty"`
	Node      *domain.NodeRunResult `json:"node,omitempty"`
}

type eventBroker struct {
	mu          sync.RWMutex
	nextID      int
	subscribers map[int]chan streamEvent
}

func newEventBroker() *eventBroker {
	return &eventBroker{
		subscribers: make(map[int]chan streamEvent),
	}
}

func (b *eventBroker) Subscribe() (<-chan streamEvent, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	subscriptionID := b.nextID
	b.nextID++
	ch := make(chan streamEvent, 32)
	b.subscribers[subscriptionID] = ch

	unsubscribe := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		subscriber, ok := b.subscribers[subscriptionID]
		if !ok {
			return
		}
		delete(b.subscribers, subscriptionID)
		close(subscriber)
	}
	return ch, unsubscribe
}

func (b *eventBroker) Publish(event streamEvent) {
	b.mu.RLock()
	subscribers := make([]chan streamEvent, 0, len(b.subscribers))
	for _, subscriber := range b.subscribers {
		subscribers = append(subscribers, subscriber)
	}
	b.mu.RUnlock()

	for _, subscriber := range subscribers {
		select {
		case subscriber <- event:
		default:
		}
	}
}

func newStreamEvent(runID string, event workflow.Event) streamEvent {
	return streamEvent{
		Type:      event.Type,
		RunID:     runID,
		Timestamp: event.Timestamp,
		Summary:   event.Summary,
		Node:      event.Node,
	}
}

func readyEvent() streamEvent {
	return streamEvent{
		Type:      eventTypePortalReady,
		Timestamp: time.Now().UTC().UnixMilli(),
	}
}

func writeSSE(w http.ResponseWriter, event streamEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal sse payload: %w", err)
	}
	if _, err := fmt.Fprintf(w, "event: %s\n", event.Type); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", payload); err != nil {
		return err
	}
	return nil
}
