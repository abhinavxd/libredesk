import { ref } from 'vue'
import { defineStore } from 'pinia'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import api from '@main/api'

export const useAiPromptStore = defineStore('aiPrompt', () => {
  const prompts = ref([])
  const emitter = useEmitter()
  let inflight = null
  let hasFetched = false

  const showError = (error) => {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }

  const fetchPrompts = () => {
    if (hasFetched) return Promise.resolve()
    if (inflight) return inflight

    inflight = api
      .getAiPrompts()
      .then((response) => {
        prompts.value = response?.data?.data || []
        hasFetched = true
      })
      .catch(showError)
      .finally(() => {
        inflight = null
      })

    return inflight
  }

  const refreshPrompts = async () => {
    if (inflight) await inflight
    hasFetched = false
    return fetchPrompts()
  }

  const complete = async (promptKey, content) => {
    try {
      const response = await api.aiCompletion({ prompt_key: promptKey, content })
      return response?.data?.data ?? ''
    } catch (error) {
      showError(error)
      return null
    }
  }

  return {
    prompts,
    fetchPrompts,
    refreshPrompts,
    complete
  }
})
