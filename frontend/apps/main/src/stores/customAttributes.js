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
    const fetchCustomAttributes = async () => {
        if (attributes.value.length) return
        try {
            const response = await api.getCustomAttributes()
            attributes.value = response?.data?.data || []
        } catch (error) {
            emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
                variant: 'destructive',
                description: handleHTTPError(error).message
            })
        }
    }
    return {
        attributes,
        conversationAttributeOptions,
        contactAttributeOptions,
        fetchCustomAttributes,
    }
})