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

        <!-- Host Details -->
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

              <!-- Port List -->
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
                      <el-tag v-for="vuln in row.vulns.slice(0, 2)" :key="vuln.id"
                        size="small" :type="vulnSeverityType(classifyVulnerability(vuln))">
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

        <!-- Vulnerability Summary -->
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
                <el-button v-if="row.exp_available" size="small" type="danger" @click="exploitVuln(row)">
                  Exploit
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>

        <div v-if="!assetStore.loading && assetStore.hosts.length === 0" class="empty-hint">
          No assets discovered yet. Run a scan to discover assets.
        </div>
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
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { getProjectTargets, type Target } from '@/api/target'
import { fetchLatestTargetRun, startScan as startScanAPI } from '@/api/scan'
import { listExploits, triggerExploit, type ExploitInfo } from '@/api/exploit'
import { useAssetStore } from '@/stores/asset'

const route = useRoute()
const router = useRouter()
const assetStore = useAssetStore()

const projectId = computed(() => route.params.id as string)
const targetId = computed(() => route.params.tid as string)
const currentTarget = ref<Target | null>(null)
const activeTab = ref('scan')
const scanning = ref(false)
const scanResult = ref<any>(null)
const exploits = ref<ExploitInfo[]>([])

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

async function exploitVuln(vuln: any) {
  if (!currentTarget.value) {
    ElMessage.warning('No target selected')
    return
  }

  try {
    const result = await triggerExploit({
      target: vuln.exploit_target || vuln.target || currentTarget.value.url,
      exploit_id: vuln.exploit_id || 'auto',
      mode: 'execute',
      command: 'id',
      project_id: projectId.value,
      target_id: targetId.value
    })

    if (result.result?.success) {
      ElMessage.success({
        message: `Exploit successful! Output: ${result.result.output || 'Command executed'}`,
        duration: 5000
      })
    } else {
      const detail = result.result?.detail || result.message || 'No output detected'
      ElMessage.warning({
        message: `Exploit attempted but no confirmed output: ${detail}`,
        duration: 5000
      })
    }
  } catch (e: any) {
    ElMessage.error(`Exploit failed: ${e.message}`)
  }
}

function goBack() {
  router.push(`/projects/${projectId.value}`)
}

async function loadTarget() {
  try {
    const targets = await getProjectTargets(projectId.value)
    currentTarget.value = targets.find(t => t.id === targetId.value) || null
  } catch (e: any) {
    ElMessage.error(`Failed to load target: ${e.message}`)
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
      target_id: targetId.value
    })
    scanResult.value = result
    await assetStore.load(result.run_id)
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
    low: 'info'
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
      target_id: targetId.value
    })

    if (result.result?.vulnerable) {
      ElMessage.success(`Vulnerable! ${result.result.detail || ''}`)
    } else {
      ElMessage.info(`Not vulnerable: ${result.result?.detail || 'Check completed'}`)
    }
  } catch (e: any) {
    ElMessage.error(`Check failed: ${e.message}`)
  }
}

async function runExploit(exploit: ExploitInfo) {
  if (!currentTarget.value) {
    ElMessage.warning('No target selected')
    return
  }

  try {
    const result = await triggerExploit({
      target: currentTarget.value.url,
      exploit_id: exploit.id,
      mode: 'execute',
      command: 'id',
      project_id: projectId.value,
      target_id: targetId.value
    })

    if (result.result?.success) {
      ElMessage.success(`Exploit successful! Output: ${result.result.output || ''}`)
    } else {
      ElMessage.warning(`Exploit failed: ${result.result?.detail || result.message || 'No output'}`)
    }
  } catch (e: any) {
    ElMessage.error(`Exploit failed: ${e.message}`)
  }
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
</style>
