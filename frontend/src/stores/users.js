import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { handleHTTPError } from '@/utils/http'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import api from '@/api'

// TODO: rename this store to agents
export const useUsersStore = defineStore('users', () => {
    const users = ref([])
    const emitter = useEmitter()
    const options = computed(() => users.value.map(user => ({
        label: user.first_name + ' ' + user.last_name,
        value: String(user.id),
        avatar_url: user.avatar_url,
    })))
    const fetchUsers = async (force = false) => {
        if (!force && users.value.length) return
        try {
            const response = await api.getUsersCompact()
            users.value = response?.data?.data || []
        } catch (error) {
            emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
                variant: 'destructive',
                description: handleHTTPError(error).message
            })
        }
    }
    return {
        users,
        options,
        fetchUsers,
    }
})