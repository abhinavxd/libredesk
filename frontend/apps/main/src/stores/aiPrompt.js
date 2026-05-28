import { ref } from 'vue'
import { defineStore } from 'pinia'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useEmitter } from '../composables/useEmitter'
import { EMITTER_EVENTS } from '../constants/emitterEvents'
import api from '../api'

export const useAiPromptStore = defineStore('aiPrompt', () => {
    const prompts = ref([])
    const emitter = useEmitter()
    let inflight = null
    let hasFetched = false
    const fetchPrompts = () => {
        if (hasFetched) return Promise.resolve()
        if (inflight) return inflight
        inflight = api.getAiPrompts()
            .then(response => {
                prompts.value = response?.data?.data || []
                hasFetched = true
            })
            .catch(error => {
                emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
                    variant: 'destructive',
                    description: handleHTTPError(error).message
                })
            })
            .finally(() => { inflight = null })
        return inflight
    }
    return {
        prompts,
        fetchPrompts,
    }
})
