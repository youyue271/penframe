<template>
  <div class="dashboard">
    <div class="dashboard-header">
      <h1>{{ currentProject?.name || 'Project Targets' }}</h1>
      <div class="dashboard-actions">
        <el-button type="primary" @click="showAddDialog = true">Add Target</el-button>
        <el-button :loading="loading" @click="loadTargets">Refresh</el-button>
      </div>
    </div>

    <el-card shadow="hover" class="targets-card">
      <el-table
        v-loading="loading"
        :data="targets"
        style="width: 100%"
        size="default"
        @row-click="openTarget"
      >
        <el-table-column prop="name" label="Target Name" width="250" />
        <el-table-column prop="url" label="URL" show-overflow-tooltip />
        <el-table-column label="Last Scanned" width="190">
          <template #default="{ row }">
            {{ formatDate(row.last_scanned) }}
          </template>
        </el-table-column>
        <el-table-column label="Created" width="190">
          <template #default="{ row }">
            {{ formatDate(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="Actions" width="180" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click.stop="editTarget(row)">Edit</el-button>
            <el-popconfirm title="Delete this target?" @confirm.stop="removeTarget(row.id)">
              <template #reference>
                <el-button size="small" type="danger" :loading="deletingId === row.id" @click.stop>Delete</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>

      <div v-if="!loading && !targets.length" class="empty-hint">
        No targets yet. Click "Add Target" to get started.
      </div>
    </el-card>

    <el-dialog v-model="showAddDialog" title="Add Target" width="500px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="Name">
          <el-input v-model="form.name" placeholder="e.g. Main Site" />
        </el-form-item>
        <el-form-item label="URL">
          <el-input v-model="form.url" placeholder="https://target.example.com" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showAddDialog = false">Cancel</el-button>
        <el-button type="primary" :loading="adding" @click="addTarget">Add</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="showEditDialog" title="Edit Target" width="500px">
      <el-form :model="editForm" label-width="80px">
        <el-form-item label="Name">
          <el-input v-model="editForm.name" placeholder="e.g. Main Site" />
        </el-form-item>
        <el-form-item label="URL">
          <el-input v-model="editForm.url" placeholder="https://target.example.com" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showEditDialog = false">Cancel</el-button>
        <el-button type="primary" :loading="updating" @click="updateTargetSubmit">Update</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { addTarget as addTargetAPI, updateTarget as updateTargetAPI, deleteTarget, getProjectTargets } from '@/api/target'
import { listProjects, type Project, type Target } from '@/api/project'

const route = useRoute()
const router = useRouter()

const projectId = computed(() => route.params.id as string)
const currentProject = ref<Project | null>(null)
const targets = ref<Target[]>([])
const loading = ref(false)
const adding = ref(false)
const updating = ref(false)
const deletingId = ref('')
const showAddDialog = ref(false)
const showEditDialog = ref(false)
const form = ref({ name: '', url: '' })
const editForm = ref({ id: '', name: '', url: '' })

function formatDate(value?: string) {
  return value ? new Date(value).toLocaleString() : '-'
}

async function loadProject() {
  try {
    const resp = await listProjects()
    currentProject.value = resp.projects.find(p => p.id === projectId.value) || null
  } catch (e: any) {
    ElMessage.warning(`Could not load project: ${e.message}`)
  }
}

async function loadTargets() {
  loading.value = true
  try {
    targets.value = await getProjectTargets(projectId.value)
  } catch (e: any) {
    ElMessage.warning(`Could not load targets: ${e.message}`)
  } finally {
    loading.value = false
  }
}

async function addTarget() {
  const name = form.value.name.trim()
  const url = form.value.url.trim()
  if (!name || !url) {
    ElMessage.warning('Name and URL are required')
    return
  }

  adding.value = true
  try {
    const created = await addTargetAPI(projectId.value, name, url)
    targets.value = [...targets.value, created]
    form.value = { name: '', url: '' }
    showAddDialog.value = false
    ElMessage.success('Target added')
  } catch (e: any) {
    ElMessage.error(e.message)
  } finally {
    adding.value = false
  }
}

function editTarget(target: Target) {
  editForm.value = { id: target.id, name: target.name, url: target.url }
  showEditDialog.value = true
}

async function updateTargetSubmit() {
  const name = editForm.value.name.trim()
  const url = editForm.value.url.trim()
  if (!name || !url) {
    ElMessage.warning('Name and URL are required')
    return
  }

  updating.value = true
  try {
    const updated = await updateTargetAPI(projectId.value, editForm.value.id, name, url)
    const idx = targets.value.findIndex(t => t.id === updated.id)
    if (idx !== -1) {
      targets.value[idx] = updated
    }
    showEditDialog.value = false
    ElMessage.success('Target updated')
  } catch (e: any) {
    ElMessage.error(e.message)
  } finally {
    updating.value = false
  }
}

async function removeTarget(targetId: string) {
  deletingId.value = targetId
  try {
    await deleteTarget(projectId.value, targetId)
    targets.value = targets.value.filter(t => t.id !== targetId)
    ElMessage.success('Target deleted')
  } catch (e: any) {
    ElMessage.error(e.message)
  } finally {
    deletingId.value = ''
  }
}

function openTarget(row: Target) {
  router.push(`/projects/${projectId.value}/targets/${row.id}`)
}

onMounted(() => {
  loadProject()
  loadTargets()
})
</script>

<style scoped>
.dashboard {
  padding: 0;
  animation: fadeIn 0.5s ease-out;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.dashboard-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 32px;
  padding: 24px;
  background: linear-gradient(135deg, rgba(0, 217, 255, 0.05), rgba(168, 85, 247, 0.05));
  border: 1px solid var(--pf-border);
  border-radius: var(--pf-radius-lg);
  box-shadow: var(--pf-shadow-md);
  position: relative;
  overflow: hidden;
}

.dashboard-header::before {
  content: '';
  position: absolute;
  top: 0;
  left: -100%;
  width: 100%;
  height: 100%;
  background: linear-gradient(90deg, transparent, rgba(0, 217, 255, 0.1), transparent);
  animation: shimmer 3s infinite;
}

@keyframes shimmer {
  0% { left: -100%; }
  100% { left: 100%; }
}

.dashboard-header h1 {
  font-family: 'Orbitron', 'JetBrains Mono', monospace;
  color: var(--pf-text-primary);
  margin: 0;
  font-size: 28px;
  font-weight: 700;
  letter-spacing: 2px;
  text-transform: uppercase;
  background: linear-gradient(135deg, #88a8b8, var(--pf-text-primary));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  position: relative;
  z-index: 1;
}

.dashboard-actions {
  display: flex;
  gap: 12px;
  position: relative;
  z-index: 1;
}

.targets-card {
  margin-bottom: 20px;
  background: var(--pf-bg-elevated);
  border: 1px solid var(--pf-border);
  border-radius: var(--pf-radius-lg);
  overflow: hidden;
  box-shadow: var(--pf-shadow-md);
  transition: all 0.3s ease;
}

.targets-card:hover {
  border-color: rgba(0, 217, 255, 0.3);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.6), 0 0 20px rgba(0, 217, 255, 0.1);
}

.targets-card :deep(.el-card__body) {
  padding: 0;
}

.targets-card :deep(.el-table) {
  background: transparent;
  color: var(--pf-text-primary);
}

.targets-card :deep(.el-table__header-wrapper) {
  background: var(--pf-bg-surface);
}

.targets-card :deep(.el-table th) {
  background: var(--pf-bg-surface);
  color: #88a8b8;
  font-family: 'JetBrains Mono', monospace;
  font-weight: 600;
  font-size: 12px;
  letter-spacing: 1px;
  text-transform: uppercase;
  border-bottom: 2px solid var(--pf-border);
}

.targets-card :deep(.el-table td) {
  border-bottom: 1px solid var(--pf-border);
  font-family: 'JetBrains Mono', monospace;
  font-size: 13px;
}

.targets-card :deep(.el-table__row) {
  background: var(--pf-bg-elevated);
  transition: all 0.2s ease;
  cursor: pointer;
}

.targets-card :deep(.el-table__row:hover) {
  background: var(--pf-bg-hover) !important;
}

.targets-card :deep(.el-table__body tr) {
  background: var(--pf-bg-elevated) !important;
}

.targets-card :deep(.el-table__body tr:hover) {
  background: var(--pf-bg-hover) !important;
}

.empty-hint {
  text-align: center;
  padding: 60px 40px;
  color: var(--pf-text-muted);
  font-family: 'JetBrains Mono', monospace;
  font-size: 14px;
  letter-spacing: 0.5px;
  background: linear-gradient(135deg, rgba(0, 217, 255, 0.02), rgba(168, 85, 247, 0.02));
  border-radius: var(--pf-radius);
}

/* Dialog styling */
:deep(.el-dialog) {
  background: var(--pf-bg-elevated);
  border: 1px solid var(--pf-border);
  border-radius: var(--pf-radius-lg);
  box-shadow: 0 16px 48px rgba(0, 0, 0, 0.8), 0 0 40px rgba(0, 217, 255, 0.2);
}

:deep(.el-dialog__header) {
  background: linear-gradient(135deg, rgba(0, 217, 255, 0.1), rgba(168, 85, 247, 0.1));
  border-bottom: 1px solid var(--pf-border);
  padding: 20px 24px;
}

:deep(.el-dialog__title) {
  font-family: 'Orbitron', 'JetBrains Mono', monospace;
  color: var(--pf-accent-cyan);
  font-weight: 700;
  font-size: 18px;
  letter-spacing: 1px;
  text-transform: uppercase;
}

:deep(.el-dialog__body) {
  padding: 24px;
}

:deep(.el-form-item__label) {
  font-family: 'JetBrains Mono', monospace;
  color: var(--pf-text-secondary);
  font-weight: 600;
  font-size: 12px;
  letter-spacing: 0.5px;
}

:deep(.el-input__wrapper) {
  background: var(--pf-bg-surface);
  border: 1px solid var(--pf-border);
  border-radius: var(--pf-radius);
  box-shadow: inset 0 2px 4px rgba(0, 0, 0, 0.3);
  transition: all 0.3s ease;
}

:deep(.el-input__wrapper:hover) {
  border-color: var(--pf-border-light);
}

:deep(.el-input__wrapper.is-focus) {
  border-color: var(--pf-accent-cyan);
  box-shadow: 0 0 0 2px rgba(0, 217, 255, 0.2), inset 0 2px 4px rgba(0, 0, 0, 0.3);
}

:deep(.el-input__inner) {
  font-family: 'JetBrains Mono', monospace;
  color: var(--pf-text-primary);
  font-size: 13px;
}
</style>
