import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { AssetGraphResponse, CytoscapeElement } from '@/types'
import { fetchAssets } from '@/api/assets'

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

  async function load() {
    loading.value = true
    error.value = ''
    try {
      graph.value = await fetchAssets()
    } catch (e: any) {
      error.value = e.message
    } finally {
      loading.value = false
    }
  }

  function updateFromSSE(data: AssetGraphResponse) {
    graph.value = data
  }

  return { graph, loading, error, elements, summary, load, updateFromSSE }
})
