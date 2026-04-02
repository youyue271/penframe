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
          <span>Scan Tasks</span>
          <el-button size="small" @click="scanStore.loadTasks()">Refresh</el-button>
        </div>
      </template>
      <el-table :data="scanStore.tasks" style="width: 100%" size="small" stripe>
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
import { ref, onMounted } from 'vue'
import { useScanStore } from '@/stores/scan'
import { ElMessage } from 'element-plus'

const scanStore = useScanStore()

const form = ref({
  target: '',
  strategy: 'full',
  timeout: 1800,
})

const lastRunId = ref('')

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

onMounted(() => {
  scanStore.loadTasks()
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
</style>
