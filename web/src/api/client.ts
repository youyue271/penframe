// HTTP client for the Penframe API.

const API_BASE = import.meta.env.VITE_API_BASE || ''

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

export function createSSE(path: string): EventSource {
  return new EventSource(`${API_BASE}${path}`)
}
