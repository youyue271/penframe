import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { AssetGraphResponse, AssetHost, CytoscapeElement } from '@/types'
import { fetchAssets, fetchAssetsByRun } from '@/api/assets'

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

  function clear() {
    graph.value = null
    error.value = ''
  }

  function updateFromSSE(data: AssetGraphResponse) {
    graph.value = data
  }

  return { graph, loading, error, elements, summary, hosts, target, runId, load, clear, updateFromSSE }
})
