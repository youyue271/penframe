<template>
  <div class="run-history">
    <h1>Run History</h1>

    <el-card shadow="hover">
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center;">
          <span>Recent Runs</span>
          <el-button size="small" @click="scanStore.loadRuns()">Refresh</el-button>
        </div>
      </template>

      <el-table
        :data="scanStore.runs"
        style="width: 100%"
        stripe
        @row-click="selectRun"
        highlight-current-row
      >
        <el-table-column prop="id" label="Run ID" width="240" />
        <el-table-column prop="summary.workflow" label="Workflow" width="200" />
        <el-table-column prop="summary.status" label="Status" width="120">
          <template #default="{ row }">
            <el-tag
              :type="row.summary.status === 'succeeded' ? 'success' : row.summary.status === 'failed' ? 'danger' : row.summary.status === 'running' ? 'warning' : 'info'"
            >
              {{ row.summary.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="Nodes" width="200">
          <template #default="{ row }">
            <span>
              {{ row.summary.stats.succeeded_nodes }}/{{ row.summary.stats.total_nodes }}
              <el-tag v-if="row.summary.stats.failed_nodes > 0" type="danger" size="small">
                {{ row.summary.stats.failed_nodes }} failed
              </el-tag>
            </span>
          </template>
        </el-table-column>
        <el-table-column label="Started" width="200">
          <template #default="{ row }">
            {{ formatDate(row.summary.started_at) }}
          </template>
        </el-table-column>
        <el-table-column prop="summary.error" label="Error" show-overflow-tooltip />
      </el-table>
    </el-card>

    <!-- Run detail -->
    <el-card v-if="selectedRun" shadow="hover" style="margin-top: 20px;">
      <template #header>
        <span>Run Detail: {{ selectedRun.id }}</span>
      </template>

      <el-descriptions :column="2" border size="small">
        <el-descriptions-item label="Workflow">{{ selectedRun.summary.workflow }}</el-descriptions-item>
        <el-descriptions-item label="Status">
          <el-tag :type="selectedRun.summary.status === 'succeeded' ? 'success' : 'danger'">
            {{ selectedRun.summary.status }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="Started">{{ formatDate(selectedRun.summary.started_at) }}</el-descriptions-item>
        <el-descriptions-item label="Finished">{{ formatDate(selectedRun.summary.finished_at) }}</el-descriptions-item>
      </el-descriptions>

      <h4 style="margin: 16px 0 8px; color: #e5e5e5;">Node Results</h4>
      <el-table
        :data="nodeResults"
        style="width: 100%"
        size="small"
        stripe
      >
        <el-table-column prop="node_id" label="Node" width="160" />
        <el-table-column prop="tool" label="Tool" width="140" />
        <el-table-column prop="executor" label="Executor" width="80" />
        <el-table-column prop="status" label="Status" width="100">
          <template #default="{ row }">
            <el-tag
              size="small"
              :type="row.status === 'succeeded' ? 'success' : row.status === 'failed' ? 'danger' : row.status === 'skipped' ? 'warning' : 'info'"
            >
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
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useScanStore } from '@/stores/scan'
import type { StoredRun, NodeRunResult } from '@/types'

const scanStore = useScanStore()
const selectedRun = ref<StoredRun | null>(null)

const nodeResults = computed<NodeRunResult[]>(() => {
  if (!selectedRun.value) return []
  const order = selectedRun.value.summary.execution_order || []
  const results = selectedRun.value.summary.node_results || {}
  // Show in execution order first, then any remaining.
  const ordered: NodeRunResult[] = []
  for (const id of order) {
    if (results[id]) ordered.push(results[id])
  }
  for (const [id, result] of Object.entries(results)) {
    if (!order.includes(id)) ordered.push(result)
  }
  return ordered
})

function selectRun(run: StoredRun) {
  selectedRun.value = run
}

function formatDate(iso: string) {
  if (!iso) return '-'
  return new Date(iso).toLocaleString()
}

onMounted(() => {
  scanStore.loadRuns()
})
</script>

<style scoped>
.run-history h1 {
  color: #e5e5e5;
  margin-bottom: 20px;
}
</style>
