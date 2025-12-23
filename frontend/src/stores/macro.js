import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { handleHTTPError } from '@/utils/http'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import { useUserStore } from './user'
import api from '@/api'
import { permissions as perms } from '@/constants/permissions.js'

export const useMacroStore = defineStore('macroStore', () => {
    const macroList = ref([])
    const emitter = useEmitter()
    const userStore = useUserStore()
    const currentView = ref('')

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
        // Filter macros based on visibility set.
        const userTeams = userStore.teams.map(team => String(team.id))
        let filtered = macroList.value.filter(macro =>
            macro.visibility === 'all' ||
            userTeams.includes(macro.team_id) ||
            String(macro.user_id) === String(userStore.userID)
        )

        // Filter by visible_when if currentView is set.
        if (currentView.value) {
            filtered = filtered.filter(macro =>
                !macro.visible_when?.length || macro.visible_when.includes(currentView.value)
            )
        }

        // Filter macros based on permissions.
        filtered.forEach(macro => {
            macro.actions = macro.actions.filter(action => {
                const permission = actionPermissions[action.type]
                if (!permission) return true
                return userStore.can(permission)
            })
        })

        // Skip macros that do not have any actions left AND the macro field `message_content` is empty.
        filtered = filtered.filter(macro => !(macro.actions.length === 0 && macro.message_content === ""))

        return filtered.map(macro => ({
            ...macro,
            label: macro.name,
            value: String(macro.id),
        }))
    })

    const loadMacros = async (force = false) => {
        if (!force && macroList.value.length) return
        try {
            const response = await api.getAllMacros()
            macroList.value = response?.data?.data || []
        } catch (error) {
            emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
                variant: 'destructive',
                description: handleHTTPError(error).message
            })
        }
    }

    const setCurrentView = (view) => {
        currentView.value = view
    }

    return {
        macroList,
        macroOptions,
        loadMacros,
        setCurrentView
    }
})