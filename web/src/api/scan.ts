import { get, post } from './client'
import type { PortalStateResponse, ScanRequest, ScanResponse, ScanTask, StoredRun } from '@/types'

export interface ScanRequestWithContext extends ScanRequest {
  project_id?: string
  target_id?: string
}

export interface RunQuery {
  projectId?: string
  targetId?: string
}

export interface OutputFile {
  name: string
  size_bytes: number
  type: string
}

export function startScan(req: ScanRequestWithContext): Promise<ScanResponse> {
  return post<ScanResponse>('/api/scan', req)
}

export function fetchTasks(runId?: string): Promise<{ tasks: ScanTask[] }> {
  const query = runId ? `?run_id=${encodeURIComponent(runId)}` : ''
  return get<{ tasks: ScanTask[] }>(`/api/tasks${query}`)
}

export function fetchRuns(limit = 20, query: RunQuery = {}): Promise<{ runs: StoredRun[] }> {
  const params = new URLSearchParams({ limit: String(limit) })
  if (query.projectId) params.set('project_id', query.projectId)
  if (query.targetId) params.set('target_id', query.targetId)
  return get<{ runs: StoredRun[] }>(`/api/runs?${params.toString()}`)
}

export async function fetchLatestTargetRun(projectId: string, targetId: string): Promise<StoredRun | null> {
  const response = await fetchRuns(1, { projectId, targetId })
  return response.runs?.[0] || null
}

export function fetchRunById(id: string): Promise<{ run: StoredRun }> {
  return get<{ run: StoredRun }>(`/api/runs/${id}`)
}

export function fetchState(): Promise<PortalStateResponse> {
  return get<PortalStateResponse>('/api/state')
}

export function reloadConfig(): Promise<any> {
  return post<any>('/api/reload')
}

export function fetchOutputFiles(runId: string): Promise<{ files: OutputFile[] }> {
  return get<{ files: OutputFile[] }>(`/api/output-files/${runId}`)
}

export function fetchOutputFileContent(runId: string, filename: string): Promise<{ content?: string, lines?: string[] }> {
  return get<{ content?: string, lines?: string[] }>(`/api/output-files/${runId}/${encodeURIComponent(filename)}`)
}
