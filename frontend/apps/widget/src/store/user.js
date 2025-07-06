import { defineStore } from 'pinia'
import { computed } from 'vue'
import { useStorage } from '@vueuse/core'
import { parseJWT } from '@shared-ui/utils/string'

export const useUserStore = defineStore('user', () => {
    const userSessionToken = useStorage('libredesk_session', "")

    const isVisitor = computed(() => {
        const token = userSessionToken.value
        // Token not set, assume visitor.
        if (!token) return true
        const jwt = parseJWT(token)
        return jwt.is_visitor
    })

    const clearSessionToken = () => {
        userSessionToken.value = ""
    }

    return {
        userSessionToken,
        isVisitor,
        clearSessionToken
    }
})