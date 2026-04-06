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
        stripe
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
  padding: 20px;
}

.dashboard-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.dashboard-header h1 {
  color: #e5e5e5;
  margin: 0;
}

.dashboard-actions {
  display: flex;
  gap: 10px;
}

.targets-card {
  margin-bottom: 20px;
}

.empty-hint {
  text-align: center;
  padding: 40px;
  color: #909399;
}
</style>
