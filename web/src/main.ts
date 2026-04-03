import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { createRouter, createWebHistory } from 'vue-router'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'

import App from './App.vue'

import Dashboard from './views/Dashboard.vue'
import TargetWorkspace from './views/TargetWorkspace.vue'
import AssetGraph from './views/AssetGraph.vue'
import ScanControl from './views/ScanControl.vue'
import ExploitPanel from './views/ExploitPanel.vue'
import RunHistory from './views/RunHistory.vue'
import LogViewer from './views/LogViewer.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'target', component: TargetWorkspace },
    { path: '/dashboard', name: 'dashboard', component: Dashboard },
    { path: '/assets', name: 'assets', component: AssetGraph },
    { path: '/scan', name: 'scan', component: ScanControl },
    { path: '/exploit', name: 'exploit', component: ExploitPanel },
    { path: '/history', name: 'history', component: RunHistory },
    { path: '/logs', name: 'logs', component: LogViewer },
  ],
})


const app = createApp(App)
app.use(createPinia())
app.use(router)
app.use(ElementPlus)
app.mount('#app')
