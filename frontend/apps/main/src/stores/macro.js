import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useEmitter } from '../composables/useEmitter'
import { EMITTER_EVENTS } from '../constants/emitterEvents'
import { useUserStore } from './user'
import api from '../api'
import { permissions as perms } from '../constants/permissions.js'

export const useMacroStore = defineStore('macroStore', () => {
    const searchResults = ref([])
    const searchLoading = ref(false)
    // Per-macro message content, fetched on demand (the search response carries none).
    const macroContents = ref({})
    const contentFetches = {}
    const emitter = useEmitter()
    const userStore = useUserStore()
    const currentView = ref('')
    let searchSeq = 0

    // actionPermissions is a map of action names to their corresponding permissions that a user must have to perform the action.
    const actionPermissions = {
        assign_team: perms.CONVERSATIONS_UPDATE_TEAM_ASSIGNEE,
        assign_user: perms.CONVERSATIONS_UPDATE_USER_ASSIGNEE,
        set_status: perms.CONVERSATIONS_UPDATE_STATUS,
        set_priority: perms.CONVERSATIONS_UPDATE_PRIORITY,
        send_private_note: perms.MESSAGES_WRITE,
        send_reply: perms.MESSAGES_WRITE,
        add_tags: perms.CONVERSATIONS_UPDATE_TAGS,
        set_tags: perms.CONVERSATIONS_UPDATE_TAGS,
        remove_tags: perms.CONVERSATIONS_UPDATE_TAGS,
    }

    const macroOptions = computed(() => {
        let filtered = searchResults.value.map(macro => ({
            ...macro,
            actions: macro.actions.filter(action => {
                const permission = actionPermissions[action.type]
                if (!permission) return true
                return userStore.can(permission)
            })
        }))

        // Skip macros that do not have any actions left AND no message content.
        filtered = filtered.filter(macro => !(macro.actions.length === 0 && !macro.has_message_content))

        return filtered.map(macro => ({
            ...macro,
            label: macro.name,
            value: String(macro.id),
        }))
    })

    const searchMacros = async (query = '') => {
        const seq = ++searchSeq
        searchLoading.value = true
        try {
            const response = await api.searchMacros({ q: query, view: currentView.value })
            if (seq !== searchSeq) return
            searchResults.value = response?.data?.data || []
        } catch (error) {
            if (seq !== searchSeq) return
            emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
                variant: 'destructive',
                description: handleHTTPError(error).message
            })
        } finally {
            if (seq === searchSeq) searchLoading.value = false
        }
    }

    const fetchMacroContent = (id) => {
        if (id in macroContents.value) return Promise.resolve(macroContents.value[id])
        if (!contentFetches[id]) {
            contentFetches[id] = api.getMacro(id)
                .then(response => {
                    const content = response?.data?.data?.message_content || ''
                    macroContents.value[id] = content
                    return content
                })
                .finally(() => {
                    delete contentFetches[id]
                })
        }
        return contentFetches[id]
    }

    const clearCache = () => {
        searchResults.value = []
        macroContents.value = {}
    }

    const setCurrentView = (view) => {
        currentView.value = view
    }

    return {
        searchResults,
        searchLoading,
        macroOptions,
        macroContents,
        currentView,
        searchMacros,
        fetchMacroContent,
        clearCache,
        setCurrentView
    }
})
