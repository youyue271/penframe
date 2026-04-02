import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { ScanTask, StoredRun } from '@/types'
import { startScan, fetchTasks, fetchRuns } from '@/api/scan'
import type { ScanRequest } from '@/types'

export const useScanStore = defineStore('scan', () => {
  const tasks = ref<ScanTask[]>([])
  const runs = ref<StoredRun[]>([])
  const currentRunId = ref('')
  const scanning = ref(false)
  const error = ref('')

  async function scan(req: ScanRequest) {
    scanning.value = true
    error.value = ''
    try {
      const resp = await startScan(req)
      currentRunId.value = resp.run_id
      tasks.value = resp.tasks
      return resp
    } catch (e: any) {
      error.value = e.message
      throw e
    } finally {
      scanning.value = false
    }
  }

  async function loadTasks() {
    try {
      const resp = await fetchTasks()
      tasks.value = resp.tasks
    } catch (e: any) {
      error.value = e.message
    }
  }

  async function loadRuns(limit = 20) {
    try {
      const resp = await fetchRuns(limit)
      runs.value = resp.runs
    } catch (e: any) {
      error.value = e.message
    }
  }

  return { tasks, runs, currentRunId, scanning, error, scan, loadTasks, loadRuns }
})
