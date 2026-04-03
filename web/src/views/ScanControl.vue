<template>
  <div class="scan-control">
    <h1>Scan Control</h1>

    <el-card shadow="hover" class="scan-form-card">
      <template #header>
        <span>Start Scan</span>
      </template>
      <el-form label-width="120px" @submit.prevent="doScan">
        <el-form-item label="Target">
          <el-input
            v-model="form.target"
            placeholder="IP / CIDR / Domain / URL (e.g. https://target:3000)"
            clearable
          />
        </el-form-item>
        <el-form-item label="Strategy">
          <el-radio-group v-model="form.strategy">
            <el-radio-button value="full">Full</el-radio-button>
            <el-radio-button value="recon">Recon</el-radio-button>
            <el-radio-button value="discovery">Discovery</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="Timeout (s)">
          <el-input-number v-model="form.timeout" :min="30" :max="86400" :step="60" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="doScan" :loading="scanStore.scanning">
            Start Scan
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card shadow="hover" style="margin-top: 20px;">
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center;">
          <span>{{ scanStore.currentRunId ? 'Current Run Tasks' : 'Scan Tasks' }}</span>
          <el-button size="small" @click="scanStore.loadTasks()">Refresh</el-button>
        </div>
      </template>
      <el-table :data="currentRunTasks" style="width: 100%" size="small" stripe>
        <el-table-column prop="id" label="Task ID" width="200" show-overflow-tooltip />
        <el-table-column prop="type" label="Type" width="140">
          <template #default="{ row }">
            <el-tag size="small" :type="typeTagColor(row.type)">{{ row.type }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="target" label="Target" show-overflow-tooltip />
        <el-table-column prop="status" label="Status" width="100">
          <template #default="{ row }">
            <el-tag
              size="small"
              :type="row.status === 'done' ? 'success' : row.status === 'running' ? 'warning' : row.status === 'failed' ? 'danger' : 'info'"
            >
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="error" label="Error" show-overflow-tooltip />
      </el-table>
    </el-card>

    <el-card v-if="scanStore.currentRun" shadow="hover" style="margin-top: 20px;">
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center;">
          <span>Latest Scan Result</span>
          <el-button size="small" @click="refreshCurrentRun">Refresh</el-button>
        </div>
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

      <h4 style="margin: 16px 0 8px; color: #e5e5e5;">Node Results</h4>
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
        <el-table-column label="Command" min-width="280">
          <template #default="{ row }">
            <code v-if="row.rendered_command" class="cmd-text">{{ row.rendered_command }}</code>
            <span v-else class="cmd-na">-</span>
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

    <el-card v-if="directOutputs.length" shadow="hover" style="margin-top: 20px;">
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

    <el-alert
      v-if="scanStore.error"
      :title="scanStore.error"
      type="error"
      closable
      style="margin-top: 16px;"
      @close="scanStore.error = ''"
    />

    <el-alert
      v-if="lastRunId"
      :title="`Scan started: ${lastRunId}`"
      type="success"
      closable
      style="margin-top: 16px;"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed, onBeforeUnmount } from 'vue'
import { useScanStore } from '@/stores/scan'
import { useSSEStore } from '@/stores/sse'
import { ElMessage } from 'element-plus'
import type { NodeRunResult } from '@/types'

const scanStore = useScanStore()
const sseStore = useSSEStore()

const form = ref({
  target: '',
  strategy: 'full',
  timeout: 1800,
})

const lastRunId = ref('')
let unsubscribe: (() => void) | null = null

type DirectOutput = {
  key: string
  title: string
  status: string
  content: string
}

const currentRunTasks = computed(() => {
  if (!scanStore.currentRunId) return scanStore.tasks
  return scanStore.tasks.filter((task) => task.parent_id === scanStore.currentRunId)
})

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

async function doScan() {
  if (!form.value.target.trim()) {
    ElMessage.warning('Please enter a target')
    return
  }
  try {
    const resp = await scanStore.scan({
      target: form.value.target.trim(),
      strategy: form.value.strategy,
      timeout_seconds: form.value.timeout,
    })
    lastRunId.value = resp.run_id
    ElMessage.success(`Scan started: ${resp.run_id}`)
  } catch (e: any) {
    ElMessage.error(e.message)
  }
}

async function refreshCurrentRun() {
  if (!scanStore.currentRunId) return
  try {
    await scanStore.loadRun(scanStore.currentRunId)
    await scanStore.loadTasks(scanStore.currentRunId)
  } catch {
    // store already records the error
  }
}

function typeTagColor(type: string): string {
  const colors: Record<string, string> = {
    seed: 'info',
    host_discovery: '',
    port_scan: 'warning',
    path_scan: 'success',
    vuln_scan: 'danger',
    exploit: 'danger',
  }
  return colors[type] || 'info'
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
    await scanStore.loadTasks()
    if (!scanStore.currentRunId) {
      await scanStore.loadLatestRun()
    }
    if (scanStore.currentRunId) {
      await refreshCurrentRun()
    }
  })()
  unsubscribe = sseStore.onEvent((event) => {
    if (!scanStore.currentRunId || event.run_id !== scanStore.currentRunId) return
    if (event.type === 'run_started' || event.type === 'node_started' || event.type === 'node_finished' || event.type === 'run_finished' || event.type === 'scan_error') {
      void refreshCurrentRun()
    }
  })
})

onBeforeUnmount(() => {
  unsubscribe?.()
})
</script>

<style scoped>
.scan-control h1 {
  color: #e5e5e5;
  margin-bottom: 20px;
}

.scan-form-card {
  background: #1d1e1f;
  border-color: #303030;
}

.output-title {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.result-output {
  margin: 0;
  padding: 16px;
  border-radius: 6px;
  background: #0d0d0d;
  color: #d4d7de;
  font-family: monospace;
  font-size: 12px;
  line-height: 1.6;
  overflow-x: auto;
  white-space: pre-wrap;
  word-break: break-word;
}

.cmd-text {
  font-family: 'Consolas', 'Fira Code', monospace;
  font-size: 12px;
  color: #a0cfff;
  word-break: break-all;
  white-space: pre-wrap;
}

.cmd-na {
  color: #606266;
}
</style>
