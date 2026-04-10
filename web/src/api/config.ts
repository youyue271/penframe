import { get, put } from './client'

export interface ToolPathConfigItem {
  tool: string
  label: string
  var_name: string
  value: string
  default: string
  source: string
}

export interface ToolPathConfigResponse {
  workflow_path: string
  items: ToolPathConfigItem[]
}

export function fetchToolPathConfig(): Promise<ToolPathConfigResponse> {
  return get<ToolPathConfigResponse>('/api/config/tool-paths')
}

export function updateToolPathConfig(paths: Record<string, string>): Promise<ToolPathConfigResponse> {
  return put<ToolPathConfigResponse>('/api/config/tool-paths', { paths })
}
