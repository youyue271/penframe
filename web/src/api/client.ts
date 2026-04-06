// HTTP client for the Penframe API.

export const API_BASE = import.meta.env.VITE_API_BASE || ''

export async function get<T>(path: string): Promise<T> {
  const resp = await fetch(`${API_BASE}${path}`)
  if (!resp.ok) {
    throw new Error(`GET ${path} failed: ${resp.status}`)
  }
  return resp.json()
}

export async function post<T>(path: string, body?: any): Promise<T> {
  const resp = await fetch(`${API_BASE}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: body ? JSON.stringify(body) : undefined,
  })
  if (!resp.ok) {
    const text = await resp.text()
    throw new Error(`POST ${path} failed: ${resp.status} - ${text}`)
  }
  return resp.json()
}

export async function put<T>(path: string, body?: any): Promise<T> {
  const resp = await fetch(`${API_BASE}${path}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: body ? JSON.stringify(body) : undefined,
  })
  if (!resp.ok) {
    const text = await resp.text()
    throw new Error(`PUT ${path} failed: ${resp.status} - ${text}`)
  }
  return resp.json()
}

export async function del(path: string): Promise<void> {
  const resp = await fetch(`${API_BASE}${path}`, {
    method: 'DELETE',
  })
  if (!resp.ok) {
    const text = await resp.text()
    throw new Error(`DELETE ${path} failed: ${resp.status} - ${text}`)
  }
}

export function createSSE(path: string): EventSource {
  return new EventSource(`${API_BASE}${path}`)
}
