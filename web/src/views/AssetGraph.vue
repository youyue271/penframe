<template>
  <div class="asset-graph">
    <!-- Left sidebar for project/target selection -->
    <div class="sidebar">
      <div class="sidebar-section">
        <h3>Projects</h3>
        <el-select
          v-model="selectedProjectId"
          placeholder="Select Project"
          @change="onProjectChange"
          style="width: 100%"
        >
          <el-option
            v-for="project in projects"
            :key="project.id"
            :label="project.name"
            :value="project.id"
          />
        </el-select>
      </div>

      <div class="sidebar-section" v-if="selectedProjectId">
        <h3>Targets</h3>
        <div v-loading="loadingTargets" class="targets-list">
          <div
            v-for="target in targets"
            :key="target.id"
            :class="['target-item', { active: selectedTargetId === target.id }]"
            @click="selectTarget(target.id)"
          >
            <div class="target-name">{{ target.name }}</div>
            <div class="target-url">{{ target.url }}</div>
          </div>
          <div v-if="!loadingTargets && !targets.length" class="empty-hint">
            No targets yet
          </div>
        </div>
        <el-button @click="showAddTargetDialog = true" type="primary" size="small" style="width: 100%; margin-top: 12px">
          Add Target
        </el-button>
      </div>
    </div>

    <!-- Main content area -->
    <div class="main-content">
      <div class="graph-header">
        <h1>Asset Graph</h1>
        <div class="graph-controls">
          <el-button @click="refreshGraph" :loading="assetStore.loading" type="primary" size="small">
            Refresh
          </el-button>
          <el-button @click="fitView" size="small">Fit View</el-button>
          <el-button @click="showDetailPanel = !showDetailPanel" size="small">
            {{ showDetailPanel ? 'Hide' : 'Show' }} Details
          </el-button>
        </div>
      </div>

      <!-- Asset Details Panel -->
      <el-card v-if="showDetailPanel" class="details-panel" shadow="hover">
        <template #header>
          <div class="panel-header">
            <span>Asset Summary</span>
            <el-tag type="info" size="small">
              {{ assetStore.summary.hosts }}H / {{ assetStore.summary.ports }}P / {{ assetStore.summary.paths }}Pa / {{ assetStore.summary.vulns }}V
            </el-tag>
          </div>
        </template>

        <el-collapse v-model="activeCollapse">
          <el-collapse-item title="Hosts" name="hosts">
            <div v-if="assetStore.hosts.length > 0" class="host-list">
              <div v-for="host in assetStore.hosts" :key="host.id" class="host-item">
                <div class="host-header">
                  <el-tag type="success" size="small">{{ host.ip }}</el-tag>
                  <span class="host-domain" v-if="host.hostname">{{ host.hostname }}</span>
                </div>
                <div class="host-stats">
                  <el-tag size="small" type="warning">{{ host.ports?.length || 0 }} ports</el-tag>
                  <el-tag size="small" type="info">{{ countPaths(host) }} paths</el-tag>
                  <el-tag size="small" type="danger" v-if="countVulns(host) > 0">{{ countVulns(host) }} vulns</el-tag>
                </div>
                <div v-if="host.ports && host.ports.length > 0" class="port-list">
                  <div v-for="port in host.ports.slice(0, 5)" :key="port.id" class="port-item">
                    <span class="port-number">{{ port.port }}</span>
                    <span class="port-service">{{ port.service || 'unknown' }}</span>
                    <el-tag v-if="port.vulns && port.vulns.length > 0" size="small" type="danger">
                      {{ port.vulns.length }} vuln(s)
                    </el-tag>
                  </div>
                  <div v-if="host.ports.length > 5" class="more-hint">
                    ... and {{ host.ports.length - 5 }} more ports
                  </div>
                </div>
              </div>
            </div>
            <div v-else class="empty-hint">No hosts discovered yet</div>
          </el-collapse-item>

          <el-collapse-item title="Vulnerabilities" name="vulns">
            <div v-if="allVulns.length > 0" class="vuln-list">
              <div v-for="vuln in allVulns" :key="vuln.id" class="vuln-item">
                <div class="vuln-header">
                  <el-tag :type="vulnSeverityType(vuln.severity)" size="small">
                    {{ vuln.severity || 'unknown' }}
                  </el-tag>
                  <span class="vuln-name">{{ vuln.name }}</span>
                </div>
                <div class="vuln-target">{{ vuln.target }}</div>
                <div v-if="vuln.exp_available" class="vuln-exploit">
                  <el-tag type="danger" size="small">Exploit Available</el-tag>
                </div>
              </div>
            </div>
            <div v-else class="empty-hint">No vulnerabilities found</div>
          </el-collapse-item>
        </el-collapse>
      </el-card>

      <div class="graph-container">
        <div ref="cyContainer" class="cy-container"></div>

      <!-- Node detail panel -->
      <transition name="slide">
        <div v-if="selectedNode" class="node-detail">
          <div class="detail-header">
            <h3>{{ selectedNode.label }}</h3>
            <el-button @click="selectedNode = null" size="small" text>Close</el-button>
          </div>
          <el-descriptions :column="1" size="small" border>
            <el-descriptions-item label="Type">
              <el-tag size="small">{{ selectedNode.type }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item v-for="(val, key) in selectedNode" :key="key" :label="String(key)">
              {{ val }}
            </el-descriptions-item>
          </el-descriptions>
          <div class="detail-actions" v-if="selectedNode.type !== 'target'">
            <el-button
              v-if="selectedNode.type === 'host'"
              size="small" type="primary"
              @click="actionScan('scan_ports', selectedNode.ip)"
            >Scan Ports</el-button>
            <el-button
              v-if="selectedNode.type === 'port'"
              size="small" type="primary"
              @click="actionScan('scan_paths', selectedNode.id)"
            >Scan Paths</el-button>
            <el-button
              v-if="selectedNode.type === 'port' || selectedNode.type === 'path'"
              size="small" type="warning"
              @click="actionScan('scan_vulns', selectedNode.id)"
            >Vuln Scan</el-button>
            <el-button
              v-if="selectedNode.type === 'vuln' && selectedNode.exp_available"
              size="small" type="danger"
              @click="actionExploit(selectedNode)"
            >Exploit</el-button>
          </div>
        </div>
      </transition>
      </div>
    </div>

    <!-- Add Target Dialog -->
    <el-dialog v-model="showAddTargetDialog" title="Add Target" width="500px">
      <el-form :model="targetForm" label-width="80px">
        <el-form-item label="Name">
          <el-input v-model="targetForm.name" placeholder="e.g. Main Site" />
        </el-form-item>
        <el-form-item label="URL">
          <el-input v-model="targetForm.url" placeholder="https://target.example.com" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showAddTargetDialog = false">Cancel</el-button>
        <el-button type="primary" :loading="addingTarget" @click="addTarget">Add</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { useAssetStore } from '@/stores/asset'
import { listProjects, type Project, type Target } from '@/api/project'
import { getProjectTargets, addTarget as addTargetAPI } from '@/api/target'
import { ElMessage } from 'element-plus'
import cytoscape from 'cytoscape'

const assetStore = useAssetStore()
const cyContainer = ref<HTMLElement | null>(null)
const selectedNode = ref<any>(null)
let cy: any = null

const projects = ref<Project[]>([])
const selectedProjectId = ref<string>('')
const targets = ref<Target[]>([])
const selectedTargetId = ref<string>('')
const loadingTargets = ref(false)
const showAddTargetDialog = ref(false)
const addingTarget = ref(false)
const targetForm = ref({ name: '', url: '' })
const showDetailPanel = ref(true)
const activeCollapse = ref(['hosts', 'vulns'])

const allVulns = computed(() => {
  const vulns: any[] = []
  assetStore.hosts.forEach((host: any) => {
    host.ports?.forEach((port: any) => {
      port.vulns?.forEach((vuln: any) => {
        vulns.push(vuln)
      })
      port.paths?.forEach((path: any) => {
        path.vulns?.forEach((vuln: any) => {
          vulns.push(vuln)
        })
      })
    })
  })
  return vulns
})

function countPaths(host: any): number {
  let count = 0
  host.ports?.forEach((port: any) => {
    count += port.paths?.length || 0
  })
  return count
}

function countVulns(host: any): number {
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

async function loadProjects() {
  try {
    const resp = await listProjects()
    projects.value = resp.projects
    if (projects.value.length > 0 && !selectedProjectId.value) {
      selectedProjectId.value = projects.value[0].id
      await loadTargets()
    }
  } catch (e: any) {
    ElMessage.error(`Failed to load projects: ${e.message}`)
  }
}

async function loadTargets() {
  if (!selectedProjectId.value) return
  loadingTargets.value = true
  try {
    targets.value = await getProjectTargets(selectedProjectId.value)
    if (targets.value.length > 0 && !selectedTargetId.value) {
      selectedTargetId.value = targets.value[0].id
    }
    await loadSelectedTargetAssets()
  } catch (e: any) {
    ElMessage.error(`Failed to load targets: ${e.message}`)
  } finally {
    loadingTargets.value = false
  }
}

async function onProjectChange() {
  selectedTargetId.value = ''
  targets.value = []
  assetStore.clear()
  updateGraph()
  await loadTargets()
}

async function selectTarget(targetId: string) {
  selectedTargetId.value = targetId
  await loadSelectedTargetAssets()
}

async function loadSelectedTargetAssets() {
  const target = targets.value.find(t => t.id === selectedTargetId.value)
  if (!target) {
    assetStore.clear()
    updateGraph()
    return
  }
  await assetStore.loadByTarget(target.url, target.id)
  updateGraph()
}

async function addTarget() {
  const name = targetForm.value.name.trim()
  const url = targetForm.value.url.trim()
  if (!name || !url) {
    ElMessage.warning('Name and URL are required')
    return
  }

  addingTarget.value = true
  try {
    const created = await addTargetAPI(selectedProjectId.value, name, url)
    targets.value = [...targets.value, created]
    targetForm.value = { name: '', url: '' }
    showAddTargetDialog.value = false
    ElMessage.success('Target added')
    if (!selectedTargetId.value) {
      selectedTargetId.value = created.id
    }
  } catch (e: any) {
    ElMessage.error(e.message)
  } finally {
    addingTarget.value = false
  }
}

function initCytoscape() {
  if (!cyContainer.value) return
  cy = cytoscape({
    container: cyContainer.value,
    elements: [],
    style: [
      {
        selector: 'node',
        style: {
          label: 'data(label)',
          'text-valign': 'bottom',
          'text-halign': 'center',
          'font-size': '10px',
          color: '#6b5f56',
          'text-margin-y': 4,
          'background-color': '#9a8e84',
          width: 30,
          height: 30,
        },
      },
      {
        selector: 'node.target',
        style: { 'background-color': '#6b8cbe', width: 50, height: 50, 'font-size': '12px' },
      },
      {
        selector: 'node.host',
        style: { 'background-color': '#6a9e7a', width: 40, height: 40 },
      },
      {
        selector: 'node.port',
        style: { 'background-color': '#c49a4a', width: 30, height: 30 },
      },
      {
        selector: 'node.path',
        style: { 'background-color': '#9a8e84', width: 24, height: 24 },
      },
      {
        selector: 'node.vuln',
        style: { 'background-color': '#c06060', width: 28, height: 28, shape: 'diamond' },
      },
      {
        selector: 'node.vuln.vuln-critical',
        style: { 'background-color': '#b04040', width: 34, height: 34 },
      },
      {
        selector: 'node.exploitable',
        style: { 'border-width': 3, 'border-color': '#c06060' },
      },
      {
        selector: 'edge',
        style: {
          width: 1.5,
          'line-color': '#d4ccc2',
          'target-arrow-color': '#d4ccc2',
          'target-arrow-shape': 'triangle',
          'curve-style': 'bezier',
        },
      },
      {
        selector: ':selected',
        style: { 'border-width': 3, 'border-color': '#6b8cbe' },
      },
    ],
    layout: { name: 'cose', animate: false, nodeDimensionsIncludeLabels: true },
    minZoom: 0.2,
    maxZoom: 5,
  })

  cy.on('tap', 'node', (event: any) => {
    selectedNode.value = event.target.data()
  })

  cy.on('tap', (event: any) => {
    if (event.target === cy) {
      selectedNode.value = null
    }
  })
}

function updateGraph() {
  if (!cy) return
  const elements = assetStore.elements

  cy.elements().remove()
  if (elements.length === 0) {
    selectedNode.value = null
    return
  }
  cy.add(elements)
  cy.layout({ name: 'cose', animate: false, nodeDimensionsIncludeLabels: true }).run()
}

function refreshGraph() {
  if (selectedTargetId.value) {
    void loadSelectedTargetAssets()
    return
  }
  assetStore.load().then(() => updateGraph())
}

function fitView() {
  if (cy) cy.fit()
}

function actionScan(action: string, target: string) {
  // Placeholder - will wire up to scan API.
  console.log('Action:', action, 'Target:', target)
}

function actionExploit(node: any) {
  console.log('Exploit:', node)
}

watch(() => assetStore.elements, updateGraph)

onMounted(async () => {
  await loadProjects()
  await nextTick()
  initCytoscape()
  await loadSelectedTargetAssets()
})

onUnmounted(() => {
  if (cy) {
    cy.destroy()
    cy = null
  }
})
</script>

<style scoped>
.asset-graph {
  height: calc(100vh - 40px);
  display: flex;
  flex-direction: row;
}

.sidebar {
  width: 280px;
  background: var(--pf-bg-surface);
  border: 1px solid var(--pf-border);
  padding: 16px;
  overflow-y: auto;
}

.sidebar-section {
  margin-bottom: 24px;
}

.sidebar-section h3 {
  color: var(--pf-text-primary);
  font-size: 14px;
  margin-bottom: 12px;
}

.targets-list {
  max-height: 400px;
  overflow-y: auto;
  margin-bottom: 8px;
}

.target-item {
  padding: 10px;
  margin-bottom: 8px;
  background: var(--pf-bg-base);
  border: 1px solid var(--pf-border);
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.2s;
}

.target-item:hover {
  border-color: var(--pf-accent-cyan);
  background: var(--pf-bg-elevated);
}

.target-item.active {
  border-color: var(--pf-accent-cyan);
  background: var(--pf-bg-active);
}

.target-name {
  color: var(--pf-text-primary);
  font-size: 13px;
  font-weight: 500;
  margin-bottom: 4px;
}

.target-url {
  color: var(--pf-text-muted);
  font-size: 11px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.empty-hint {
  text-align: center;
  padding: 20px;
  color: var(--pf-text-dim);
  font-size: 12px;
}

.main-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 20px;
  overflow: hidden;
}

.details-panel {
  margin-bottom: 16px;
  max-height: 300px;
  overflow-y: auto;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.host-list, .vuln-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.host-item {
  padding: 12px;
  background: var(--pf-bg-base);
  border: 1px solid var(--pf-border);
  border-radius: 4px;
}

.host-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.host-domain {
  color: var(--pf-text-muted);
  font-size: 12px;
}

.host-stats {
  display: flex;
  gap: 6px;
  margin-bottom: 8px;
}

.port-list {
  margin-top: 8px;
  padding-left: 12px;
  border-left: 2px solid var(--pf-border);
}

.port-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 0;
  font-size: 12px;
}

.port-number {
  color: #e6a23c;
  font-weight: 500;
  min-width: 50px;
}

.port-service {
  color: var(--pf-text-muted);
  flex: 1;
}

.more-hint {
  color: var(--pf-text-dim);
  font-size: 11px;
  padding: 4px 0;
  font-style: italic;
}

.vuln-item {
  padding: 10px;
  background: var(--pf-bg-base);
  border: 1px solid var(--pf-border);
  border-radius: 4px;
}

.vuln-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.vuln-name {
  color: var(--pf-text-primary);
  font-size: 13px;
  font-weight: 500;
}

.vuln-target {
  color: var(--pf-text-muted);
  font-size: 11px;
  margin-bottom: 4px;
}

.vuln-exploit {
  margin-top: 6px;
}

.graph-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.graph-header h1 {
  color: var(--pf-text-primary);
}

.graph-controls {
  display: flex;
  gap: 8px;
  align-items: center;
}

.graph-container {
  flex: 1;
  position: relative;
  border: 1px solid var(--pf-border);
  border-radius: 4px;
  overflow: hidden;
}

.cy-container {
  width: 100%;
  height: 100%;
  background: var(--pf-bg-terminal);
}

.node-detail {
  position: absolute;
  top: 0;
  right: 0;
  width: 320px;
  height: 100%;
  background: var(--pf-bg-surface);
  border: 1px solid var(--pf-border);
  padding: 16px;
  overflow-y: auto;
}

.detail-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.detail-header h3 {
  color: var(--pf-text-primary);
  font-size: 14px;
}

.detail-actions {
  margin-top: 16px;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.slide-enter-active, .slide-leave-active {
  transition: transform 0.2s;
}
.slide-enter-from, .slide-leave-to {
  transform: translateX(100%);
}
</style>
