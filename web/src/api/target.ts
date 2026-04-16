import { get, post, put, del } from './client'

export interface Target {
  id: string
  project_id: string
  name: string
  url: string
  created_at: string
  last_scanned?: string
  vshell_config?: {
    enabled?: boolean
    listener_id?: string
    host?: string
    port?: number
  }
}

export async function getProjectTargets(projectId: string): Promise<Target[]> {
  const res = await get<{ targets: Target[] }>(`/api/projects/${projectId}/targets`)
  return res.targets || []
}

export async function addTarget(projectId: string, name: string, url: string): Promise<Target> {
  return await post(`/api/projects/${projectId}/targets`, { name, url })
}

export async function updateTarget(
  projectId: string,
  targetId: string,
  name: string,
  url: string,
  vshell_config?: { enabled?: boolean; host?: string; port?: number }
): Promise<Target> {
  return await put<Target>(`/api/projects/${projectId}/targets/${targetId}`, { name, url, vshell_config })
}

export async function deleteTarget(projectId: string, targetId: string): Promise<void> {
  await del(`/api/projects/${projectId}/targets/${targetId}`)
}

export async function getTarget(targetId: string): Promise<Target> {
  return await get(`/api/targets/${targetId}`)
}
