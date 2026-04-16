import { get, put, post } from './client'

export interface VShellConfig {
  host: string
  port: number
  enabled: boolean
  web_url: string
}

export interface GeneratePayloadRequest {
  payload_type?: string // bash, nc, python, perl, php
}

export interface GeneratePayloadResponse {
  payload: string
  host: string
  port: number
  payload_type: string
}

export async function getVShellConfig(): Promise<VShellConfig> {
  return get('/api/v1/vshell/config')
}

export async function updateVShellConfig(config: Partial<VShellConfig>): Promise<VShellConfig> {
  return put('/api/v1/vshell/config', config)
}

export async function generatePayload(request: GeneratePayloadRequest = {}): Promise<GeneratePayloadResponse> {
  return post('/api/v1/vshell/generate_payload', request)
}

export async function generatePayloadForTarget(
  targetId: string,
  payloadType: string = 'bash'
): Promise<GeneratePayloadResponse> {
  return post(`/api/v1/vshell/generate_payload_for_target`, {
    target_id: targetId,
    payload_type: payloadType,
  })
}

export interface VShellListener {
  id: string
  name: string
  protocol: string
  host?: string
  port: number
  status?: string
  listen_addr?: string
  connect_addr?: string
}

export interface AddListenerRequest {
  name: string
  port: number
  protocol?: string
  listener_type?: string
  host?: string
  connect_host?: string
  disconnect_timeout?: number
  ping_interval?: number
  vkey?: string
  encrypt_salt?: string
}

export interface GenerateShellcodeRequest {
  listener_id: string
  client_type?: string
  arch?: string
  tp?: string
  host?: string
  port?: number
  upx?: boolean
  vkey?: string
  salt?: string
  listen?: boolean
  ebpf?: boolean
}

export interface ListListenersResponse {
  listeners: VShellListener[]
  total: number
}

export interface AddListenerResponse {
  success: boolean
  listener: VShellListener
  error?: string
}

export async function listListeners(): Promise<VShellListener[]> {
  const response = await get<ListListenersResponse>('/api/v1/vshell/listeners')
  return response.listeners || []
}

export async function addListener(request: AddListenerRequest): Promise<VShellListener> {
  const response = await post<AddListenerResponse>('/api/v1/vshell/listeners/add', request)
  if (!response.success) {
    throw new Error(response.error || 'Failed to add listener')
  }
  return response.listener
}

export async function generateShellcode(request: GenerateShellcodeRequest): Promise<Blob> {
  const response = await fetch('/api/v1/vshell/generate_shellcode', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(request),
  })

  if (!response.ok) {
    throw new Error(`Failed to generate shellcode: ${response.statusText}`)
  }

  return response.blob()
}
