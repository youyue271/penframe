// Core domain types mirroring Go models.

export interface AssetHost {
  id: string
  ip: string
  hostname?: string
  ports?: AssetPort[]
  status: string
  source?: string
}

export interface AssetPort {
  id: string
  host_id: string
  port: number
  protocol: string
  service?: string
  banner?: string
  paths?: AssetPath[]
  vulns?: AssetVuln[]
  source?: string
}

export interface AssetPath {
  id: string
  port_id: string
  path: string
  status_code: number
  title?: string
  tech?: string
  source?: string
}

export interface AssetVuln {
  id: string
  port_id?: string
  path_id?: string
  cve?: string
  name: string
  severity: string
  source: string
  exp_available: boolean
  detail?: string
}

export interface ScanTask {
  id: string
  type: string
  target: string
  status: string
  parent_id?: string
  node_id?: string
  error?: string
}

export interface CytoscapeElement {
  group: 'nodes' | 'edges'
  data: Record<string, any>
  classes?: string
}

export interface AssetGraphResponse {
  run_id: string
  target: string
  summary: { hosts: number; ports: number; paths: number; vulns: number }
  hosts: AssetHost[]
  elements: CytoscapeElement[]
}

export interface ScanRequest {
  target: string
  strategy?: string
  phases?: string[]
  tools?: Record<string, string>
  vars?: Record<string, any>
  timeout_seconds?: number
}

export interface ScanResponse {
  run_id: string
  tasks: ScanTask[]
  input: any
  run?: StoredRun
  error?: string
}

export interface ToolVariableDefinition {
  name: string
  required: boolean
  default?: string
  description?: string
}

export interface ToolDefinition {
  name: string
  category: string
  description?: string
  command_template?: string
  parser?: string
  variables?: ToolVariableDefinition[]
  private_variables?: ToolVariableDefinition[]
  metadata?: Record<string, any>
}

export interface WorkflowNodeDefinition {
  id: string
  tool: string
  executor: string
  shell?: string
  env?: Record<string, any>
  timeout_seconds?: number
  continue_on_error?: boolean
  inputs?: Record<string, any>
  mock?: Record<string, any>
}

export interface WorkflowDefinition {
  name: string
  description?: string
  global_vars?: Record<string, any>
  nodes: WorkflowNodeDefinition[]
  edges?: Array<{ from: string; to: string; condition?: string }>
}

export interface PortalStateResponse {
  workflow: WorkflowDefinition
  tools: ToolDefinition[]
  latest_run?: StoredRun
  recent_runs: StoredRun[]
  workflow_meta: {
    node_count: number
    edge_count: number
    entry_nodes: string[]
  }
}

export interface ExploitInfo {
  id: string
  name: string
  description: string
  cve: string
  severity: string
  targets: string[]
  supports_check?: boolean
  supports_exploit?: boolean
  supports_execute?: boolean
  supports_command?: boolean
  exploit_kind?: string
  tags?: string[]
  options?: ExploitOption[]
  default_command?: string
}

export interface ExploitOption {
  key: string
  label: string
  type?: string
  placeholder?: string
  description?: string
  required?: boolean
  modes?: string[]
}

export interface ExploitRequest {
  target: string
  exploit_id?: string
  mode?: 'check' | 'execute'
  command?: string
  leak_path?: string
  options?: Record<string, string>
}

export interface Target {
  id: string
  project_id: string
  name: string
  url: string
  created_at: string
  last_scanned?: string
  vshell_config?: {
    enabled?: boolean
    host?: string
    port?: number
  }
}

export interface ProjectItem {
  id: string
  name: string
  created_at: string
  targets?: Target[]
}

export interface RunSummary {
  workflow: string
  status: string
  error?: string
  started_at: string
  finished_at: string
  vars: Record<string, any>
  assets: Record<string, any>
  node_results: Record<string, NodeRunResult>
  execution_order: string[]
  stats: RunStats
}

export interface NodeRunResult {
  node_id: string
  tool: string
  executor: string
  status: string
  rendered_command?: string
  inputs?: Record<string, any>
  stdout?: string
  metadata?: Record<string, any>
  records?: any[]
  record_count: number
  error?: string
  skip_reason?: string
  duration_millis: number
  started_at: string
  finished_at: string
}

export interface RunStats {
  total_nodes: number
  executed_nodes: number
  succeeded_nodes: number
  failed_nodes: number
  skipped_nodes: number
}

export interface StoredRun {
  id: string
  project_id?: string
  target_id?: string
  summary: RunSummary
}

export interface SSEvent {
  type: string
  run_id?: string
  timestamp_unix_milli: number
  summary?: RunSummary
  node?: NodeRunResult
}
