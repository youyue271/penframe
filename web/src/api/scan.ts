import { get, post } from './client'
import type { PortalStateResponse, ScanRequest, ScanResponse, ScanTask, StoredRun } from '@/types'

export function startScan(req: ScanRequest): Promise<ScanResponse> {
  return post<ScanResponse>('/api/scan', req)
}

export function fetchTasks(runId?: string): Promise<{ tasks: ScanTask[] }> {
  const query = runId ? `?run_id=${encodeURIComponent(runId)}` : ''
  return get<{ tasks: ScanTask[] }>(`/api/tasks${query}`)
}

export function fetchRuns(limit = 20): Promise<{ runs: StoredRun[] }> {
  return get<{ runs: StoredRun[] }>(`/api/runs?limit=${limit}`)
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
