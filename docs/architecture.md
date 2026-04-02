# Architecture Mapping

## Current Scope

This repository implements the orchestration core:

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
- `GET /api/events` for SSE lifecycle updates
- `GET /api/state` for workflow metadata and recent runs
- `GET /api/runs` and `GET /api/runs/{id}` for run lookup
- `POST /api/reload` for config reload without restarting the server

Suggested next interface:

- Add workflow submission and run lookup beyond the demo config
- Add node log tailing on top of the existing SSE lifecycle stream

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
- `local` follows the current `$SHELL` by default and can be overridden per workflow node
- `local` supports per-node environment overrides for cases like proxy and DNS control
- in WSL, `local` auto-switches to `powershell.exe` when the command target is `.exe`

Extension points:

- `Executor` interface in `internal/executor/executor.go`
- Each executor can decide how to transform rendered inputs into execution results

Suggested next steps:

- Add command auditing and allowlists
- Add per-node timeout and environment controls
- Add realtime stdout/stderr streaming
- Implement WebShell / VShell / C2 bridge adapters
- Implement pivoting agents and reverse connection channels
- Implement automated exploit execution modules
- Implement autonomous attack-path generation

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

## Planned Capabilities

The following capabilities are planned for implementation to support real-world penetration testing:

- WebShell / VShell / C2 bridge adapters
- Pivoting agents and reverse connection channels
- Automated exploit execution
- Autonomous attack-path generation

These will be implemented as executor extensions and workflow modules to enable full-scope penetration testing workflows.
