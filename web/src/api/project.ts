import { get, post, del } from './client'

export interface Project {
  id: string
  name: string
  created_at: string
  targets?: Target[]
}

export interface Target {
  id: string
  project_id: string
  name: string
  url: string
  created_at: string
  last_scanned?: string
}

export function listProjects(): Promise<{ projects: Project[] }> {
  return get<{ projects: Project[] }>('/api/projects')
}

export function createProject(name: string): Promise<Project> {
  return post<Project>('/api/projects', { name })
}

export function deleteProject(id: string): Promise<void> {
  return del(`/api/projects/${id}`)
}
