<template>
  <el-container class="app-layout">
    <el-aside width="200px" class="app-sidebar">
      <div class="logo">
        <h2>Penframe</h2>
      </div>
      <el-menu
        :default-active="route.path"
        router
        background-color="#1d1e1f"
        text-color="#c0c4cc"
        active-text-color="#409eff"
      >
        <el-menu-item index="/projects">
          <el-icon><Folder /></el-icon>
          <span>Projects</span>
        </el-menu-item>
        <el-menu-item index="/assets">
          <el-icon><Share /></el-icon>
          <span>Asset Graph</span>
        </el-menu-item>
        <el-menu-item index="/exploit">
          <el-icon><Lightning /></el-icon>
          <span>Exploit</span>
        </el-menu-item>
      </el-menu>
      <div class="sse-status">
        <el-tag :type="sse.connected ? 'success' : 'danger'" size="small">
          {{ sse.connected ? 'Connected' : 'Disconnected' }}
        </el-tag>
      </div>
    </el-aside>
    <el-main class="app-main">
      <router-view />
    </el-main>
  </el-container>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { Monitor, Share, Folder } from '@element-plus/icons-vue'
import { useSSEStore } from '@/stores/sse'

// Lightning icon placeholder since it may not exist in the icon pack.
const Lightning = Monitor

const route = useRoute()
const sse = useSSEStore()

onMounted(() => {
  console.log('[App] Component mounted, route:', route.path)
  try {
    sse.connect()
    console.log('[App] SSE connection initiated')
  } catch (err) {
    console.error('[App] SSE connection failed:', err)
  }
})
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html, body, #app {
  height: 100%;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  background: #141414;
  color: #e5e5e5;
}

.app-layout {
  height: 100vh;
}

.app-sidebar {
  background: #1d1e1f;
  border-right: 1px solid #303030;
  overflow-y: auto;
}

.logo {
  padding: 20px;
  text-align: center;
  border-bottom: 1px solid #303030;
}

.logo h2 {
  color: #409eff;
  font-size: 20px;
  letter-spacing: 2px;
}

.sse-status {
  position: absolute;
  bottom: 16px;
  left: 16px;
}

.app-main {
  background: #141414;
  padding: 20px;
  overflow-y: auto;
}

/* Dark theme overrides for Element Plus */
.el-menu {
  border-right: none !important;
}
</style>
