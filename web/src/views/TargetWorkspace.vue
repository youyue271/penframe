<template>
  <div class="target-workspace">
    <h1>Target Workspace</h1>

    <el-card shadow="hover" class="workspace-card">
      <template #header>
        <span>Target</span>
      </template>
      <el-form label-width="120px" @submit.prevent="startTargetScan">
        <el-form-item label="Target Address">
          <el-input
            v-model="targetForm.target"
            placeholder="IP / CIDR / Domain / URL (e.g. https://target:3000)"
            clearable
          />
        </el-form-item>
        <el-form-item label="Strategy">
          <el-radio-group v-model="targetForm.strategy">
            <el-radio-button value="full">Full</el-radio-button>
            <el-radio-button value="recon">Recon</el-radio-button>
            <el-radio-button value="discovery">Discovery</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="Timeout (s)">
          <el-input-number v-model="targetForm.timeout" :min="30" :max="86400" :step="60" />
        </el-form-item>
      </el-form>
    </el-card>

    <el-card shadow="hover" class="workspace-card">
      <template #header>
        <span>Actions</span>
      </template>
      <div class="actions-row">
        <el-button type="primary" :loading="scanStore.scanning" @click="startTargetScan">
          Start Scan
        </el-button>
        <el-button @click="refreshCurrentRun" :disabled="!scanStore.currentRunId">
          Refresh Result
        </el-button>
      </div>

      <div class="actions-row actions-row-secondary">
        <el-select v-model="selectedExploitId" placeholder="Select exploit module" clearable style="width: 280px;">
          <el-option
            v-for="item in exploits"
            :key="item.id"
            :label="`${item.id} · ${item.name}`"
            :value="item.id"
          />
        </el-select>
        <el-button @click="loadExploits" :loading="loadingExploits">Refresh Modules</el-button>
        <el-button type="danger" :loading="executingExploit" @click="executeExploit">
          Execute Exploit
        </el-button>
      </div>
    </el-card>

    <el-card v-if="scanStore.currentRun" shadow="hover" class="workspace-card">
      <template #header>
        <span>Scan Result</span>
      </template>

      <el-descriptions :column="2" border size="small">
        <el-descriptions-item label="Run ID">{{ scanStore.currentRun.id }}</el-descriptions-item>
        <el-descriptions-item label="Status">
          <el-tag :type="runStatusTag(scanStore.currentRun.summary.status)">
            {{ scanStore.currentRun.summary.status }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="Workflow">{{ scanStore.currentRun.summary.workflow || '-' }}</el-descriptions-item>
        <el-descriptions-item label="Started">{{ formatDate(scanStore.currentRun.summary.started_at) }}</el-descriptions-item>
        <el-descriptions-item label="Finished">{{ formatDate(scanStore.currentRun.summary.finished_at) }}</el-descriptions-item>
        <el-descriptions-item label="Error">{{ scanStore.currentRun.summary.error || '-' }}</el-descriptions-item>
      </el-descriptions>

      <h4 class="section-title">Node Results</h4>
      <el-table :data="nodeResults" style="width: 100%" size="small" stripe>
        <el-table-column prop="node_id" label="Node" width="160" />
        <el-table-column prop="tool" label="Tool" width="140" />
        <el-table-column prop="executor" label="Executor" width="100" />
        <el-table-column prop="status" label="Status" width="100">
          <template #default="{ row }">
            <el-tag size="small" :type="nodeStatusTag(row.status)">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="duration_millis" label="Duration" width="100">
          <template #default="{ row }">
            {{ (row.duration_millis / 1000).toFixed(1) }}s
          </template>
        </el-table-column>
        <el-table-column prop="record_count" label="Records" width="80" />
        <el-table-column prop="error" label="Error" show-overflow-tooltip />
      </el-table>
    </el-card>

    <el-card v-if="directOutputs.length" shadow="hover" class="workspace-card">
      <template #header>
        <span>Direct Output</span>
      </template>
      <el-collapse>
        <el-collapse-item v-for="item in directOutputs" :key="item.key" :name="item.key">
          <template #title>
            <div class="output-title">
              <span>{{ item.title }}</span>
              <el-tag size="small" :type="nodeStatusTag(item.status)">
                {{ item.status }}
              </el-tag>
            </div>
          </template>
          <pre class="result-output">{{ item.content }}</pre>
        </el-collapse-item>
      </el-collapse>
    </el-card>

    <el-card v-if="exploitResult" shadow="hover" class="workspace-card">
      <template #header>
        <span>Exploit Result</span>
      </template>
      <pre class="result-output">{{ JSON.stringify(exploitResult, null, 2) }}</pre>
    </el-card>

    <el-alert
      v-if="scanStore.error"
      :title="scanStore.error"
      type="error"
      closable
      class="workspace-alert"
      @close="scanStore.error = ''"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useScanStore } from '@/stores/scan'
import { useSSEStore } from '@/stores/sse'
import { listExploits, triggerExploit } from '@/api/exploit'
import type { ExploitInfo, NodeRunResult } from '@/types'

const scanStore = useScanStore()
const sseStore = useSSEStore()

const targetForm = ref({
  target: '',
  strategy: 'full',
  timeout: 1800,
})

const exploits = ref<ExploitInfo[]>([])
const selectedExploitId = ref('')
const loadingExploits = ref(false)
const executingExploit = ref(false)
const exploitResult = ref<any>(null)
let unsubscribe: (() => void) | null = null

type DirectOutput = {
  key: string
  title: string
  status: string
  content: string
}

const nodeResults = computed<NodeRunResult[]>(() => {
  const run = scanStore.currentRun
  if (!run) return []
  const order = run.summary.execution_order || []
  const results = run.summary.node_results || {}
  const ordered: NodeRunResult[] = []
  for (const id of order) {
    if (results[id]) ordered.push(results[id])
  }
  for (const [id, result] of Object.entries(results)) {
    if (!order.includes(id)) ordered.push(result)
  }
  return ordered
})

const directOutputs = computed<DirectOutput[]>(() => {
  const outputs: DirectOutput[] = []
  for (const result of nodeResults.value) {
    if (result.records?.length) {
      outputs.push({
        key: `${result.node_id}-records`,
        title: `${result.node_id} · parsed records`,
        status: result.status,
        content: JSON.stringify(result.records, null, 2),
      })
    }
    if (result.metadata && Object.keys(result.metadata).length > 0) {
      outputs.push({
        key: `${result.node_id}-metadata`,
        title: `${result.node_id} · metadata`,
        status: result.status,
        content: JSON.stringify(result.metadata, null, 2),
      })
    }
    if (result.stdout?.trim()) {
      outputs.push({
        key: `${result.node_id}-stdout`,
        title: `${result.node_id} · stdout`,
        status: result.status,
        content: result.stdout.trim(),
      })
    }
  }
  return outputs
})

async function startTargetScan() {
  if (!targetForm.value.target.trim()) {
    ElMessage.warning('Please enter a target')
    return
  }
  try {
    const resp = await scanStore.scan({
      target: targetForm.value.target.trim(),
      strategy: targetForm.value.strategy,
      timeout_seconds: targetForm.value.timeout,
    })
    ElMessage.success(`Scan started: ${resp.run_id}`)
  } catch (e: any) {
    ElMessage.error(e.message)
  }
}

async function refreshCurrentRun() {
  if (!scanStore.currentRunId) return
  try {
    await scanStore.loadRun(scanStore.currentRunId)
    await scanStore.loadTasks()
  } catch {
    // store already records the error
  }
}

async function loadExploits() {
  loadingExploits.value = true
  try {
    const resp = await listExploits()
    exploits.value = resp.exploits || []
  } catch (e: any) {
    ElMessage.warning(`Could not load exploits: ${e.message}`)
    exploits.value = []
  } finally {
    loadingExploits.value = false
  }
}

async function executeExploit() {
  if (!targetForm.value.target.trim()) {
    ElMessage.warning('Please enter a target')
    return
  }
  executingExploit.value = true
  try {
    const resp = await triggerExploit(targetForm.value.target.trim(), selectedExploitId.value || undefined)
    exploitResult.value = resp
    ElMessage.success('Exploit executed')
  } catch (e: any) {
    ElMessage.error(e.message)
  } finally {
    executingExploit.value = false
  }
}

function runStatusTag(status: string): string {
  return status === 'succeeded' ? 'success' : status === 'failed' ? 'danger' : status === 'running' ? 'warning' : 'info'
}

function nodeStatusTag(status: string): string {
  return status === 'succeeded' ? 'success' : status === 'failed' ? 'danger' : status === 'skipped' ? 'warning' : status === 'running' ? '' : 'info'
}

function formatDate(iso: string) {
  if (!iso) return '-'
  return new Date(iso).toLocaleString()
}

onMounted(() => {
  void (async () => {
    await loadExploits()
    if (!scanStore.currentRunId) {
      await scanStore.loadLatestRun()
    }
    if (scanStore.currentRunId) {
      await refreshCurrentRun()
    }
  })()
  unsubscribe = sseStore.onEvent((event) => {
    if (!scanStore.currentRunId || event.run_id !== scanStore.currentRunId) return
    if (event.type === 'node_started' || event.type === 'node_finished' || event.type === 'run_finished' || event.type === 'scan_error') {
      void refreshCurrentRun()
    }
  })
})

onBeforeUnmount(() => {
  unsubscribe?.()
})
</script>

<style scoped>
.target-workspace h1 {
  color: #e5e5e5;
  margin-bottom: 20px;
}

.workspace-card {
  margin-bottom: 20px;
}

.actions-row {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
  align-items: center;
}

.actions-row-secondary {
  margin-top: 14px;
}

.section-title {
  margin: 16px 0 8px;
  color: #e5e5e5;
}

.result-output {
  margin: 0;
  background: #0d0d0d;
  color: #d4d7de;
  padding: 16px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 13px;
  line-height: 1.6;
  max-height: 420px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-word;
}

.workspace-alert {
  margin-top: 16px;
}

.output-title {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
</style>
