import { defineStore } from 'pinia'
import { ref } from 'vue'
import { listProjects, type Project } from '@/api/project'
import { getProjectTargets, type Target } from '@/api/target'

export const useProjectStore = defineStore('project', () => {
  const currentProjectId = ref<string | null>(null)
  const currentTargetId = ref<string | null>(null)

  const projects = ref<Project[]>([])
  const currentTargets = ref<Target[]>([])
  const loading = ref(false)

  async function loadProjects() {
    loading.value = true
    try {
      const res = await listProjects()
      projects.value = res.projects || []
    } finally {
      loading.value = false
    }
  }

  async function selectProject(id: string) {
    currentProjectId.value = id
    currentTargetId.value = null
    await loadTargets(id)
  }

  async function loadTargets(projectId: string) {
    loading.value = true
    try {
      const res = await getProjectTargets(projectId)
      currentTargets.value = res || []
    } finally {
      loading.value = false
    }
  }

  function selectTarget(id: string) {
    currentTargetId.value = id
  }

  return {
    currentProjectId,
    currentTargetId,
    projects,
    currentTargets,
    loading,
    loadProjects,
    selectProject,
    loadTargets,
    selectTarget
  }
})
