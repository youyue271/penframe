<template>
  <el-drawer
    v-model="vshell.visible"
    title="VShell Terminal"
    direction="rtl"
    size="100%"
    :close-on-press-escape="true"
    :close-on-click-modal="false"
  >
    <div v-if="vshell.loading" class="loading-container">
      <el-icon class="is-loading"><Loading /></el-icon>
      <span>Loading VShell...</span>
    </div>
    <div v-else-if="!vshell.config?.enabled" class="disabled-container">
      <el-empty description="VShell is not enabled">
        <el-text type="info">
          Enable VShell in the Config page to use the terminal.
        </el-text>
      </el-empty>
    </div>
    <iframe
      v-else-if="vshell.config?.web_url"
      :src="vshell.config.web_url"
      class="vshell-iframe"
      frameborder="0"
    />
    <div v-else class="error-container">
      <el-empty description="VShell URL not configured" />
    </div>
  </el-drawer>
</template>

<script setup lang="ts">
import { Loading } from '@element-plus/icons-vue'
import { useVShellStore } from '@/stores/vshell'

const vshell = useVShellStore()
</script>

<style scoped>
.loading-container,
.disabled-container,
.error-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  gap: 16px;
  background: var(--pf-bg-base);
  font-family: 'JetBrains Mono', monospace;
}

.vshell-iframe {
  width: 100%;
  height: 100%;
  border: none;
  background: var(--pf-bg-terminal);
}

.el-icon.is-loading {
  font-size: 32px;
  color: var(--pf-accent-cyan);
  filter: drop-shadow(0 0 8px var(--pf-accent-cyan));
  animation: pulse 2s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
    transform: scale(1);
  }
  50% {
    opacity: 0.6;
    transform: scale(1.1);
  }
}

:deep(.el-drawer) {
  background: var(--pf-bg-elevated);
  border-left: 1px solid var(--pf-border);
  box-shadow: -8px 0 32px rgba(0, 0, 0, 0.6);
}

:deep(.el-drawer__header) {
  background: linear-gradient(135deg, rgba(0, 217, 255, 0.1), rgba(168, 85, 247, 0.1));
  border-bottom: 1px solid var(--pf-border);
  padding: 20px 24px;
  margin-bottom: 0;
}

:deep(.el-drawer__title) {
  font-family: 'Orbitron', 'JetBrains Mono', monospace;
  color: var(--pf-accent-cyan);
  font-weight: 700;
  font-size: 18px;
  letter-spacing: 2px;
  text-transform: uppercase;
  text-shadow: 0 0 12px rgba(0, 217, 255, 0.5);
}

:deep(.el-drawer__body) {
  padding: 0;
  background: var(--pf-bg-terminal);
}

:deep(.el-empty__description) {
  font-family: 'JetBrains Mono', monospace;
  color: var(--pf-text-secondary);
  font-size: 14px;
  letter-spacing: 0.5px;
}

:deep(.el-text) {
  font-family: 'JetBrains Mono', monospace;
  font-size: 13px;
}
</style>
