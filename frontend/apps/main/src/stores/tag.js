import { computed } from 'vue'
import { defineStore } from 'pinia'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import { createRemoteLookup } from '@/utils/remote-lookup'
import api from '@/api'

export const useTagStore = defineStore('tags', () => {
    const emitter = useEmitter()
    const showFetchError = (error) => {
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
            variant: 'destructive',
            description: handleHTTPError(error).message
        })
    }
    const lookup = createRemoteLookup({
        fetchRows: (params) => api.getTags(params),
        onError: showFetchError
    })

    const tags = lookup.rows
    const tagNames = computed(() => tags.value.map(tag => tag.name))
    const tagOptions = computed(() => tags.value.map(tag => ({
        label: tag.name,
        value: String(tag.id),
    })))
    const searchTagNames = async (query) => (await lookup.search(query)).map(tag => ({
        label: tag.name,
        value: tag.name,
    }))
    const searchTagOptions = async (query) => (await lookup.search(query)).map(tag => ({
        label: tag.name,
        value: String(tag.id),
    }))

    return {
        tags,
        tagOptions,
        tagNames,
        fetchTags: lookup.fetchFirstPage,
        searchTagNames,
        searchTagOptions,
        ensureTagIDs: lookup.ensureIDs,
    }
})
