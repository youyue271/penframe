<template>
  <el-container class="app-layout">
    <el-aside width="200px" class="app-sidebar">
      <div class="sidebar-inner">
        <div class="logo">
          <h2>Penframe</h2>
        </div>
        <el-menu
          :default-active="route.path"
          router
          background-color="transparent"
          :text-color="themeStore.isDark ? '#b8bcc4' : '#1a1d18'"
          :active-text-color="themeStore.isDark ? '#7b9ea8' : '#4a6d78'"
          class="sidebar-menu"
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
          <el-menu-item index="/config">
            <el-icon><Setting /></el-icon>
            <span>Config</span>
          </el-menu-item>
        </el-menu>
        <div class="sidebar-bottom">
          <el-button
            type="primary"
            :icon="Monitor"
            @click="vshellStore.toggle()"
            class="vshell-toggle"
            size="small"
          >
            VShell
          </el-button>
          <el-button
            @click="themeStore.toggleTheme()"
            class="theme-toggle"
            size="small"
          >
            <span class="theme-icon">{{ themeStore.isDark ? '☀️' : '🌙' }}</span>
            <span>{{ themeStore.isDark ? 'Light' : 'Dark' }}</span>
          </el-button>
          <div class="sse-status">
            <el-tag :type="sse.connected ? 'success' : 'danger'" size="small">
              {{ sse.connected ? 'Connected' : 'Disconnected' }}
            </el-tag>
          </div>
        </div>
      </div>
    </el-aside>
    <el-main class="app-main">
      <router-view />
    </el-main>
    <VShellPanel />
  </el-container>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { Monitor, Share, Folder, Setting } from '@element-plus/icons-vue'
import { useSSEStore } from '@/stores/sse'
import { useVShellStore } from '@/stores/vshell'
import { useThemeStore } from '@/stores/theme'
import VShellPanel from '@/components/VShellPanel.vue'

// Lightning icon placeholder since it may not exist in the icon pack.
const Lightning = Monitor

const route = useRoute()
const sse = useSSEStore()
const vshellStore = useVShellStore()
const themeStore = useThemeStore()

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
@import url('https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600;700&family=Orbitron:wght@700;900&display=swap');

:root {
  /* Core Background Colors - Morandi Style */
  --pf-bg-base: #1a1d28;
  --pf-bg-elevated: #22252f;
  --pf-bg-surface: #2a2d37;
  --pf-bg-hover: #32353f;
  --pf-bg-active: #3a3d47;
  --pf-bg-terminal: #14161d;

  /* Border Colors */
  --pf-border: #3a3d47;
  --pf-border-light: #4a4d57;
  --pf-border-glow: rgba(139, 146, 156, 0.3);

  /* Text Colors */
  --pf-text-primary: #e8eaed;
  --pf-text-secondary: #b8bcc4;
  --pf-text-muted: #8b929c;
  --pf-text-dim: #5a5f6a;

  /* Accent Colors - Morandi Low Saturation with Better Contrast */
  --pf-accent-cyan: #6b8a94;
  --pf-accent-magenta: #9a7d8e;
  --pf-accent-green: #7d9a7d;
  --pf-accent-purple: #8b7d9a;
  --pf-accent-yellow: #9a916b;

  /* Semantic Colors */
  --pf-success: #7d9a7d;
  --pf-warning: #9a916b;
  --pf-danger: #9a7d7d;
  --pf-info: #6b8a94;

  /* Radius */
  --pf-radius: 8px;
  --pf-radius-lg: 12px;

  /* Shadows & Glows */
  --pf-shadow-sm: 0 2px 8px rgba(0, 0, 0, 0.3);
  --pf-shadow-md: 0 4px 16px rgba(0, 0, 0, 0.4);
  --pf-shadow-lg: 0 8px 32px rgba(0, 0, 0, 0.5);
  --pf-glow-cyan: 0 0 20px rgba(141, 180, 190, 0.3);
  --pf-glow-magenta: 0 0 20px rgba(184, 157, 174, 0.3);
  --pf-glow-green: 0 0 20px rgba(157, 184, 157, 0.3);
}

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html, body, #app {
  height: 100%;
  font-family: 'JetBrains Mono', 'Consolas', 'Monaco', monospace;
  background: var(--pf-bg-base);
  color: var(--pf-text-primary);
  font-size: 14px;
  line-height: 1.6;
}

/* Animated grid background */
body::before {
  content: '';
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-image:
    linear-gradient(rgba(139, 146, 156, 0.02) 1px, transparent 1px),
    linear-gradient(90deg, rgba(139, 146, 156, 0.02) 1px, transparent 1px);
  background-size: 50px 50px;
  pointer-events: none;
  z-index: 0;
}

/* Scanline effect - removed for cleaner look */
body::after {
  content: '';
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
  z-index: 1;
}

.app-layout {
  height: 100vh;
  position: relative;
  z-index: 2;
}

.app-sidebar {
  background: var(--pf-sidebar-bg, #1f2229);
  border-right: 1px solid var(--pf-border);
  box-shadow: 4px 0 24px rgba(0, 0, 0, 0.5);
  position: relative;
}

.app-sidebar::before {
  content: '';
  position: absolute;
  top: 0;
  right: 0;
  width: 1px;
  height: 100%;
  background: linear-gradient(
    to bottom,
    transparent,
    var(--pf-accent-cyan) 50%,
    transparent
  );
  opacity: 0.3;
}

.sidebar-inner {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  position: relative;
  z-index: 3;
}

.sidebar-menu {
  border-right: none !important;
  background: transparent !important;
}

.sidebar-menu .el-menu-item {
  font-family: 'JetBrains Mono', monospace;
  font-weight: 500;
  font-size: 13px;
  letter-spacing: 0.5px;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  border-left: 3px solid transparent;
  margin: 4px 0;
}

.sidebar-menu .el-menu-item:hover {
  background: var(--pf-bg-hover) !important;
  border-left-color: var(--pf-accent-cyan);
}

.sidebar-menu .el-menu-item.is-active {
  background: var(--pf-bg-active) !important;
  border-left-color: var(--pf-accent-cyan);
  color: var(--pf-accent-cyan) !important;
}

.sidebar-bottom {
  margin-top: auto;
  padding: 20px 16px;
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 12px;
  border-top: 1px solid var(--pf-border);
  background: transparent;
}

.vshell-toggle {
  width: 100%;
  font-family: 'JetBrains Mono', monospace;
  font-weight: 600;
  letter-spacing: 1px;
  background: linear-gradient(135deg, var(--pf-accent-cyan), var(--pf-accent-purple));
  border: none;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
  transition: all 0.3s ease;
}

.vshell-toggle:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
}

.vshell-toggle:active {
  transform: translateY(0);
}

.theme-toggle {
  width: 100%;
  font-family: 'JetBrains Mono', monospace;
  font-weight: 600;
  letter-spacing: 0.5px;
  background: var(--pf-bg-surface);
  border: 1px solid var(--pf-border);
  color: var(--pf-text-primary);
  transition: all 0.3s ease;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

.theme-toggle:hover {
  background: var(--pf-bg-hover);
  border-color: var(--pf-accent-cyan);
  transform: translateY(-1px);
}

.theme-toggle:active {
  transform: translateY(0);
}

.theme-icon {
  font-size: 14px;
  line-height: 1;
  flex-shrink: 0;
  display: inline-block;
  width: 14px;
  text-align: center;
}

.logo {
  padding: 28px 20px;
  text-align: center;
  border-bottom: 1px solid var(--pf-border);
  background: transparent;
  position: relative;
}

.logo::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 50%;
  transform: translateX(-50%);
  width: 60%;
  height: 1px;
  background: linear-gradient(
    90deg,
    transparent,
    var(--pf-accent-cyan),
    transparent
  );
  opacity: 0.5;
}

.logo h2 {
  font-family: 'Orbitron', 'JetBrains Mono', monospace;
  color: var(--pf-text-primary);
  font-size: 18px;
  font-weight: 900;
  letter-spacing: 3px;
  text-transform: uppercase;
  background: linear-gradient(135deg, var(--pf-accent-cyan), var(--pf-accent-purple));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  animation: logoGlow 2s ease-in-out infinite alternate;
}

@keyframes logoGlow {
  from {
    filter: drop-shadow(0 0 4px rgba(141, 180, 190, 0.3));
  }
  to {
    filter: drop-shadow(0 0 8px rgba(141, 180, 190, 0.5));
  }
}

.sse-status {
  display: flex;
  justify-content: center;
  align-items: center;
  width: 100%;
}

.sse-status .el-tag {
  font-family: 'JetBrains Mono', monospace;
  font-weight: 600;
  font-size: 11px;
  letter-spacing: 0.5px;
  border: 1px solid;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
}

.sse-status .el-tag.el-tag--success {
  background: rgba(136, 184, 136, 0.2);
  border-color: #88b888;
  color: #88b888;
  box-shadow: 0 0 8px rgba(136, 184, 136, 0.25);
  font-weight: 600;
}

.sse-status .el-tag.el-tag--danger {
  background: rgba(200, 136, 136, 0.2);
  border-color: #c88888;
  color: #c88888;
  box-shadow: 0 0 8px rgba(200, 136, 136, 0.25);
  font-weight: 600;
}

.app-main {
  background: var(--pf-bg-base);
  padding: 24px;
  overflow-y: auto;
  position: relative;
}

/* Custom scrollbar */
.app-main::-webkit-scrollbar {
  width: 8px;
}

.app-main::-webkit-scrollbar-track {
  background: var(--pf-bg-elevated);
}

.app-main::-webkit-scrollbar-thumb {
  background: var(--pf-border-light);
  border-radius: 4px;
}

.app-main::-webkit-scrollbar-thumb:hover {
  background: var(--pf-accent-cyan);
  box-shadow: 0 0 8px var(--pf-accent-cyan);
}

.el-menu {
  border-right: none !important;
}

/* Override Element Plus dark theme */
.el-button--primary {
  background: linear-gradient(135deg, var(--pf-accent-cyan), var(--pf-accent-purple));
  border: none;
  font-family: 'JetBrains Mono', monospace;
  font-weight: 600;
  letter-spacing: 0.5px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
  transition: all 0.3s ease;
}

.el-button--primary:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
}

.el-button {
  font-family: 'JetBrains Mono', monospace;
  font-weight: 500;
  border-radius: var(--pf-radius);
  transition: all 0.3s ease;
}

.el-card {
  background: var(--pf-bg-elevated);
  border: 1px solid var(--pf-border);
  border-radius: var(--pf-radius-lg);
  box-shadow: var(--pf-shadow-md);
  transition: all 0.3s ease;
}

.el-card:hover {
  border-color: rgba(141, 180, 190, 0.4);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.5), 0 0 20px rgba(141, 180, 190, 0.1);
}
</style>
