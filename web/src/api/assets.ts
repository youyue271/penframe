import { get } from './client'
import type { AssetGraphResponse } from '@/types'

export function fetchAssets(): Promise<AssetGraphResponse> {
  return get<AssetGraphResponse>('/api/assets')
}

export function fetchAssetsByRun(runId: string): Promise<AssetGraphResponse> {
  return get<AssetGraphResponse>(`/api/assets/${runId}`)
}

export function fetchHosts(): Promise<any[]> {
  return get<any[]>('/api/hosts')
}

export function fetchHostPorts(hostId: string): Promise<any[]> {
  return get<any[]>(`/api/hosts/${hostId}`)
}

export function fetchPortDetails(portId: string): Promise<any> {
  return get<any>(`/api/ports/${portId}`)
}
