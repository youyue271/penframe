import { defineStore } from 'pinia'
import { ref, onUnmounted } from 'vue'
import { createSSE } from '@/api/client'
import type { SSEvent } from '@/types'

export const useSSEStore = defineStore('sse', () => {
  const connected = ref(false)
  const lastEvent = ref<SSEvent | null>(null)
  const events = ref<SSEvent[]>([])
  let source: EventSource | null = null

  type Listener = (event: SSEvent) => void
  const listeners: Listener[] = []

  function connect() {
    if (source) return

    try {
      source = createSSE('/api/events')

      source.onopen = () => {
        console.log('[SSE] Connected')
        connected.value = true
      }

      source.onerror = (err) => {
        console.warn('[SSE] Connection error, will auto-reconnect:', err)
        connected.value = false
        // Auto-reconnect is handled by EventSource.
      }
    } catch (err) {
      console.error('[SSE] Failed to create EventSource:', err)
    }

    // Listen for all event types.
    const eventTypes = [
      'portal_ready',
      'run_started',
      'node_started',
      'node_finished',
      'run_finished',
      'scan_error',
    ]

    for (const type of eventTypes) {
      source?.addEventListener(type, (e: MessageEvent) => {
        try {
          const data = JSON.parse(e.data) as SSEvent
          lastEvent.value = data
          events.value.push(data)
          // Keep only last 200 events in memory.
          if (events.value.length > 200) {
            events.value = events.value.slice(-200)
          }
          for (const listener of listeners) {
            listener(data)
          }
        } catch {
          // Ignore parse errors.
        }
      })
    }
  }

  function disconnect() {
    if (source) {
      source.close()
      source = null
      connected.value = false
    }
  }

  function onEvent(listener: Listener) {
    listeners.push(listener)
    return () => {
      const idx = listeners.indexOf(listener)
      if (idx >= 0) listeners.splice(idx, 1)
    }
  }

  return { connected, lastEvent, events, connect, disconnect, onEvent }
})
