<template>
  <div class="project-view">
    <h1>Projects</h1>

    <el-card shadow="hover" class="project-card">
      <template #header>
        <span>Add Project</span>
      </template>
      <el-form :inline="true" @submit.prevent="addProject">
        <el-form-item label="Name">
          <el-input v-model="form.name" placeholder="e.g. Dify staging" clearable />
        </el-form-item>
        <el-form-item label="URL">
          <el-input v-model="form.url" placeholder="https://target.example" clearable style="width: 320px" />
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

      <el-table v-loading="loading" :data="projects" size="small" stripe style="width: 100%">
        <el-table-column prop="name" label="Name" width="220" />
        <el-table-column prop="url" label="URL / Target" show-overflow-tooltip />
        <el-table-column label="Created" width="190">
          <template #default="{ row }">
            {{ formatDate(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="Actions" width="120" fixed="right">
          <template #default="{ row }">
            <el-popconfirm title="Delete this project?" @confirm="removeProject(row.id)">
              <template #reference>
                <el-button size="small" type="danger" :loading="deletingId === row.id">Delete</el-button>
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
import { ElMessage } from 'element-plus'
import { createProject, deleteProject, listProjects, type Project } from '@/api/project'

const projects = ref<Project[]>([])
const loading = ref(false)
const adding = ref(false)
const deletingId = ref('')
const form = ref({ name: '', url: '' })

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
  const url = form.value.url.trim()
  if (!name || !url) {
    ElMessage.warning('Name and URL are required')
    return
  }

  adding.value = true
  try {
    const created = await createProject(name, url)
    projects.value = [...projects.value, created]
    form.value = { name: '', url: '' }
    ElMessage.success('Project added')
  } catch (e: any) {
    ElMessage.error(e.message)
  } finally {
    adding.value = false
  }
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
.project-view h1 {
  color: #e5e5e5;
  margin-bottom: 20px;
}

.project-card {
  margin-bottom: 20px;
}

.project-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.empty-hint {
  text-align: center;
  color: #909399;
  padding: 24px 0 8px;
}
</style>
