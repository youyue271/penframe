import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getVShellConfig, type VShellConfig } from '@/api/vshell'

export const useVShellStore = defineStore('vshell', () => {
  const visible = ref(false)
  const config = ref<VShellConfig | null>(null)
  const loading = ref(false)

  async function loadConfig() {
    loading.value = true
    try {
      config.value = await getVShellConfig()
    } catch (error) {
      console.error('[VShell] Failed to load config:', error)
    } finally {
      loading.value = false
    }
  }

  function toggle() {
    visible.value = !visible.value
    if (visible.value && !config.value) {
      loadConfig()
    }
  }

  function show() {
    visible.value = true
    if (!config.value) {
      loadConfig()
    }
  }

  function hide() {
    visible.value = false
  }

  return {
    visible,
    config,
    loading,
    toggle,
    show,
    hide,
    loadConfig,
  }
})
