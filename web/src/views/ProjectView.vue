<template>
  <div class="project-view">
    <h1>Projects</h1>

    <el-card shadow="hover" class="project-card">
      <template #header>
        <span>Add Project</span>
      </template>
      <el-form :inline="true" @submit.prevent="addProject">
        <el-form-item label="Name">
          <el-input v-model="form.name" placeholder="e.g. Dify staging" clearable style="width: 320px" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" native-type="submit" :loading="adding">Add</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card shadow="hover" class="project-card">
      <template #header>
        <div class="project-header">
          <span>Project List</span>
          <el-button size="small" @click="loadProjects" :loading="loading">Refresh</el-button>
        </div>
      </template>

      <el-table v-loading="loading" :data="projects" size="small" style="width: 100%" @row-click="openProject">
        <el-table-column prop="name" label="Name" width="300" />
        <el-table-column label="Targets" width="120">
          <template #default="{ row }">
            {{ row.targets?.length || 0 }}
          </template>
        </el-table-column>
        <el-table-column label="Created" width="190">
          <template #default="{ row }">
            {{ formatDate(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="Actions" width="120" fixed="right">
          <template #default="{ row }">
            <el-popconfirm title="Delete this project?" @confirm.stop="removeProject(row.id)">
              <template #reference>
                <el-button size="small" type="danger" :loading="deletingId === row.id" @click.stop>Delete</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>

      <div v-if="!loading && !projects.length" class="empty-hint">
        No projects yet.
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { createProject, deleteProject, listProjects, type Project } from '@/api/project'

const router = useRouter()
const projects = ref<Project[]>([])
const loading = ref(false)
const adding = ref(false)
const deletingId = ref('')
const form = ref({ name: '' })

function formatDate(value: string) {
  return value ? new Date(value).toLocaleString() : '-'
}

async function loadProjects() {
  loading.value = true
  try {
    const resp = await listProjects()
    projects.value = resp.projects || []
  } catch (e: any) {
    ElMessage.warning(`Could not load projects: ${e.message}`)
  } finally {
    loading.value = false
  }
}

async function addProject() {
  const name = form.value.name.trim()
  if (!name) {
    ElMessage.warning('Name is required')
    return
  }

  adding.value = true
  try {
    const created = await createProject(name)
    projects.value = [...projects.value, created]
    form.value = { name: '' }
    ElMessage.success('Project added')
  } catch (e: any) {
    ElMessage.error(e.message)
  } finally {
    adding.value = false
  }
}

function openProject(row: Project) {
  router.push(`/projects/${row.id}`)
}

async function removeProject(id: string) {
  deletingId.value = id
  try {
    await deleteProject(id)
    projects.value = projects.value.filter((item) => item.id !== id)
    ElMessage.success('Project deleted')
  } catch (e: any) {
    ElMessage.error(e.message)
  } finally {
    deletingId.value = ''
  }
}

onMounted(() => {
  void loadProjects()
})
</script>

<style scoped>
.project-view {
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

.project-view h1 {
  font-family: 'Orbitron', 'JetBrains Mono', monospace;
  color: var(--pf-text-primary);
  margin-bottom: 24px;
  font-size: 28px;
  font-weight: 700;
  letter-spacing: 2px;
  text-transform: uppercase;
  background: linear-gradient(135deg, #88a8b8, var(--pf-text-primary));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  padding: 24px;
  border: 1px solid var(--pf-border);
  border-radius: var(--pf-radius-lg);
  background-color: var(--pf-bg-elevated);
  box-shadow: var(--pf-shadow-md);
}

.project-card {
  margin-bottom: 24px;
  background: var(--pf-bg-elevated);
  border: 1px solid var(--pf-border);
  border-radius: var(--pf-radius-lg);
  box-shadow: var(--pf-shadow-md);
  transition: all 0.3s ease;
}

.project-card:hover {
  border-color: rgba(0, 217, 255, 0.3);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.6), 0 0 20px rgba(0, 217, 255, 0.1);
}

.project-card :deep(.el-card__header) {
  background: linear-gradient(135deg, rgba(0, 217, 255, 0.08), rgba(168, 85, 247, 0.08));
  border-bottom: 1px solid var(--pf-border);
  padding: 16px 20px;
  font-family: 'Orbitron', 'JetBrains Mono', monospace;
  font-weight: 700;
  font-size: 14px;
  letter-spacing: 1px;
  text-transform: uppercase;
  color: var(--pf-accent-cyan);
}

.project-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.project-card :deep(.el-table) {
  background: transparent;
  color: var(--pf-text-primary);
}

.project-card :deep(.el-table__header-wrapper) {
  background: var(--pf-bg-surface);
}

.project-card :deep(.el-table th) {
  background: var(--pf-bg-surface);
  color: #88a8b8;
  font-family: 'JetBrains Mono', monospace;
  font-weight: 600;
  font-size: 12px;
  letter-spacing: 1px;
  text-transform: uppercase;
  border-bottom: 2px solid var(--pf-border);
}

.project-card :deep(.el-table td) {
  border-bottom: 1px solid var(--pf-border);
  font-family: 'JetBrains Mono', monospace;
  font-size: 13px;
}

.project-card :deep(.el-table__row) {
  background: var(--pf-bg-elevated);
  transition: all 0.2s ease;
  cursor: pointer;
}

.project-card :deep(.el-table__row:hover) {
  background: var(--pf-bg-hover) !important;
}

.project-card :deep(.el-table__body tr) {
  background: var(--pf-bg-elevated) !important;
}

.project-card :deep(.el-table__body tr:hover) {
  background: var(--pf-bg-hover) !important;
}

.empty-hint {
  text-align: center;
  color: var(--pf-text-muted);
  padding: 40px 0 20px;
  font-family: 'JetBrains Mono', monospace;
  font-size: 14px;
  letter-spacing: 0.5px;
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
