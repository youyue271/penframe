import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useThemeStore = defineStore('theme', () => {
  const isDark = ref(true)

  function toggleTheme() {
    isDark.value = !isDark.value
    applyTheme()
  }

  function applyTheme() {
    const root = document.documentElement
    if (isDark.value) {
      root.style.setProperty('--pf-bg-base', '#1a1d28')
      root.style.setProperty('--pf-bg-elevated', '#22252f')
      root.style.setProperty('--pf-bg-surface', '#2a2d37')
      root.style.setProperty('--pf-bg-hover', '#32353f')
      root.style.setProperty('--pf-bg-active', '#3a3d47')
      root.style.setProperty('--pf-bg-terminal', '#14161d')
      root.style.setProperty('--pf-border', '#3a3d47')
      root.style.setProperty('--pf-border-light', '#4a4d57')
      root.style.setProperty('--pf-text-primary', '#e8eaed')
      root.style.setProperty('--pf-text-secondary', '#b8bcc4')
      root.style.setProperty('--pf-text-muted', '#8b929c')
      root.style.setProperty('--pf-text-dim', '#5a5f6a')
      root.style.setProperty('--pf-sidebar-bg', '#1f2229')
    } else {
      // Light mode - Morandi style with high contrast
      root.style.setProperty('--pf-bg-base', '#e8e8e3')
      root.style.setProperty('--pf-bg-elevated', '#f5f5f0')
      root.style.setProperty('--pf-bg-surface', '#ededea')
      root.style.setProperty('--pf-bg-hover', '#e0e0db')
      root.style.setProperty('--pf-bg-active', '#d8d8d3')
      root.style.setProperty('--pf-bg-terminal', '#f8f8f5')
      root.style.setProperty('--pf-border', '#b0b0a8')
      root.style.setProperty('--pf-border-light', '#a0a098')
      root.style.setProperty('--pf-text-primary', '#0a0d08')
      root.style.setProperty('--pf-text-secondary', '#2a2d28')
      root.style.setProperty('--pf-text-muted', '#4a4d48')
      root.style.setProperty('--pf-text-dim', '#6a6d68')
      root.style.setProperty('--pf-sidebar-bg', '#a8a8a0')
    }
  }

  // Initialize theme on store creation
  applyTheme()

  return {
    isDark,
    toggleTheme,
  }
})
