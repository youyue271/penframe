<template>
  <div class="dashboard">
    <div class="dashboard-header">
      <h1>Dashboard</h1>
      <div class="dashboard-actions">
        <el-select
          v-model="selectedRunId"
          filterable
          clearable
          placeholder="Select target run"
          style="width: 420px;"
          @change="handleRunChange"
        >
          <el-option
            v-for="run in scanStore.runs"
            :key="run.id"
            :label="runOptionLabel(run)"
            :value="run.id"
          />
        </el-select>
        <el-button :loading="refreshing" @click="refreshSelectedRun()">Refresh</el-button>
      </div>
    </div>

    <el-alert
      v-if="assetStore.error"
      :title="assetStore.error"
      type="error"
      closable
      class="dashboard-alert"
      @close="assetStore.error = ''"
    />

    <el-card shadow="hover" class="run-summary-card">
      <el-descriptions :column="3" border size="small">
        <el-descriptions-item label="Target">{{ selectedTargetLabel }}</el-descriptions-item>
        <el-descriptions-item label="Run ID">{{ selectedRun?.id || '-' }}</el-descriptions-item>
        <el-descriptions-item label="Status">
          <el-tag v-if="selectedRun" size="small" :type="runStatusTag(selectedRun.summary.status)">
            {{ selectedRun.summary.status }}
          </el-tag>
          <span v-else>-</span>
        </el-descriptions-item>
        <el-descriptions-item label="Workflow">{{ selectedRun?.summary.workflow || '-' }}</el-descriptions-item>
        <el-descriptions-item label="Started">{{ formatDate(selectedRun?.summary.started_at) }}</el-descriptions-item>
        <el-descriptions-item label="Finished">{{ formatDate(selectedRun?.summary.finished_at) }}</el-descriptions-item>
      </el-descriptions>
    </el-card>

    <el-row :gutter="20" class="stat-cards">
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-number">{{ assetStore.summary.hosts }}</div>
          <div class="stat-label">Hosts</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-number">{{ assetStore.summary.ports }}</div>
          <div class="stat-label">Ports</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-number">{{ assetStore.summary.paths }}</div>
          <div class="stat-label">Paths</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-number">{{ assetStore.summary.vulns }}</div>
          <div class="stat-label">Vulns</div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" class="asset-layout">
      <el-col :span="10">
        <el-card shadow="hover" class="asset-card">
          <template #header>
            <div class="card-header">
              <span>Hosts</span>
              <el-tag size="small" type="info">{{ hostRows.length }}</el-tag>
            </div>
          </template>

          <el-table
            :data="hostRows"
            style="width: 100%"
            size="small"
            stripe
            highlight-current-row
            @row-click="selectHost"
          >
            <el-table-column prop="ip" label="IP" width="160" />
            <el-table-column prop="hostname" label="Hostname" show-overflow-tooltip />
            <el-table-column prop="status" label="Status" width="90">
              <template #default="{ row }">
                <el-tag size="small" :type="row.status === 'alive' ? 'success' : 'info'">
                  {{ row.status }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="port_count" label="Ports" width="70" />
            <el-table-column prop="path_count" label="Paths" width="70" />
            <el-table-column prop="vuln_count" label="Vulns" width="70" />
          </el-table>
        </el-card>
      </el-col>

      <el-col :span="14">
        <el-card shadow="hover" class="asset-card">
          <template #header>
            <div class="card-header">
              <span>{{ selectedHost ? `${selectedHost.ip} · Ports` : 'Ports' }}</span>
              <el-button v-if="selectedHost" size="small" text @click="resetHostSelection">Back to Hosts</el-button>
            </div>
          </template>

          <template v-if="selectedHost">
            <el-descriptions :column="4" border size="small" class="section-summary">
              <el-descriptions-item label="Host">{{ selectedHost.ip }}</el-descriptions-item>
              <el-descriptions-item label="Hostname">{{ selectedHost.hostname || '-' }}</el-descriptions-item>
              <el-descriptions-item label="Ports">{{ selectedHost.port_count }}</el-descriptions-item>
              <el-descriptions-item label="Paths / Vulns">{{ selectedHost.path_count }} / {{ selectedHost.vuln_count }}</el-descriptions-item>
            </el-descriptions>

            <el-table
              :data="selectedHost.ports || []"
              style="width: 100%"
              size="small"
              stripe
              highlight-current-row
              @row-click="selectPort"
            >
              <el-table-column prop="port" label="Port" width="90" />
              <el-table-column prop="protocol" label="Protocol" width="90" />
              <el-table-column prop="service" label="Service" width="140" />
              <el-table-column prop="banner" label="Banner" show-overflow-tooltip />
              <el-table-column label="Paths" width="80">
                <template #default="{ row }">
                  {{ row.paths?.length || 0 }}
                </template>
              </el-table-column>
              <el-table-column label="Vulns" width="80">
                <template #default="{ row }">
                  <el-tag v-if="row.vulns?.length" type="danger" size="small">
                    {{ row.vulns.length }}
                  </el-tag>
                  <span v-else>0</span>
                </template>
              </el-table-column>
            </el-table>
          </template>

          <div v-else class="empty-hint">
            Select a host to narrow down ports, paths, and vulnerabilities for the current target.
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-card v-if="selectedPort" shadow="hover" class="asset-card">
      <template #header>
        <div class="card-header">
          <span>{{ selectedHost?.ip }} / {{ selectedPort.port }}/{{ selectedPort.protocol }}</span>
          <el-button size="small" text @click="selectedPortId = ''">Back to Ports</el-button>
        </div>
      </template>

      <el-descriptions :column="4" border size="small" class="section-summary">
        <el-descriptions-item label="Service">{{ selectedPort.service || '-' }}</el-descriptions-item>
        <el-descriptions-item label="Banner">{{ selectedPort.banner || '-' }}</el-descriptions-item>
        <el-descriptions-item label="Paths">{{ selectedPort.paths?.length || 0 }}</el-descriptions-item>
        <el-descriptions-item label="Vulns">{{ selectedPort.vulns?.length || 0 }}</el-descriptions-item>
      </el-descriptions>

      <el-tabs v-model="portDetailTab">
        <el-tab-pane :label="`Paths (${selectedPort.paths?.length || 0})`" name="paths">
          <el-table :data="selectedPort.paths || []" style="width: 100%" size="small" stripe>
            <el-table-column prop="path" label="Path" />
            <el-table-column prop="status_code" label="Status" width="100" />
            <el-table-column prop="title" label="Title" show-overflow-tooltip />
            <el-table-column prop="tech" label="Tech" width="150" />
            <el-table-column prop="source" label="Source" width="150" />
          </el-table>
        </el-tab-pane>
        <el-tab-pane :label="`Vulnerabilities (${selectedPort.vulns?.length || 0})`" name="vulns">
          <el-table :data="selectedPort.vulns || []" style="width: 100%" size="small" stripe>
            <el-table-column prop="cve" label="CVE" width="150" />
            <el-table-column prop="name" label="Name" />
            <el-table-column prop="severity" label="Severity" width="120">
              <template #default="{ row }">
                <el-tag :type="severityType(row.severity)" size="small">
                  {{ row.severity }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="exp_available" label="Exploit" width="100">
              <template #default="{ row }">
                <el-tag v-if="row.exp_available" type="warning" size="small">Available</el-tag>
                <span v-else>-</span>
              </template>
            </el-table-column>
            <el-table-column prop="source" label="Source" width="150" />
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <el-row :gutter="20" class="footer-layout">
      <el-col :span="12">
        <el-card shadow="hover">
          <template #header>
            <span>Recent Runs</span>
          </template>
          <el-table :data="scanStore.runs" style="width: 100%" size="small" stripe @row-click="selectRunRow">
            <el-table-column prop="id" label="Run ID" width="200" show-overflow-tooltip />
            <el-table-column label="Target" show-overflow-tooltip>
              <template #default="{ row }">
                {{ runTarget(row) }}
              </template>
            </el-table-column>
            <el-table-column prop="summary.status" label="Status" width="100">
              <template #default="{ row }">
                <el-tag :type="runStatusTag(row.summary.status)" size="small">
                  {{ row.summary.status }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="summary.stats.total_nodes" label="Nodes" width="80" />
          </el-table>
        </el-card>
      </el-col>

      <el-col :span="12">
        <el-card shadow="hover">
          <template #header>
            <span>Live Events</span>
          </template>
          <div class="event-log">
            <div v-for="event in recentEvents" :key="event.timestamp_unix_milli" class="event-entry">
              <el-tag size="small" :type="eventTagType(event.type)">{{ event.type }}</el-tag>
              <span class="event-run">{{ event.run_id }}</span>
              <span v-if="event.node" class="event-node">{{ event.node.node_id }}</span>
              <span class="event-time">{{ formatTime(event.timestamp_unix_milli) }}</span>
            </div>
            <div v-if="recentEvents.length === 0" class="empty-hint">No events yet</div>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useAssetStore } from '@/stores/asset'
import { useScanStore } from '@/stores/scan'
import { useSSEStore } from '@/stores/sse'
import type { AssetHost, AssetPort, StoredRun } from '@/types'

type HostRow = AssetHost & {
  port_count: number
  path_count: number
  vuln_count: number
}

const assetStore = useAssetStore()
const scanStore = useScanStore()
const sseStore = useSSEStore()

const selectedRunId = ref('')
const selectedHostId = ref('')
const selectedPortId = ref('')
const portDetailTab = ref('paths')
const refreshing = ref(false)
let unsubscribe: (() => void) | null = null

const recentEvents = computed(() => {
  return [...sseStore.events].reverse().slice(0, 30)
})

const selectedRun = computed<StoredRun | null>(() => {
  const fromRuns = scanStore.runs.find((run) => run.id === selectedRunId.value)
  if (fromRuns) return fromRuns
  if (scanStore.currentRun?.id === selectedRunId.value) return scanStore.currentRun
  return null
})

const hostRows = computed<HostRow[]>(() => {
  return assetStore.hosts.map((host) => {
    const ports = host.ports || []
    const pathCount = ports.reduce((total, port) => total + (port.paths?.length || 0), 0)
    const vulnCount = ports.reduce((total, port) => total + (port.vulns?.length || 0), 0)
    return {
      ...host,
      port_count: ports.length,
      path_count: pathCount,
      vuln_count: vulnCount,
    }
  })
})

const selectedHost = computed<HostRow | null>(() => {
  return hostRows.value.find((host) => host.id === selectedHostId.value) || null
})

const selectedPort = computed<AssetPort | null>(() => {
  const ports = selectedHost.value?.ports || []
  return ports.find((port) => port.id === selectedPortId.value) || null
})

const selectedTargetLabel = computed(() => {
  const target = selectedRun.value?.summary.vars?.target
  return String(target || assetStore.target || '-')
})

function runTarget(run: StoredRun) {
  return String(run.summary.vars?.target || run.summary.vars?.target_host || run.id)
}

function runOptionLabel(run: StoredRun) {
  return `${runTarget(run)} · ${run.summary.status} · ${run.id}`
}

function runStatusTag(status: string) {
  if (status === 'succeeded') return 'success'
  if (status === 'failed') return 'danger'
  if (status === 'running') return 'warning'
  return 'info'
}

function eventTagType(type: string) {
  if (type.includes('finished')) return 'success'
  if (type.includes('started')) return 'warning'
  if (type.includes('error')) return 'danger'
  return 'info'
}

function severityType(severity: string) {
  const normalized = severity?.toLowerCase()
  if (normalized === 'critical' || normalized === 'high') return 'danger'
  if (normalized === 'medium') return 'warning'
  return 'info'
}

function formatDate(iso?: string) {
  if (!iso) return '-'
  return new Date(iso).toLocaleString()
}

function formatTime(ms: number) {
  return new Date(ms).toLocaleTimeString()
}

function selectHost(row: HostRow) {
  selectedHostId.value = row.id
  selectedPortId.value = ''
  portDetailTab.value = 'paths'
}

function selectPort(row: AssetPort) {
  selectedPortId.value = row.id
  portDetailTab.value = 'paths'
}

function resetHostSelection() {
  selectedHostId.value = ''
  selectedPortId.value = ''
}

function selectRunRow(row: StoredRun) {
  selectedRunId.value = row.id
  void handleRunChange(row.id)
}

async function handleRunChange(runId = selectedRunId.value) {
  selectedRunId.value = runId || ''
  resetHostSelection()
  await refreshSelectedRun(runId, false)
}

async function refreshSelectedRun(runId = selectedRunId.value, reloadRuns = true) {
  refreshing.value = true
  try {
    if (reloadRuns) {
      await scanStore.loadRuns()
    }

    const preferredRunId = runId || selectedRunId.value || scanStore.currentRunId || scanStore.runs[0]?.id || ''
    const runExists = !preferredRunId
      || scanStore.runs.some((item) => item.id === preferredRunId)
      || scanStore.currentRun?.id === preferredRunId
    const resolvedRunId = runExists ? preferredRunId : (scanStore.currentRunId || scanStore.runs[0]?.id || '')
    selectedRunId.value = resolvedRunId

    if (!resolvedRunId) {
      assetStore.clear()
      return
    }

    await assetStore.load(resolvedRunId)
  } finally {
    refreshing.value = false
  }
}

watch(hostRows, (rows) => {
  if (selectedHostId.value && !rows.some((host) => host.id === selectedHostId.value)) {
    selectedHostId.value = ''
    selectedPortId.value = ''
  }
})

watch(selectedHost, (host) => {
  const ports = host?.ports || []
  if (selectedPortId.value && !ports.some((port) => port.id === selectedPortId.value)) {
    selectedPortId.value = ''
  }
})

onMounted(() => {
  void refreshSelectedRun('', true)

  unsubscribe = sseStore.onEvent((event) => {
    if (!event.run_id) return
    if (!['run_started', 'node_finished', 'run_finished', 'scan_error'].includes(event.type)) return

    void (async () => {
      await scanStore.loadRuns()
      if (!selectedRunId.value) {
        selectedRunId.value = scanStore.currentRunId || event.run_id || scanStore.runs[0]?.id || ''
      }
      if (event.run_id === selectedRunId.value) {
        await assetStore.load(selectedRunId.value)
      }
    })()
  })
})

onBeforeUnmount(() => {
  unsubscribe?.()
})
</script>

<style scoped>
.dashboard h1 {
  color: #e5e5e5;
}

.dashboard-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 20px;
}

.dashboard-actions {
  display: flex;
  gap: 12px;
  align-items: center;
}

.dashboard-alert {
  margin-bottom: 20px;
}

.run-summary-card {
  margin-bottom: 20px;
}

.stat-cards {
  margin-bottom: 20px;
}

.stat-card {
  text-align: center;
  background: #1d1e1f;
  border-color: #303030;
}

.stat-number {
  font-size: 36px;
  font-weight: bold;
  color: #409eff;
}

.stat-label {
  color: #909399;
  margin-top: 4px;
}

.asset-layout,
.footer-layout {
  margin-top: 20px;
}

.asset-card {
  min-height: 100%;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.section-summary {
  margin-bottom: 16px;
}

.event-log {
  max-height: 320px;
  overflow-y: auto;
}

.event-entry {
  padding: 4px 0;
  display: flex;
  gap: 8px;
  align-items: center;
  font-size: 13px;
  border-bottom: 1px solid #252525;
}

.event-run {
  color: #909399;
  font-family: monospace;
}

.event-node {
  color: #67c23a;
  font-family: monospace;
}

.event-time {
  margin-left: auto;
  color: #606266;
  font-size: 12px;
}

.empty-hint {
  color: #606266;
  text-align: center;
  padding: 32px 20px;
}
</style>
