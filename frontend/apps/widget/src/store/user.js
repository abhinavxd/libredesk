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

    const userID = computed(() => {
        const token = userSessionToken.value
        if (!token) return null
        const jwt = parseJWT(token)
        return jwt.user_id || null
    })

    const clearSessionToken = () => {
        userSessionToken.value = ""
    }

    const setSessionToken = (token) => {
        if (typeof token !== 'string') {
            throw new Error('Session token must be a string')
        }
        userSessionToken.value = token
    }

    return {
        userSessionToken,
        isVisitor,
        userID,
        clearSessionToken,
        setSessionToken
    }
})