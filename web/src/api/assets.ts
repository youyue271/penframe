import { get } from './client'
import type { AssetGraphResponse } from '@/types'

export function fetchAssets(): Promise<AssetGraphResponse> {
  return get<AssetGraphResponse>('/api/assets')
}

export function fetchAssetsByRun(runId: string): Promise<AssetGraphResponse> {
  return get<AssetGraphResponse>(`/api/assets/${runId}`)
}
