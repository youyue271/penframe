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
                <div class="stat-number" :class="{ 'has-vulns': assetStore.summary.vulns > 0 }">
                  {{ assetStore.summary.vulns }}
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
                        size="small" :type="vulnSeverityType(vuln.severity)">
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
              <el-tag type="danger">{{ allVulnerabilities.length }} found</el-tag>
            </div>
          </template>
          <el-table :data="allVulnerabilities" size="small" stripe>
            <el-table-column label="Severity" width="100">
              <template #default="{ row }">
                <el-tag size="small" :type="vulnSeverityType(row.severity)">
                  {{ row.severity || 'unknown' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="name" label="Name" width="200" />
            <el-table-column prop="target" label="Target" show-overflow-tooltip />
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
import { listExploits, type ExploitInfo } from '@/api/exploit'
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

const allVulnerabilities = computed(() => {
  const vulns: any[] = []
  assetStore.hosts.forEach((host: any) => {
    host.ports?.forEach((port: any) => {
      port.vulns?.forEach((vuln: any) => {
        vulns.push({ ...vuln, host: host.ip, port: port.port })
      })
      port.paths?.forEach((path: any) => {
        path.vulns?.forEach((vuln: any) => {
          vulns.push({ ...vuln, host: host.ip, port: port.port, path: path.path })
        })
      })
    })
  })
  return vulns
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

function vulnSeverityType(severity: string): string {
  const s = (severity || '').toLowerCase()
  if (s === 'critical' || s === 'high') return 'danger'
  if (s === 'medium') return 'warning'
  return 'info'
}

function viewPortDetails(host: any, port: any) {
  ElMessage.info(`Port ${port.port} on ${host.ip}`)
}

function exploitVuln(vuln: any) {
  ElMessage.warning(`Exploit: ${vuln.name}`)
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

function checkExploit(exploit: ExploitInfo) {
  ElMessage.info(`Check exploit: ${exploit.name}`)
}

function runExploit(exploit: ExploitInfo) {
  ElMessage.warning(`Run exploit: ${exploit.name}`)
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

.empty-hint {
  text-align: center;
  padding: 40px;
  color: #909399;
}
</style>
