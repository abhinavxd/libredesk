import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useEmitter } from '../composables/useEmitter'
import { EMITTER_EVENTS } from '../constants/emitterEvents'
import { fetchAllPages } from '../utils/paged-fetch'
import api from '../api'

// TODO: rename this store to agents
export const useUsersStore = defineStore('users', () => {
    const users = ref([])
    const emitter = useEmitter()
    const options = computed(() => users.value.map(user => ({
        label: user.first_name + ' ' + user.last_name,
        value: String(user.id),
        type: user.type,
        avatar_url: user.avatar_url,
        availability_status: user.availability_status,
    })))
    let fetchSeq = 0
    const showFetchError = (error) => {
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
            variant: 'destructive',
            description: handleHTTPError(error).message
        })
    }
    const fetchUsers = async (force = false) => {
        if (!force && users.value.length) return
        const seq = ++fetchSeq
        try {
            users.value = []
            await fetchAllPages(
                (params) => api.getUsersCompact(params),
                (rows) => { if (seq === fetchSeq) users.value.push(...rows) },
                showFetchError
            )
        } catch (error) {
            showFetchError(error)
        }
    }
    const setAvailability = (agentID, status) => {
        const u = users.value.find(x => x.id === agentID)
        if (u) u.availability_status = status
    }
    return {
        users,
        options,
        fetchUsers,
        setAvailability,
    }
})