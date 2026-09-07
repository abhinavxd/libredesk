import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '@/api'

export const useAIAssistantStore = defineStore('aiAssistant', () => {
  const assistants = ref([])
  const loaded = ref(false)

  const loadAssistants = async (force = false) => {
    if (loaded.value && !force) return
    try {
      const resp = await api.getAIAssistantsCompact()
      assistants.value = resp.data.data || []
      loaded.value = true
    } catch {
      assistants.value = []
    }
  }

  const invalidate = () => {
    loaded.value = false
  }

  return {
    assistants,
    loadAssistants,
    invalidate
  }
})
