<template>
  <div class="config-view">
    <div class="config-header">
      <div>
        <h1>Config</h1>
        <p class="config-subtitle">统一配置框架使用的外部工具路径和全局代理。</p>
      </div>
      <el-button :loading="loading" @click="loadConfig">Refresh</el-button>
    </div>

    <el-card shadow="hover" class="config-card">
      <template #header>
        <div class="card-header">
          <span>Runtime Config</span>
          <el-tag type="info" size="small">{{ workflowPath || 'workflow not loaded' }}</el-tag>
        </div>
      </template>

      <el-form label-width="120px" v-loading="loading">
        <el-form-item v-for="item in items" :key="item.var_name" :label="item.label">
          <el-input v-model="form[item.var_name]" :placeholder="item.default || 'path'" clearable />
          <div class="field-meta">
            <span>{{ item.tool }}</span>
            <span>{{ item.var_name }}</span>
          </div>
        </el-form-item>

        <el-form-item v-if="items.length === 0">
          <div class="empty-hint">No configurable tool paths found.</div>
        </el-form-item>

        <el-form-item>
          <el-button type="primary" :loading="saving" @click="saveConfig">Save</el-button>
          <el-button @click="resetConfig">Reset</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card shadow="hover" class="config-card">
      <template #header>
        <span>Runtime Compatibility</span>
      </template>
      <ul class="compat-list">
        <li>`.exe` 路径会走 `powershell.exe`，实际执行的是 Windows 侧工具。</li>
        <li>普通 Linux 可执行文件会走 WSL shell，实际执行的是 WSL 侧工具。</li>
        <li>所以当前框架是两者兼容，不强绑单一侧。</li>
        <li>`HTTP Proxy` 会自动注入 `HTTP_PROXY` / `HTTPS_PROXY`。</li>
        <li>`SOCKS5 Proxy` 会自动注入 `ALL_PROXY`。</li>
      </ul>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { fetchToolPathConfig, updateToolPathConfig, type ToolPathConfigItem } from '@/api/config'

const loading = ref(false)
const saving = ref(false)
const workflowPath = ref('')
const items = ref<ToolPathConfigItem[]>([])
const form = ref<Record<string, string>>({})

function hydrateForm(entries: ToolPathConfigItem[]) {
  const next: Record<string, string> = {}
  entries.forEach((item) => {
    next[item.var_name] = item.value || item.default || ''
  })
  form.value = next
}

async function loadConfig() {
  loading.value = true
  try {
    const response = await fetchToolPathConfig()
    workflowPath.value = response.workflow_path
    items.value = response.items || []
    hydrateForm(items.value)
  } catch (e: any) {
    ElMessage.error(`Failed to load config: ${e.message}`)
  } finally {
    loading.value = false
  }
}

function resetConfig() {
  hydrateForm(items.value)
}

async function saveConfig() {
  saving.value = true
  try {
    const payload: Record<string, string> = {}
    items.value.forEach((item) => {
      payload[item.var_name] = String(form.value[item.var_name] || '').trim()
    })
    const response = await updateToolPathConfig(payload)
    workflowPath.value = response.workflow_path
    items.value = response.items || []
    hydrateForm(items.value)
    ElMessage.success('Config saved')
  } catch (e: any) {
    ElMessage.error(`Failed to save config: ${e.message}`)
  } finally {
    saving.value = false
  }
}

onMounted(loadConfig)
</script>

<style scoped>
.config-view {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.config-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
}

.config-header h1 {
  margin: 0;
  color: #e5e5e5;
}

.config-subtitle {
  margin-top: 8px;
  color: #909399;
}

.config-card {
  border-radius: 10px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

.field-meta {
  display: flex;
  gap: 12px;
  margin-top: 6px;
  font-size: 12px;
  color: #909399;
}

.compat-list {
  margin-left: 18px;
  color: #c0c4cc;
  line-height: 1.8;
}

.empty-hint {
  color: #909399;
}
</style>
