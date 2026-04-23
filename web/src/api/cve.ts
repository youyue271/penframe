import { get, put, del } from './client'

export interface NucleiTemplate {
  id: string
  name: string
  severity: string
  cve_id: string
  vendor: string
  tags: string
}

export interface CVEEntry {
  id: string
  vendor: string
  severity: string
  tags: string[]
  templates: NucleiTemplate[]
  exp_module: any | null
  tested: boolean
}

export interface TagInfo {
  name: string
  count: number
}

export async function listCVEs(filter?: string, tag?: string): Promise<{ cves: CVEEntry[]; count: number }> {
  const params = new URLSearchParams()
  if (filter) params.set('filter', filter)
  if (tag) params.set('tag', tag)
  const qs = params.toString()
  return get<{ cves: CVEEntry[]; count: number }>(`/api/cves${qs ? '?' + qs : ''}`)
}

export async function setTested(cveId: string): Promise<void> {
  await put(`/api/cves/${encodeURIComponent(cveId)}/tested`, {})
}

export async function setUntested(cveId: string): Promise<void> {
  await del(`/api/cves/${encodeURIComponent(cveId)}/tested`)
}

export async function listTags(): Promise<TagInfo[]> {
  const res = await get<{ tags: TagInfo[] }>('/api/cve-tags')
  return res.tags || []
}
