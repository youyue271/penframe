import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { ScanTask, StoredRun } from '@/types'
import { startScan, fetchTasks, fetchRuns, fetchRunById } from '@/api/scan'
import type { ScanRequest } from '@/types'

export const useScanStore = defineStore('scan', () => {
  const tasks = ref<ScanTask[]>([])
  const runs = ref<StoredRun[]>([])
  const currentRunId = ref('')
  const currentRun = ref<StoredRun | null>(null)
  const scanning = ref(false)
  const error = ref('')

  function upsertRun(run: StoredRun) {
    runs.value = [run, ...runs.value.filter((item) => item.id !== run.id)]
  }

  async function scan(req: ScanRequest) {
    scanning.value = true
    error.value = ''
    try {
      const resp = await startScan(req)
      currentRunId.value = resp.run_id
      tasks.value = resp.tasks
      if (resp.run) {
        currentRun.value = resp.run
        upsertRun(resp.run)
      } else {
        await loadRun(resp.run_id)
      }
      return resp
    } catch (e: any) {
      error.value = e.message
      throw e
    } finally {
      scanning.value = false
    }
  }

  async function loadTasks(runId = currentRunId.value) {
    try {
      const resp = await fetchTasks(runId)
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

  async function loadRun(id = currentRunId.value) {
    if (!id) return null
    try {
      currentRunId.value = id
      const resp = await fetchRunById(id)
      currentRun.value = resp.run
      upsertRun(resp.run)
      return resp.run
    } catch (e: any) {
      error.value = e.message
      throw e
    }
  }

  async function loadLatestRun() {
    try {
      const resp = await fetchRuns(1)
      runs.value = resp.runs
      const latest = resp.runs[0] || null
      currentRun.value = latest
      currentRunId.value = latest?.id || ''
      return latest
    } catch (e: any) {
      error.value = e.message
      return null
    }
  }

  return { tasks, runs, currentRunId, currentRun, scanning, error, scan, loadTasks, loadRuns, loadRun, loadLatestRun }
})
