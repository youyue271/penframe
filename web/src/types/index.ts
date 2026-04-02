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
  elements: CytoscapeElement[]
}

export interface ScanRequest {
  target: string
  strategy?: string
  vars?: Record<string, any>
  timeout_seconds?: number
}

export interface ScanResponse {
  run_id: string
  tasks: ScanTask[]
  input: any
  error?: string
}

export interface ExploitInfo {
  id: string
  name: string
  description: string
  cve: string
  severity: string
  targets: string[]
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
  summary: RunSummary
}

export interface SSEvent {
  type: string
  run_id?: string
  timestamp_unix_milli: number
  summary?: RunSummary
  node?: NodeRunResult
}
