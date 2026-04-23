import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { createRouter, createWebHistory } from 'vue-router'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'
import './styles/cyberpunk.css'

import App from './App.vue'

import Dashboard from './views/Dashboard.vue'
import TargetWorkspace from './views/TargetWorkspace.vue'
import AssetGraph from './views/AssetGraph.vue'
import ExploitPanel from './views/ExploitPanel.vue'
import ProjectView from './views/ProjectView.vue'
import ConfigView from './views/ConfigView.vue'

console.log('[Penframe] Starting application...')

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/projects' },
    { path: '/projects', name: 'projects', component: ProjectView },
    { path: '/projects/:id', name: 'project-detail', component: Dashboard },
    { path: '/projects/:id/targets/:tid', name: 'target-detail', component: TargetWorkspace },
    { path: '/projects/:id/targets/:tid/exploit', name: 'target-exploit', component: ExploitPanel },
    { path: '/assets', name: 'assets', component: AssetGraph },
    { path: '/exploit', name: 'exploit', component: ExploitPanel },
    { path: '/config', name: 'config', component: ConfigView },
  ],
})

console.log('[Penframe] Router created')

try {
  const app = createApp(App)
  console.log('[Penframe] App created')

  app.use(createPinia())
  console.log('[Penframe] Pinia installed')

  app.use(router)
  console.log('[Penframe] Router installed')

  app.use(ElementPlus)
  console.log('[Penframe] ElementPlus installed')

  app.mount('#app')
  console.log('[Penframe] App mounted successfully!')
} catch (err: any) {
  console.error('[Penframe] Fatal error:', err)
  document.body.innerHTML = '<div style="color: red; padding: 20px; font-family: monospace;"><h1>Error Loading Penframe</h1><pre>' + (err?.message || String(err)) + '\n\n' + (err?.stack || '') + '</pre></div>'
}
