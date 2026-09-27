import { ref } from 'vue'
import { defineStore } from 'pinia'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import api from '@/api'

export const useViewStore = defineStore('view', () => {
    const emitter = useEmitter()
    const views = ref([])

    const fetchViews = async () => {
        try {
            const response = await api.getCurrentUserViews()
            views.value = response.data.data
        } catch (error) {
            emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
                variant: 'destructive',
                description: handleHTTPError(error).message
            })
        }
    }

    return { views, fetchViews }
})
