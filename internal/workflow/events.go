package workflow

import (
	"context"

	"penframe/internal/domain"
)

const (
	EventRunStarted   = "run_started"
	EventNodeStarted  = "node_started"
	EventNodeFinished = "node_finished"
	EventRunFinished  = "run_finished"
)

type Event struct {
	Type      string                `json:"type"`
	Timestamp int64                 `json:"timestamp_unix_milli"`
	Summary   *domain.RunSummary    `json:"summary,omitempty"`
	Node      *domain.NodeRunResult `json:"node,omitempty"`
}

type EventObserver interface {
	OnWorkflowEvent(Event)
}

type EventObserverFunc func(Event)

func (fn EventObserverFunc) OnWorkflowEvent(event Event) {
	if fn == nil {
		return
	}
	fn(event)
}

type eventObserverContextKey struct{}

func WithEventObserver(ctx context.Context, observer EventObserver) context.Context {
	if observer == nil {
		return ctx
	}
	return context.WithValue(ctx, eventObserverContextKey{}, observer)
}

func emitEvent(ctx context.Context, event Event) {
	observer, ok := ctx.Value(eventObserverContextKey{}).(EventObserver)
	if !ok || observer == nil {
		return
	}
	observer.OnWorkflowEvent(event)
}

func emitRunStarted(ctx context.Context, summary domain.RunSummary) {
	snapshot := snapshotRunSummary(summary)
	emitEvent(ctx, Event{
		Type:      EventRunStarted,
		Timestamp: summary.StartedAt.UnixMilli(),
		Summary:   &snapshot,
	})
}

func emitNodeStarted(ctx context.Context, nodeResult domain.NodeRunResult) {
	snapshot := snapshotNodeRunResult(nodeResult)
	emitEvent(ctx, Event{
		Type:      EventNodeStarted,
		Timestamp: nodeResult.StartedAt.UnixMilli(),
		Node:      &snapshot,
	})
}

func emitNodeFinished(ctx context.Context, nodeResult domain.NodeRunResult, summary domain.RunSummary) {
	nodeSnapshot := snapshotNodeRunResult(nodeResult)
	summarySnapshot := snapshotRunSummary(summary)
	emitEvent(ctx, Event{
		Type:      EventNodeFinished,
		Timestamp: nodeResult.FinishedAt.UnixMilli(),
		Node:      &nodeSnapshot,
		Summary:   &summarySnapshot,
	})
}

func emitRunFinished(ctx context.Context, summary domain.RunSummary) {
	snapshot := snapshotRunSummary(summary)
	emitEvent(ctx, Event{
		Type:      EventRunFinished,
		Timestamp: summary.FinishedAt.UnixMilli(),
		Summary:   &snapshot,
	})
}

func snapshotRunSummary(summary domain.RunSummary) domain.RunSummary {
	nodeResults := make(map[string]domain.NodeRunResult, len(summary.NodeResults))
	for nodeID, nodeResult := range summary.NodeResults {
		nodeResults[nodeID] = snapshotNodeRunResult(nodeResult)
	}

	return domain.RunSummary{
		Workflow:       summary.Workflow,
		Status:         summary.Status,
		Error:          summary.Error,
		StartedAt:      summary.StartedAt,
		FinishedAt:     summary.FinishedAt,
		Vars:           copyDynamicMap(summary.Vars),
		Assets:         copyDynamicMap(summary.Assets),
		NodeResults:    nodeResults,
		ExecutionOrder: append([]string(nil), summary.ExecutionOrder...),
		Stats:          summary.Stats,
	}
}

func snapshotNodeRunResult(nodeResult domain.NodeRunResult) domain.NodeRunResult {
	records := make([]domain.ParsedRecord, 0, len(nodeResult.Records))
	for _, record := range nodeResult.Records {
		records = append(records, domain.ParsedRecord{
			Rule:   record.Rule,
			Path:   record.Path,
			Fields: copyStringMap(record.Fields),
		})
	}

	return domain.NodeRunResult{
		NodeID:          nodeResult.NodeID,
		Tool:            nodeResult.Tool,
		Executor:        nodeResult.Executor,
		Status:          nodeResult.Status,
		RenderedCommand: nodeResult.RenderedCommand,
		Inputs:          copyDynamicMap(nodeResult.Inputs),
		Stdout:          nodeResult.Stdout,
		Metadata:        copyDynamicMap(nodeResult.Metadata),
		Records:         records,
		RecordCount:     nodeResult.RecordCount,
		Error:           nodeResult.Error,
		SkipReason:      nodeResult.SkipReason,
		DurationMillis:  nodeResult.DurationMillis,
		StartedAt:       nodeResult.StartedAt,
		FinishedAt:      nodeResult.FinishedAt,
	}
}

func copyDynamicMap(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	cloned := make(map[string]any, len(input))
	for key, value := range input {
		cloned[key] = copyDynamicValue(value)
	}
	return cloned
}

func copyStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func copyDynamicSlice(input []any) []any {
	if input == nil {
		return nil
	}
	cloned := make([]any, len(input))
	for idx, value := range input {
		cloned[idx] = copyDynamicValue(value)
	}
	return cloned
}

func copyDynamicValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return copyDynamicMap(typed)
	case []any:
		return copyDynamicSlice(typed)
	case []map[string]any:
		cloned := make([]map[string]any, len(typed))
		for idx, item := range typed {
			cloned[idx] = copyDynamicMap(item)
		}
		return cloned
	case map[string]string:
		return copyStringMap(typed)
	case []string:
		return append([]string(nil), typed...)
	case []int:
		return append([]int(nil), typed...)
	case []int64:
		return append([]int64(nil), typed...)
	case []float64:
		return append([]float64(nil), typed...)
	case []bool:
		return append([]bool(nil), typed...)
	default:
		return value
	}
}
