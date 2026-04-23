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
        <div class="card-header">
          <span>VShell Integration</span>
          <el-tag :type="vshellConfig.enabled ? 'success' : 'info'" size="small">
            {{ vshellConfig.enabled ? 'Enabled' : 'Disabled' }}
          </el-tag>
        </div>
      </template>

      <el-form label-width="140px" v-loading="vshellLoading">
        <el-form-item label="Enable VShell">
          <el-switch v-model="vshellConfig.enabled" />
          <div class="field-hint">Enable reverse shell integration with vshell C2 platform</div>
        </el-form-item>

        <el-form-item label="Listener Host">
          <el-input v-model="vshellConfig.host" placeholder="127.0.0.1 or your_ip" />
          <div class="field-hint">VShell listener host (where shells will connect back to)</div>
        </el-form-item>

        <el-form-item label="Listener Port">
          <el-input-number v-model="vshellConfig.port" :min="1" :max="65535" />
          <div class="field-hint">VShell listener port (configure this in vshell first)</div>
        </el-form-item>

        <el-form-item label="VShell Web URL">
          <el-input v-model="vshellConfig.web_url" placeholder="http://localhost:8082" />
          <div class="field-hint">VShell web interface URL (for iframe embedding)</div>
        </el-form-item>

        <el-form-item>
          <el-button type="primary" :loading="vshellSaving" @click="saveVShellConfig">Save VShell Config</el-button>
          <el-button @click="loadVShellConfig">Reset</el-button>
        </el-form-item>
      </el-form>

      <el-alert type="info" :closable="false" style="margin-top: 16px;">
        <template #title>How to use VShell integration</template>
        <ol style="margin: 8px 0; padding-left: 20px;">
          <li>Start vshell and create a listener on the configured port</li>
          <li>Configure the listener host and port above</li>
          <li>When executing RCE exploits, penframe will generate reverse shell payloads</li>
          <li>The target will connect back to your vshell listener</li>
          <li>Manage the shell in vshell's web interface</li>
        </ol>
      </el-alert>
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
import { getVShellConfig, updateVShellConfig, type VShellConfig } from '@/api/vshell'

const loading = ref(false)
const saving = ref(false)
const workflowPath = ref('')
const items = ref<ToolPathConfigItem[]>([])
const form = ref<Record<string, string>>({})

const vshellLoading = ref(false)
const vshellSaving = ref(false)
const vshellConfig = ref<VShellConfig>({
  host: '127.0.0.1',
  port: 4444,
  enabled: false,
  web_url: 'http://localhost:8082',
})

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

async function loadVShellConfig() {
  vshellLoading.value = true
  try {
    const config = await getVShellConfig()
    vshellConfig.value = config
  } catch (e: any) {
    ElMessage.error(`Failed to load vshell config: ${e.message}`)
  } finally {
    vshellLoading.value = false
  }
}

async function saveVShellConfig() {
  vshellSaving.value = true
  try {
    const updated = await updateVShellConfig(vshellConfig.value)
    vshellConfig.value = updated
    ElMessage.success('VShell config saved')
  } catch (e: any) {
    ElMessage.error(`Failed to save vshell config: ${e.message}`)
  } finally {
    vshellSaving.value = false
  }
}

onMounted(() => {
  loadConfig()
  loadVShellConfig()
})
</script>

<style scoped>
.config-view {
  display: flex;
  flex-direction: column;
  gap: 24px;
  animation: fadeIn 0.5s ease-out;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.config-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 24px;
  background: linear-gradient(135deg, rgba(0, 217, 255, 0.05), rgba(168, 85, 247, 0.05));
  border: 1px solid var(--pf-border);
  border-radius: var(--pf-radius-lg);
  box-shadow: var(--pf-shadow-md);
}

.config-header h1 {
  margin: 0;
  font-family: 'Orbitron', 'JetBrains Mono', monospace;
  color: var(--pf-text-primary);
  font-size: 28px;
  font-weight: 700;
  letter-spacing: 2px;
  text-transform: uppercase;
  background: linear-gradient(135deg, #88a8b8, var(--pf-text-primary));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.config-subtitle {
  margin-top: 8px;
  color: var(--pf-text-secondary);
  font-family: 'JetBrains Mono', monospace;
  font-size: 13px;
  letter-spacing: 0.5px;
}

.config-card {
  border-radius: var(--pf-radius-lg);
  background: var(--pf-bg-elevated);
  border: 1px solid var(--pf-border);
  box-shadow: var(--pf-shadow-md);
  transition: all 0.3s ease;
}

.config-card:hover {
  border-color: rgba(0, 217, 255, 0.3);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.6), 0 0 20px rgba(0, 217, 255, 0.1);
}

.config-card :deep(.el-card__header) {
  background: linear-gradient(135deg, rgba(0, 217, 255, 0.08), rgba(168, 85, 247, 0.08));
  border-bottom: 1px solid var(--pf-border);
  padding: 16px 20px;
  font-family: 'Orbitron', 'JetBrains Mono', monospace;
  font-weight: 700;
  font-size: 14px;
  letter-spacing: 1px;
  text-transform: uppercase;
  color: var(--pf-accent-cyan);
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
  font-size: 11px;
  color: var(--pf-text-muted);
  margin-top: 4px;
  font-family: 'JetBrains Mono', monospace;
  letter-spacing: 0.5px;
}

.field-hint {
  font-size: 12px;
  color: var(--pf-text-secondary);
  margin-top: 4px;
  font-family: 'JetBrains Mono', monospace;
  letter-spacing: 0.3px;
}

.compat-list {
  margin-left: 18px;
  color: var(--pf-text-secondary);
  line-height: 1.8;
  font-family: 'JetBrains Mono', monospace;
  font-size: 13px;
}

.compat-list li {
  margin: 8px 0;
  padding-left: 8px;
  border-left: 2px solid var(--pf-accent-cyan);
}

.empty-hint {
  color: var(--pf-text-muted);
  font-family: 'JetBrains Mono', monospace;
  text-align: center;
  padding: 20px;
}

:deep(.el-form-item__label) {
  font-family: 'JetBrains Mono', monospace;
  color: var(--pf-text-secondary);
  font-weight: 600;
  font-size: 12px;
  letter-spacing: 0.5px;
}

:deep(.el-input__wrapper) {
  background: var(--pf-bg-surface);
  border: 1px solid var(--pf-border);
  border-radius: var(--pf-radius);
  box-shadow: inset 0 2px 4px rgba(0, 0, 0, 0.3);
  transition: all 0.3s ease;
}

:deep(.el-input__wrapper:hover) {
  border-color: var(--pf-border-light);
}

:deep(.el-input__wrapper.is-focus) {
  border-color: var(--pf-accent-cyan);
  box-shadow: 0 0 0 2px rgba(0, 217, 255, 0.2), inset 0 2px 4px rgba(0, 0, 0, 0.3);
}

:deep(.el-input__inner) {
  font-family: 'JetBrains Mono', monospace;
  color: var(--pf-text-primary);
  font-size: 13px;
}

:deep(.el-switch) {
  --el-switch-on-color: var(--pf-accent-cyan);
}

:deep(.el-input-number) {
  font-family: 'JetBrains Mono', monospace;
}

:deep(.el-alert) {
  background: rgba(0, 217, 255, 0.05);
  border: 1px solid rgba(0, 217, 255, 0.2);
  border-radius: var(--pf-radius);
}

:deep(.el-alert__title) {
  font-family: 'JetBrains Mono', monospace;
  color: var(--pf-accent-cyan);
  font-weight: 600;
  font-size: 13px;
}

:deep(.el-alert ol) {
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
  color: var(--pf-text-secondary);
  line-height: 1.8;
}
</style>
