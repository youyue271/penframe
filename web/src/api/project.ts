import { get, post, del } from './client'

export interface Project {
  id: string
  name: string
  url: string
  created_at: string
}

export function listProjects(): Promise<{ projects: Project[] }> {
  return get<{ projects: Project[] }>('/api/projects')
}

export function createProject(name: string, url: string): Promise<Project> {
  return post<Project>('/api/projects', { name, url })
}

export function deleteProject(id: string): Promise<void> {
  return del(`/api/projects/${id}`)
}
