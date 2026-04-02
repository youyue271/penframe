<template>
  <div class="asset-graph">
    <div class="graph-header">
      <h1>Asset Graph</h1>
      <div class="graph-controls">
        <el-button @click="refreshGraph" :loading="assetStore.loading" type="primary" size="small">
          Refresh
        </el-button>
        <el-button @click="fitView" size="small">Fit View</el-button>
        <el-tag type="info" size="small">
          {{ assetStore.summary.hosts }}H / {{ assetStore.summary.ports }}P / {{ assetStore.summary.paths }}Pa / {{ assetStore.summary.vulns }}V
        </el-tag>
      </div>
    </div>

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
</template>

<script setup lang="ts">
import { ref, onMounted, watch, nextTick } from 'vue'
import { useAssetStore } from '@/stores/asset'
import cytoscape from 'cytoscape'

const assetStore = useAssetStore()
const cyContainer = ref<HTMLElement | null>(null)
const selectedNode = ref<any>(null)
let cy: any = null

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
          color: '#c0c4cc',
          'text-margin-y': 4,
          'background-color': '#606266',
          width: 30,
          height: 30,
        },
      },
      {
        selector: 'node.target',
        style: { 'background-color': '#409eff', width: 50, height: 50, 'font-size': '12px' },
      },
      {
        selector: 'node.host',
        style: { 'background-color': '#67c23a', width: 40, height: 40 },
      },
      {
        selector: 'node.port',
        style: { 'background-color': '#e6a23c', width: 30, height: 30 },
      },
      {
        selector: 'node.path',
        style: { 'background-color': '#909399', width: 24, height: 24 },
      },
      {
        selector: 'node.vuln',
        style: { 'background-color': '#f56c6c', width: 28, height: 28, shape: 'diamond' },
      },
      {
        selector: 'node.vuln.vuln-critical',
        style: { 'background-color': '#ff0000', width: 34, height: 34 },
      },
      {
        selector: 'node.exploitable',
        style: { 'border-width': 3, 'border-color': '#ff4500' },
      },
      {
        selector: 'edge',
        style: {
          width: 1.5,
          'line-color': '#404040',
          'target-arrow-color': '#404040',
          'target-arrow-shape': 'triangle',
          'curve-style': 'bezier',
        },
      },
      {
        selector: ':selected',
        style: { 'border-width': 3, 'border-color': '#409eff' },
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
  if (elements.length === 0) return

  cy.elements().remove()
  cy.add(elements)
  cy.layout({ name: 'cose', animate: false, nodeDimensionsIncludeLabels: true }).run()
}

function refreshGraph() {
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
  await nextTick()
  initCytoscape()
  await assetStore.load()
  updateGraph()
})
</script>

<style scoped>
.asset-graph {
  height: calc(100vh - 40px);
  display: flex;
  flex-direction: column;
}

.graph-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.graph-header h1 {
  color: #e5e5e5;
}

.graph-controls {
  display: flex;
  gap: 8px;
  align-items: center;
}

.graph-container {
  flex: 1;
  position: relative;
  border: 1px solid #303030;
  border-radius: 4px;
  overflow: hidden;
}

.cy-container {
  width: 100%;
  height: 100%;
  background: #0d0d0d;
}

.node-detail {
  position: absolute;
  top: 0;
  right: 0;
  width: 320px;
  height: 100%;
  background: #1d1e1f;
  border-left: 1px solid #303030;
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
  color: #e5e5e5;
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
