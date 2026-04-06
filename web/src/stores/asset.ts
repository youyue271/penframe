import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { AssetGraphResponse, AssetHost, CytoscapeElement, StoredRun } from '@/types'
import { fetchAssets, fetchAssetsByRun } from '@/api/assets'
import { fetchRuns } from '@/api/scan'

export const useAssetStore = defineStore('asset', () => {
  const graph = ref<AssetGraphResponse | null>(null)
  const loading = ref(false)
  const error = ref('')

  const elements = computed<CytoscapeElement[]>(() => {
    return (graph.value?.elements as CytoscapeElement[]) || []
  })

  const summary = computed(() => {
    return graph.value?.summary || { hosts: 0, ports: 0, paths: 0, vulns: 0 }
  })

  const hosts = computed<AssetHost[]>(() => {
    return graph.value?.hosts || []
  })

  const target = computed(() => {
    return graph.value?.target || ''
  })

  const runId = computed(() => {
    return graph.value?.run_id || ''
  })

  function emptyGraph(target = ''): AssetGraphResponse {
    return {
      run_id: '',
      target,
      summary: { hosts: 0, ports: 0, paths: 0, vulns: 0 },
      hosts: [],
      elements: [],
    }
  }

  function normalizeText(value?: string) {
    return String(value || '').trim().replace(/\/+$/, '').toLowerCase()
  }

  function normalizeOrigin(value?: string) {
    const raw = String(value || '').trim()
    if (!raw) return ''
    try {
      const parsed = raw.includes('://') ? new URL(raw) : new URL(`https://${raw}`)
      return `${parsed.protocol}//${parsed.host}`.replace(/\/+$/, '').toLowerCase()
    } catch {
      return normalizeText(raw)
    }
  }

  function runMatchesTarget(run: StoredRun, targetUrl: string) {
    const vars = run.summary?.vars || {}
    const target = normalizeText(targetUrl)
    const origin = normalizeOrigin(targetUrl)
    const candidates = [
      vars.target,
      vars.target_url,
      vars.target_origin,
      vars.target_hostport,
      vars.target_host,
    ]

    return candidates.some((candidate) => {
      const normalized = normalizeText(String(candidate || ''))
      if (!normalized) return false
      return normalized === target || normalizeOrigin(normalized) === origin || normalized === origin
    })
  }

  async function load(runId?: string) {
    loading.value = true
    error.value = ''
    try {
      graph.value = runId ? await fetchAssetsByRun(runId) : await fetchAssets()
    } catch (e: any) {
      error.value = e.message
    } finally {
      loading.value = false
    }
  }

  async function loadByTarget(targetUrl: string, targetId?: string) {
    loading.value = true
    error.value = ''
    try {
      const runsResponse = await fetchRuns(100)
      const matchedRun = runsResponse.runs.find((run) => {
        if (targetId && run.target_id && run.target_id === targetId) {
          return true
        }
        return runMatchesTarget(run, targetUrl)
      })
      if (!matchedRun) {
        graph.value = emptyGraph(targetUrl)
        return
      }

      graph.value = await fetchAssetsByRun(matchedRun.id)
    } catch (e: any) {
      error.value = e.message
      graph.value = emptyGraph(targetUrl)
    } finally {
      loading.value = false
    }
  }

  function clear() {
    graph.value = null
    error.value = ''
  }

  function updateFromSSE(data: AssetGraphResponse) {
    graph.value = data
  }

  return { graph, loading, error, elements, summary, hosts, target, runId, load, loadByTarget, clear, updateFromSSE }
})
