<template>
  <div class="log-viewer">
    <h1>Logs</h1>

    <el-card shadow="hover">
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center;">
          <span>Live Event Log</span>
          <div style="display: flex; gap: 8px; align-items: center;">
            <el-select v-model="filterType" placeholder="Filter by type" clearable size="small" style="width: 180px;">
              <el-option label="All" value="" />
              <el-option label="Run Started" value="run_started" />
              <el-option label="Node Started" value="node_started" />
              <el-option label="Node Finished" value="node_finished" />
              <el-option label="Run Finished" value="run_finished" />
            </el-select>
            <el-button size="small" @click="clearLogs">Clear</el-button>
          </div>
        </div>
      </template>

      <div class="log-output" ref="logContainer">
        <div
          v-for="(event, idx) in filteredEvents"
          :key="idx"
          class="log-line"
          :class="logLineClass(event.type)"
        >
          <span class="log-time">{{ formatTime(event.timestamp_unix_milli) }}</span>
          <span class="log-type">{{ event.type }}</span>
          <span class="log-run" v-if="event.run_id">{{ event.run_id }}</span>
          <span class="log-node" v-if="event.node">
            {{ event.node.node_id }}
            <template v-if="event.node.status"> [{{ event.node.status }}]</template>
            <template v-if="event.node.duration_millis"> {{ (event.node.duration_millis / 1000).toFixed(1) }}s</template>
          </span>
          <span class="log-error" v-if="event.node?.error">{{ event.node.error }}</span>
        </div>
        <div v-if="filteredEvents.length === 0" class="empty-hint">
          No events. Start a scan to see live logs.
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, watch } from 'vue'
import { useSSEStore } from '@/stores/sse'
import type { SSEvent } from '@/types'

const sseStore = useSSEStore()
const filterType = ref('')
const logContainer = ref<HTMLElement | null>(null)

const filteredEvents = computed(() => {
  let events = sseStore.events
  if (filterType.value) {
    events = events.filter(e => e.type === filterType.value)
  }
  return events
})

function formatTime(ms: number) {
  return new Date(ms).toLocaleTimeString()
}

function logLineClass(type: string) {
  if (type.includes('error') || type.includes('failed')) return 'log-error-line'
  if (type.includes('finished')) return 'log-success-line'
  if (type.includes('started')) return 'log-info-line'
  return ''
}

function clearLogs() {
  sseStore.events.length = 0
}

// Auto-scroll to bottom on new events.
watch(() => sseStore.events.length, async () => {
  await nextTick()
  if (logContainer.value) {
    logContainer.value.scrollTop = logContainer.value.scrollHeight
  }
})
</script>

<style scoped>
.log-viewer h1 {
  color: #e5e5e5;
  margin-bottom: 20px;
}

.log-output {
  max-height: 600px;
  overflow-y: auto;
  background: #0d0d0d;
  border-radius: 4px;
  padding: 8px;
  font-family: 'Cascadia Code', 'Fira Code', monospace;
  font-size: 13px;
}

.log-line {
  padding: 3px 8px;
  display: flex;
  gap: 8px;
  align-items: baseline;
  border-bottom: 1px solid #1a1a1a;
}

.log-time {
  color: #606266;
  min-width: 80px;
}

.log-type {
  color: #409eff;
  min-width: 120px;
}

.log-run {
  color: #909399;
}

.log-node {
  color: #67c23a;
}

.log-error {
  color: #f56c6c;
}

.log-error-line {
  background: rgba(245, 108, 108, 0.05);
}

.log-success-line {
  background: rgba(103, 194, 58, 0.05);
}

.log-info-line {
  background: rgba(64, 158, 255, 0.03);
}

.empty-hint {
  color: #606266;
  text-align: center;
  padding: 40px;
}
</style>
