# Architecture Mapping

## Current Scope

This repository currently implements the orchestration core for a safe MVP:

- `Portal`: not implemented yet
- `Orchestrator`: implemented as Go packages under `internal/workflow`, `internal/parser`, `internal/tooling`, `internal/config`
- `Executor`: `mock` and `local` executors are implemented under `internal/executor`
- `Storage`: in-memory placeholder under `internal/storage`

## Layer Mapping

### 1. Portal

Planned responsibilities:

- Workflow visual editing
- Tool registration UI
- Task history and asset dashboard

Current state:

- CLI entrypoint is `cmd/opo/main.go`
- Basic embedded portal UI is available at `cmd/portal/main.go`

Current interface:

- Embedded HTTP server serving a dashboard and JSON API
- `POST /api/run` for demo workflow execution
- `GET /api/state` for workflow metadata and recent runs
- `GET /api/runs` and `GET /api/runs/{id}` for run lookup
- `POST /api/reload` for config reload without restarting the server

Suggested next interface:

- Add workflow submission and run lookup beyond the demo config
- WebSocket or SSE stream for node state updates

### 2. Orchestrator

Current packages:

- `internal/tooling`: tool registry and lookup
- `internal/config`: YAML loading for workflows and parser rules
- `internal/parser`: regex-driven output normalization into `assets`
- `internal/runtime`: variable and template rendering
- `internal/workflow`: DAG traversal and condition evaluation

Current behavior:

- Load tool catalog and workflow definitions from YAML
- Render node inputs from `vars`, `assets`, `results`
- Run nodes through an executor abstraction
- Parse stdout into structured `assets`
- Evaluate edge conditions and schedule downstream nodes
- Persist run status as `succeeded/failed` with node-level `succeeded/failed/skipped`
- Expose run statistics and node duration/error metadata

Suggested next interface:

- Replace `MiniExprEvaluator` with `expr`
- Add node state machine and retries
- Persist run history to PostgreSQL
- Introduce run IDs, cancellation, and resumability

### 3. Executor

Current implementation:

- `mock` executor reading fixture files
- `local` executor running rendered command templates
- in WSL, `local` auto-switches to `powershell.exe` when the command target is `.exe`

Why this boundary exists:

- It keeps the orchestration core testable without binding the project to any offensive runtime channel.

Extension points:

- `Executor` interface in `internal/executor/executor.go`
- Each executor can decide how to transform rendered inputs into execution results

Safe next steps:

- Add command auditing and allowlists
- Add per-node timeout and environment controls
- Add realtime stdout/stderr streaming

### 4. Storage

Current implementation:

- `MemoryStore` placeholder

Suggested next interface:

- `RunRepository`
- `ToolRepository`
- `AssetRepository`

Suggested persistence split:

- PostgreSQL for run history, assets, node logs, tool metadata
- Redis for active run state, queue metadata, and transient scheduling signals

## Data Model Direction

The core runtime data objects are:

- `ToolDefinition`
- `ParserRuleSet`
- `Workflow`
- `NodeRunResult`
- `RunSummary`

That gives you a stable seam for:

- API DTO design
- database schema design
- report generation
- frontend graph rendering

## Deliberately Omitted

The repository does not implement:

- WebShell / VShell / C2 bridges
- pivoting agents or reverse connection channels
- automated exploit execution
- autonomous attack-path generation

Those capabilities materially increase offensive reach. The current code keeps the project at the orchestration-model layer so you can settle interfaces and data contracts first.
