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
}

.vshell-iframe {
  width: 100%;
  height: 100%;
  border: none;
  background: #000;
}

.el-icon.is-loading {
  font-size: 32px;
  color: #409eff;
}
</style>
