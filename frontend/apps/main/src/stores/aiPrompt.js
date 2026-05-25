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
    const fetchPrompts = async () => {
        if (prompts.value.length) return
        if (inflight) return inflight
        inflight = (async () => {
            try {
                const response = await api.getAiPrompts()
                prompts.value = response?.data?.data || []
            } catch (error) {
                emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
                    variant: 'destructive',
                    description: handleHTTPError(error).message
                })
            } finally {
                inflight = null
            }
        })()
        return inflight
    }
    return {
        prompts,
        fetchPrompts,
    }
})
