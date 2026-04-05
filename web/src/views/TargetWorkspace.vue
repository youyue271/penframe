<template>
  <div class="target-workspace">
    <h1>Target Workspace</h1>

    <el-card shadow="hover" class="workspace-card">
      <template #header>
        <span>Target Configuration</span>
      </template>
      <el-form label-width="140px" @submit.prevent="startScan">
        <el-form-item label="Target Address">
          <el-input
            v-model="targetForm.target"
            placeholder="IP / CIDR / Domain / URL (e.g. https://target:3000)"
            clearable
            @keyup.enter="startScan"
          />
        </el-form-item>

        <el-form-item label="Workflow">
          <div class="workflow-meta">
            <el-tag type="info" size="small">
              {{ workflowState?.workflow.name || 'Loading workflow...' }}
            </el-tag>
            <span class="workflow-description">
              {{ workflowState?.workflow.description || 'Using current backend workflow definition.' }}
            </span>
          </div>
        </el-form-item>

        <el-form-item label="Node Presets">
          <div class="workflow-preset-compact">
            <el-collapse v-if="presetSections.length" v-model="activePresetSections">
              <el-collapse-item
                v-for="section in presetSections"
                :key="section.key"
                :name="section.key"
              >
                <template #title>
                  <div class="section-title-row">
                    <div class="section-title-left">
                      <span class="section-title">{{ section.label }}</span>
                      <el-tag size="small" type="info">{{ section.presets.length }}</el-tag>
                    </div>
                    <span class="section-description">{{ section.description }}</span>
                  </div>
                </template>

                <div class="section-content">
                  <div v-if="section.presets.length" class="section-preset-list">
                    <div
                      v-for="preset in section.presets"
                      :key="preset.node.id"
                      class="preset-panel"
                    >
                      <div class="preset-title-row">
                        <div class="preset-title-left">
                          <span class="preset-node-id">{{ preset.node.id }}</span>
                          <el-tag size="small" :type="categoryTagType(preset.tool?.category)">
                            {{ preset.node.tool }}
                          </el-tag>
                          <el-tag size="small" type="info">
                            {{ preset.node.executor }}
                          </el-tag>
                        </div>
                        <el-switch
                          v-if="preset.enableVar && nodeSettings[preset.node.id]"
                          v-model="nodeSettings[preset.node.id].enabled"
                          size="small"
                        />
                      </div>

                      <div class="preset-description">
                        {{ preset.tool?.description || `Executor: ${preset.node.executor}` }}
                      </div>

                      <div class="preset-command-block">
                        <div class="preset-command-label">Command</div>
                        <pre class="preset-command">{{ commandPreview(preset) }}</pre>
                      </div>

                      <div v-if="preset.fields.length && nodeSettings[preset.node.id]" class="preset-params">
                        <div v-for="field in preset.fields" :key="`${preset.node.id}-${field.name}`" class="preset-param">
                          <label class="preset-param-label">{{ field.label }}</label>
                          <el-input
                            v-model="nodeSettings[preset.node.id].params[field.name]"
                            type="textarea"
                            :autosize="{ minRows: 1, maxRows: 3 }"
                            size="small"
                          />
                          <div v-if="field.description" class="preset-param-hint">
                            {{ field.description }}
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                  <div v-else class="empty-hint section-empty">
                    Current workflow does not define nodes in this section.
                  </div>
                </div>
              </el-collapse-item>
            </el-collapse>

            <div v-else class="empty-hint">
              Loading workflow node presets...
            </div>
          </div>
        </el-form-item>

        <el-form-item label="Timeout (s)">
          <el-input-number v-model="targetForm.timeout" :min="30" :max="86400" :step="60" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" size="large" :loading="scanStore.scanning" @click="startScan">
            Start Scan
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card v-if="scanStore.currentRunId" shadow="hover" class="workspace-card">
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center;">
          <span>
            Scan Pipeline
            <el-tag
              v-if="scanStore.currentRun"
              size="small"
              :type="runStatusTag(scanStore.currentRun.summary.status)"
              style="margin-left: 8px;"
            >
              {{ scanStore.currentRun.summary.status }}
            </el-tag>
            <el-tag v-else size="small" type="info" style="margin-left: 8px;">
              Loading...
            </el-tag>
          </span>
          <el-button size="small" @click="refreshCurrentRun">Refresh</el-button>
        </div>
      </template>

      <el-table :data="currentRunTasks" style="width: 100%" size="small" stripe>
        <el-table-column prop="node_id" label="Node" width="180" show-overflow-tooltip />
        <el-table-column prop="type" label="Tool" width="180">
          <template #default="{ row }">
            <el-tag size="small" :type="taskTagType(row.type)">
              {{ row.type }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="target" label="Target" min-width="220" show-overflow-tooltip />
        <el-table-column prop="status" label="Status" width="100">
          <template #default="{ row }">
            <el-tag size="small" :type="taskStatusTag(row.status)">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="error" label="Error" show-overflow-tooltip />
      </el-table>

      <el-divider v-if="scanStore.currentRun && nodeResults.length" content-position="left">Workflow Nodes</el-divider>

      <el-table v-if="scanStore.currentRun && nodeResults.length" :data="nodeResults" style="width: 100%" size="small" stripe>
        <el-table-column prop="node_id" label="Phase" width="180" />
        <el-table-column prop="tool" label="Tool" width="160" />
        <el-table-column prop="status" label="Status" width="100">
          <template #default="{ row }">
            <el-tag size="small" :type="nodeStatusTag(row.status)">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="Command" min-width="300">
          <template #default="{ row }">
            <code v-if="row.rendered_command" class="cmd-text">{{ row.rendered_command }}</code>
            <span v-else class="cmd-text cmd-na">-</span>
          </template>
        </el-table-column>
        <el-table-column prop="duration_millis" label="Duration" width="90">
          <template #default="{ row }">
            {{ row.duration_millis ? `${(row.duration_millis / 1000).toFixed(1)}s` : '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="record_count" label="Findings" width="80" />
        <el-table-column prop="error" label="Error" show-overflow-tooltip />
      </el-table>
      <div v-else style="text-align: center; padding: 20px; color: #909399;">
        Waiting for workflow node results...
      </div>

      <el-descriptions v-if="scanStore.currentRun" :column="3" border size="small" style="margin-top: 12px;">
        <el-descriptions-item label="Run ID">{{ scanStore.currentRun.id }}</el-descriptions-item>
        <el-descriptions-item label="Started">{{ formatDate(scanStore.currentRun.summary.started_at) }}</el-descriptions-item>
        <el-descriptions-item label="Finished">{{ formatDate(scanStore.currentRun.summary.finished_at) }}</el-descriptions-item>
      </el-descriptions>
    </el-card>

    <el-card v-if="directOutputs.length" shadow="hover" class="workspace-card">
      <template #header>
        <span>Scan Output</span>
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

    <el-card shadow="hover" class="workspace-card">
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center;">
          <span>Exploit</span>
          <el-button size="small" @click="loadExploits" :loading="loadingExploits">Refresh Modules</el-button>
        </div>
      </template>

      <div v-if="exploits.length" class="exploit-section">
        <el-table :data="exploits" style="width: 100%" size="small" stripe>
          <el-table-column prop="id" label="Module" width="180" />
          <el-table-column prop="name" label="Name" show-overflow-tooltip />
          <el-table-column prop="cve" label="CVE" width="160" />
          <el-table-column label="Capabilities" width="220">
            <template #default="{ row }">
              <div class="capability-tags">
                <el-tag size="small" type="info" v-if="row.supports_check !== false">check</el-tag>
                <el-tag size="small" type="danger" v-if="row.supports_execute">execute</el-tag>
                <el-tag size="small" type="success" v-if="row.supports_command">echo</el-tag>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="severity" label="Severity" width="90">
            <template #default="{ row }">
              <el-tag size="small" :type="row.severity === 'critical' ? 'danger' : row.severity === 'high' ? 'warning' : 'info'">
                {{ row.severity }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="Actions" width="200">
            <template #default="{ row }">
              <el-button size="small" type="primary" @click="doCheck(row)" :loading="executingExploit">
                Check
              </el-button>
              <el-button size="small" type="danger" @click="doExploit(row)" :loading="executingExploit" :disabled="row.supports_execute === false">
                Exploit
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
      <div v-else class="empty-hint">
        No exploit modules loaded. Ensure the Python exp service is running.
      </div>
    </el-card>

    <el-card v-if="exploitResult" shadow="hover" class="workspace-card">
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center;">
          <span>Exploit Result</span>
          <el-tag size="small" :type="exploitStatusTag(exploitResult?.status || exploitResult?.result?.status || '')">
            {{ exploitResult?.status || exploitResult?.result?.status || 'unknown' }}
          </el-tag>
        </div>
      </template>

      <el-descriptions :column="2" border size="small">
        <el-descriptions-item label="Mode">{{ exploitResultMode || '-' }}</el-descriptions-item>
        <el-descriptions-item label="Accepted">{{ formatResultValue(exploitResult?.accepted) }}</el-descriptions-item>
        <el-descriptions-item label="Status">{{ formatResultValue(exploitResult?.status) }}</el-descriptions-item>
        <el-descriptions-item label="Request ID">{{ formatResultValue(exploitResult?.request_id) }}</el-descriptions-item>
      </el-descriptions>

      <div v-if="exploitResultMode === 'check' && exploitResult?.result" class="exploit-result-section">
        <div class="exploit-result-title">Check Summary</div>
        <el-descriptions :column="3" border size="small">
          <el-descriptions-item label="Vulnerable">{{ formatResultValue(exploitResult.result.vulnerable) }}</el-descriptions-item>
          <el-descriptions-item label="Confidence">{{ formatConfidence(exploitResult.result.confidence) }}</el-descriptions-item>
          <el-descriptions-item label="Detail">{{ formatResultValue(exploitResult.result.detail) }}</el-descriptions-item>
        </el-descriptions>
      </div>

      <div v-if="exploitExecutionOutput" class="exploit-result-section">
        <div class="exploit-result-title">Execution Output</div>
        <pre class="result-output">{{ exploitExecutionOutput }}</pre>
      </div>

      <div v-else-if="exploitExecutionDetail" class="exploit-result-section">
        <div class="exploit-result-title">Execution Detail</div>
        <pre class="result-output">{{ exploitExecutionDetail }}</pre>
      </div>

      <div v-if="exploitEvidence" class="exploit-result-section">
        <div class="exploit-result-title">Evidence / Artifacts</div>
        <pre class="result-output">{{ JSON.stringify(exploitEvidence, null, 2) }}</pre>
      </div>

      <div class="exploit-result-section">
        <div class="exploit-result-title">Raw JSON</div>
        <pre class="result-output">{{ JSON.stringify(exploitResult, null, 2) }}</pre>
      </div>
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
import { listExploits, triggerExploit } from '@/api/exploit'
import { fetchState } from '@/api/scan'
import { useScanStore } from '@/stores/scan'
import { useSSEStore } from '@/stores/sse'
import type {
  ExploitInfo,
  NodeRunResult,
  PortalStateResponse,
  ScanTask,
  ToolDefinition,
  WorkflowNodeDefinition,
} from '@/types'

type DirectOutput = {
  key: string
  title: string
  status: string
  content: string
}

type WorkflowPresetField = {
  name: string
  label: string
  description: string
}

type WorkflowPreset = {
  node: WorkflowNodeDefinition
  tool?: ToolDefinition
  enableVar?: string
  fields: WorkflowPresetField[]
}

type PresetSectionKey = 'host' | 'port' | 'path' | 'vuln'

type PresetSection = {
  key: PresetSectionKey
  label: string
  description: string
  presets: WorkflowPreset[]
}

type NodeSetting = {
  enabled: boolean
  params: Record<string, string>
}

const targetForm = ref({
  target: '',
  timeout: 1800,
})

const derivedVarNames = new Set([
  'target',
  'target_url',
  'target_host',
  'target_hostport',
  'target_port',
  'target_scheme',
  'target_origin',
  'target_path',
  'output_root',
  'output_target',
  'output_dir',
  'output_dir_windows',
])

const scanStore = useScanStore()
const sseStore = useSSEStore()

const workflowState = ref<PortalStateResponse | null>(null)
const workflowPresets = ref<WorkflowPreset[]>([])
const nodeSettings = ref<Record<string, NodeSetting>>({})
const activePresetSections = ref<PresetSectionKey[]>(['host', 'port', 'path', 'vuln'])

const exploits = ref<ExploitInfo[]>([])
const loadingExploits = ref(false)
const executingExploit = ref(false)
const exploitResult = ref<any>(null)
const exploitResultMode = ref<'check' | 'execute' | ''>('')
let unsubscribe: (() => void) | null = null

const exploitExecutionOutput = computed(() => exploitResult.value?.result?.output || exploitResult.value?.output || '')
const exploitExecutionDetail = computed(() => exploitResult.value?.result?.detail || exploitResult.value?.message || '')
const exploitEvidence = computed(() => exploitResult.value?.result?.evidence || exploitResult.value?.result?.artifacts || null)

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
        title: `${result.node_id} · parsed records (${result.records.length})`,
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

const currentRunTasks = computed<ScanTask[]>(() => {
  if (!scanStore.currentRunId) return scanStore.tasks
  return scanStore.tasks.filter((task) => task.parent_id === scanStore.currentRunId)
})

const presetSections = computed<PresetSection[]>(() => {
  const base: Array<Omit<PresetSection, 'presets'>> = [
    { key: 'host', label: '主机发现', description: '目标初始化、主机识别与范围种子。' },
    { key: 'port', label: '端口发现', description: '端口、服务与基础服务探测。' },
    { key: 'path', label: '路径发现', description: '入口、路径、页面与 Web 面探测。' },
    { key: 'vuln', label: '漏洞发现', description: '指纹、漏洞与后续利用相关节点。' },
  ]

  return base.map((item) => ({
    ...item,
    presets: workflowPresets.value.filter((preset) => classifyPresetSection(preset) === item.key),
  }))
})

function collectVarRefs(value: any, refs: Set<string>) {
  if (typeof value === 'string') {
    const pattern = /\.vars\.([A-Za-z0-9_]+)/g
    let match: RegExpExecArray | null
    while ((match = pattern.exec(value)) !== null) {
      refs.add(match[1])
    }
    return
  }
  if (Array.isArray(value)) {
    value.forEach((item) => collectVarRefs(item, refs))
    return
  }
  if (value && typeof value === 'object') {
    Object.values(value).forEach((item) => collectVarRefs(item, refs))
  }
}

function formatVarLabel(name: string) {
  return name.replace(/_/g, ' ').replace(/\b\w/g, (letter) => letter.toUpperCase())
}

function includesAny(text: string, words: string[]) {
  return words.some((word) => text.includes(word))
}

function classifyPresetSection(preset: WorkflowPreset): PresetSectionKey {
  const text = [
    preset.node.id,
    preset.node.tool,
    preset.tool?.category || '',
    preset.tool?.description || '',
  ].join(' ').toLowerCase()

  if (includesAny(text, ['cve', 'vuln', 'nuclei', 'fingerprint', 'executor', 'exploit', 'xray'])) {
    return 'vuln'
  }
  if (includesAny(text, ['path', 'entry', 'dirsearch', 'ffuf', 'http', 'web', 'page', 'url'])) {
    return 'path'
  }
  if (includesAny(text, ['port', 'nmap', 'masscan', 'service_discovery', 'service discovery'])) {
    return 'port'
  }
  return 'host'
}

function buildWorkflowPresets(state: PortalStateResponse) {
  const toolMap = new Map(state.tools.map((tool) => [tool.name, tool]))
  const globalVars = state.workflow.global_vars || {}
  const previousSettings = nodeSettings.value
  const presets: WorkflowPreset[] = []
  for (const node of state.workflow.nodes) {
    const tool = toolMap.get(node.tool)
    if (tool?.category === 'orchestration') {
      continue
    }

    const refs = new Set<string>()
    collectVarRefs(node.inputs || {}, refs)
    const enableVar = refs.has(`run_${node.id}`)
      ? `run_${node.id}`
      : [...refs].find((name) => name.startsWith('run_'))

    const fields = [...refs]
      .filter((name) => name !== enableVar && !derivedVarNames.has(name) && globalVars[name] !== undefined)
      .map((name) => ({
        name,
        label: formatVarLabel(name),
        description: String(tool?.variables?.find((item) => item.name === name)?.description || ''),
      }))

    presets.push({
      node,
      tool,
      enableVar,
      fields,
    })
  }
  workflowPresets.value = presets

  const nextSettings: Record<string, NodeSetting> = {}
  for (const preset of workflowPresets.value) {
    const previous = previousSettings[preset.node.id]
    const params: Record<string, string> = {}
    for (const field of preset.fields) {
      params[field.name] = previous?.params?.[field.name] ?? String(globalVars[field.name] ?? '')
    }
    nextSettings[preset.node.id] = {
      enabled: previous?.enabled ?? Boolean(globalVars[preset.enableVar || ''] ?? true),
      params,
    }
  }
  nodeSettings.value = nextSettings
}

async function loadWorkflowState() {
  try {
    const response = await fetchState()
    workflowState.value = response
    if (!targetForm.value.target.trim()) {
      targetForm.value.target = String(response.workflow.global_vars?.target || '')
    }
    buildWorkflowPresets(response)
  } catch (e: any) {
    ElMessage.warning(`Could not load workflow state: ${e.message}`)
  }
}

function buildScanVars() {
  const vars: Record<string, any> = {}
  for (const preset of workflowPresets.value) {
    const setting = nodeSettings.value[preset.node.id]
    if (!setting) continue
    if (preset.enableVar) {
      vars[preset.enableVar] = setting.enabled
    }
    for (const field of preset.fields) {
      const value = setting.params[field.name]?.trim?.() ?? setting.params[field.name]
      if (value !== '') {
        vars[field.name] = value
      }
    }
  }
  return vars
}

function commandPreview(preset: WorkflowPreset) {
  return preset.tool?.command_template?.trim() || `Executor: ${preset.node.executor}`
}

function requireTarget(): boolean {
  if (!targetForm.value.target.trim()) {
    ElMessage.warning('Please enter a target')
    return false
  }
  return true
}

async function startScan() {
  if (!requireTarget()) return
  try {
    const resp = await scanStore.scan({
      target: targetForm.value.target.trim(),
      strategy: 'custom',
      vars: buildScanVars(),
      timeout_seconds: targetForm.value.timeout,
    })
    ElMessage.success(`Scan started: ${resp.run_id}`)
    await refreshCurrentRun()
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

async function doCheck(exp: ExploitInfo) {
  if (!requireTarget()) return
  executingExploit.value = true
  try {
    const resp = await triggerExploit({
      target: targetForm.value.target.trim(),
      exploit_id: exp.id,
      mode: 'check',
    })
    exploitResult.value = resp
    exploitResultMode.value = 'check'
    ElMessage.success(`Check completed: ${exp.id}`)
  } catch (e: any) {
    ElMessage.error(e.message)
  } finally {
    executingExploit.value = false
  }
}

async function doExploit(exp: ExploitInfo) {
  if (!requireTarget()) return
  executingExploit.value = true
  try {
    const payload: any = {
      target: targetForm.value.target.trim(),
      exploit_id: exp.id,
      mode: 'execute',
    }
    if (exp.supports_command) {
      payload.command = exp.default_command || 'id'
    }
    const resp = await triggerExploit(payload)
    exploitResult.value = resp
    exploitResultMode.value = 'execute'
    ElMessage.success(`Exploit executed: ${exp.id}`)
  } catch (e: any) {
    ElMessage.error(e.message)
  } finally {
    executingExploit.value = false
  }
}

function formatResultValue(value: unknown) {
  if (value === undefined || value === null || value === '') return '-'
  if (typeof value === 'boolean') return value ? 'true' : 'false'
  return String(value)
}

function formatConfidence(value: unknown) {
  if (typeof value !== 'number') return '-'
  return value.toFixed(2)
}

function exploitStatusTag(status: string): string {
  if (status === 'succeeded' || status === 'success') return 'success'
  if (status === 'failed' || status === 'error') return 'danger'
  if (status === 'running' || status === 'accepted') return 'warning'
  return 'info'
}

function runStatusTag(status: string): string {
  return status === 'succeeded' ? 'success' : status === 'failed' ? 'danger' : status === 'running' ? 'warning' : 'info'
}

function nodeStatusTag(status: string): string {
  return status === 'succeeded' ? 'success' : status === 'failed' ? 'danger' : status === 'skipped' ? 'warning' : status === 'running' ? '' : 'info'
}

function taskStatusTag(status: string): string {
  return status === 'done' ? 'success' : status === 'failed' ? 'danger' : status === 'running' ? 'warning' : status === 'skipped' ? 'info' : 'info'
}

function taskTagType(type: string): string {
  if (type.includes('nuclei') || type.includes('exploit')) return 'danger'
  if (type.includes('nmap') || type.includes('fscan')) return 'warning'
  if (type.includes('path') || type.includes('http')) return 'success'
  return 'info'
}

function categoryTagType(category?: string): string {
  if (!category) return ''
  if (category === 'vuln_scan' || category === 'exploit') return 'danger'
  if (category === 'port_scan' || category === 'host_discovery') return 'warning'
  if (category === 'path_scan' || category === 'web') return 'success'
  return 'info'
}

function formatDate(iso: string) {
  if (!iso) return '-'
  return new Date(iso).toLocaleString()
}

onMounted(() => {
  void (async () => {
    await Promise.all([loadWorkflowState(), loadExploits()])
    if (!scanStore.currentRunId) {
      await scanStore.loadLatestRun()
    }
    if (scanStore.currentRunId) {
      await refreshCurrentRun()
    } else {
      await scanStore.loadTasks()
    }
  })()

  unsubscribe = sseStore.onEvent((event) => {
    if (!scanStore.currentRunId || event.run_id !== scanStore.currentRunId) return
    if (['run_started', 'node_started', 'node_finished', 'run_finished', 'scan_error'].includes(event.type)) {
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

.workflow-meta {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.workflow-description {
  color: #909399;
  font-size: 13px;
}

.workflow-preset-compact {
  width: 100%;
}

.section-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  gap: 12px;
}

.section-title-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.section-title {
  font-weight: 600;
  color: #e5e5e5;
}

.section-description {
  color: #909399;
  font-size: 12px;
}

.section-content {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 8px 0 4px;
}

.section-preset-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.section-empty {
  padding: 16px 0;
}

.preset-panel {
  background: rgba(255, 255, 255, 0.02);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 6px;
  padding: 12px;
  transition: all 0.2s ease;
}

.preset-panel:hover {
  background: rgba(255, 255, 255, 0.04);
  border-color: rgba(255, 255, 255, 0.12);
}

.preset-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  gap: 12px;
}

.preset-title-left {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
}

.preset-node-id {
  font-weight: 500;
  color: #e5e5e5;
}

.preset-description {
  color: #909399;
  font-size: 13px;
  line-height: 1.5;
  margin-bottom: 12px;
}

.preset-command-block {
  margin-bottom: 12px;
}

.preset-command-label {
  color: #c0c4cc;
  font-size: 13px;
  font-weight: 500;
  margin-bottom: 6px;
}

.preset-command {
  margin: 0;
  padding: 12px;
  border-radius: 6px;
  background: #0d0d0d;
  color: #a0cfff;
  font-family: 'Consolas', 'Fira Code', monospace;
  font-size: 12px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-all;
}

.preset-params {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.preset-param {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.preset-param-label {
  color: #c0c4cc;
  font-size: 13px;
  font-weight: 500;
}

.preset-param-hint {
  color: #909399;
  font-size: 12px;
}

.capability-tags {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.exploit-result-section {
  margin-top: 16px;
}

.exploit-result-title {
  color: #c0c4cc;
  font-size: 13px;
  font-weight: 600;
  margin-bottom: 8px;
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

.empty-hint {
  color: #606266;
  text-align: center;
  padding: 30px;
}
</style>
