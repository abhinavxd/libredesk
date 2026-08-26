import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useEmitter } from '../composables/useEmitter'
import { EMITTER_EVENTS } from '../constants/emitterEvents'
import { fetchAllPages } from '../utils/paged-fetch'
import api from '../api'

export const useTagStore = defineStore('tags', () => {
    const tags = ref([])
    const emitter = useEmitter()
    const tagNames = computed(() => tags.value.map(tag => tag.name))
    const tagOptions = computed(() => tags.value.map(tag => ({
        label: tag.name,
        value: String(tag.id),
    })))

    const showFetchError = (error) => {
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
            variant: 'destructive',
            description: handleHTTPError(error).message
        })
    }
    const fetchTags = async () => {
        if (tags.value.length) return
        try {
            await fetchAllPages(
                (params) => api.getTags(params),
                (rows) => tags.value.push(...rows),
                showFetchError
            )
        } catch (error) {
            showFetchError(error)
        }
    }

    return {
        tags,
        tagOptions,
        tagNames,
        fetchTags,
    }
})