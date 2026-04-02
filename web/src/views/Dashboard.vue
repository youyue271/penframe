<template>
  <div class="dashboard">
    <h1>Dashboard</h1>

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

    <el-row :gutter="20" style="margin-top: 20px;">
      <el-col :span="12">
        <el-card shadow="hover">
          <template #header>
            <span>Recent Runs</span>
          </template>
          <el-table :data="scanStore.runs" style="width: 100%" size="small">
            <el-table-column prop="id" label="Run ID" width="200" />
            <el-table-column prop="summary.status" label="Status" width="100">
              <template #default="{ row }">
                <el-tag
                  :type="row.summary.status === 'succeeded' ? 'success' : row.summary.status === 'failed' ? 'danger' : 'info'"
                  size="small"
                >
                  {{ row.summary.status }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="summary.stats.total_nodes" label="Nodes" width="80" />
            <el-table-column prop="summary.workflow" label="Workflow" />
          </el-table>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card shadow="hover">
          <template #header>
            <span>Scan Tasks</span>
          </template>
          <el-table :data="scanStore.tasks" style="width: 100%" size="small">
            <el-table-column prop="id" label="Task ID" width="180" />
            <el-table-column prop="type" label="Type" width="120" />
            <el-table-column prop="status" label="Status" width="100">
              <template #default="{ row }">
                <el-tag
                  :type="row.status === 'done' ? 'success' : row.status === 'running' ? 'warning' : row.status === 'failed' ? 'danger' : 'info'"
                  size="small"
                >
                  {{ row.status }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="target" label="Target" show-overflow-tooltip />
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" style="margin-top: 20px;">
      <el-col :span="24">
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
import { onMounted, computed } from 'vue'
import { useAssetStore } from '@/stores/asset'
import { useScanStore } from '@/stores/scan'
import { useSSEStore } from '@/stores/sse'

const assetStore = useAssetStore()
const scanStore = useScanStore()
const sseStore = useSSEStore()

const recentEvents = computed(() => {
  return [...sseStore.events].reverse().slice(0, 30)
})

function eventTagType(type: string) {
  if (type.includes('finished')) return 'success'
  if (type.includes('started')) return 'warning'
  if (type.includes('error')) return 'danger'
  return 'info'
}

function formatTime(ms: number) {
  return new Date(ms).toLocaleTimeString()
}

onMounted(() => {
  assetStore.load()
  scanStore.loadRuns()
  scanStore.loadTasks()
})
</script>

<style scoped>
.dashboard h1 {
  margin-bottom: 20px;
  color: #e5e5e5;
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

.event-log {
  max-height: 300px;
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
  padding: 20px;
}
</style>
