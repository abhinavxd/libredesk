import { computed } from 'vue'
import { defineStore } from 'pinia'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import { createRemoteLookup } from '@/utils/remote-lookup'
import api from '@/api'

// TODO: rename this store to agents
export const useUsersStore = defineStore('users', () => {
    const emitter = useEmitter()
    const showFetchError = (error) => {
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
            variant: 'destructive',
            description: handleHTTPError(error).message
        })
    }
    const lookup = createRemoteLookup({
        fetchRows: (params) => api.getUsersCompact(params),
        onError: showFetchError
    })

    const toOptions = (users) => users.map(user => ({
        label: user.first_name + ' ' + user.last_name,
        value: String(user.id),
        type: user.type,
        avatar_url: user.avatar_url,
        availability_status: user.availability_status,
    }))
    const users = lookup.rows
    const options = computed(() => toOptions(users.value))
    const searchUsers = async (query) => toOptions(await lookup.search(query))

    const setAvailability = (agentID, status) => {
        lookup.patch(agentID, { availability_status: status })
    }

    return {
        users,
        options,
        fetchUsers: lookup.fetchFirstPage,
        searchUsers,
        ensureUserIDs: lookup.ensureIDs,
        setAvailability,
    }
})
