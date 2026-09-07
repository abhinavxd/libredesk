import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import api from '@main/api'

export const useWebhookStore = defineStore('webhook', () => {
    const webhooks = ref([])
    const emitter = useEmitter()

    const options = computed(() => webhooks.value.map((w) => ({
        label: w.name,
        value: String(w.id)
    })))

    const fetchWebhooks = async () => {
        try {
            const response = await api.getWebhooksCompact()
            webhooks.value = response?.data?.data || []
        } catch (error) {
            emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
                variant: 'destructive',
                description: handleHTTPError(error).message
            })
        }
    }

    return {
        webhooks,
        options,
        fetchWebhooks
    }
})
