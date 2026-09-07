import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useEmitter } from '../composables/useEmitter'
import { EMITTER_EVENTS } from '../constants/emitterEvents'
import api from '../api'

export const useCustomAttributeStore = defineStore('customAttributes', () => {
    const attributes = ref([])
    const emitter = useEmitter()
    const contactAttributeOptions = computed(() => {
        return attributes.value
            .filter(att => att.applies_to === 'contact')
            .map(att => ({
                label: att.name,
                value: String(att.id),
                ...att,
            }))
    })
    const conversationAttributeOptions = computed(() => {
        return attributes.value
            .filter(att => att.applies_to === 'conversation')
            .map(att => ({
                label: att.name,
                value: String(att.id),
                ...att,
            }))
    })
    let inflight = null
    let hasFetched = false
    const fetchCustomAttributes = () => {
        if (hasFetched) return Promise.resolve()
        if (inflight) return inflight
        inflight = api.getCustomAttributes()
            .then(response => {
                attributes.value = response?.data?.data || []
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
        attributes,
        conversationAttributeOptions,
        contactAttributeOptions,
        fetchCustomAttributes,
    }
})