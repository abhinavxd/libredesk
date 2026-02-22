import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useEmitter } from '../composables/useEmitter'
import { EMITTER_EVENTS } from '../constants/emitterEvents'
import api from '../api'

export const useTeamStore = defineStore('team', () => {
    const teams = ref([])
    const emitter = useEmitter()
    const options = computed(() => teams.value.map(team => ({
        label: team.name,
        value: String(team.id),
        emoji: team.emoji,
    })))
    const fetchTeams = async () => {
        if (teams.value.length) return
        try {
            const response = await api.getTeamsCompact()
            teams.value = response?.data?.data || []
        } catch (error) {
            emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
                variant: 'destructive',
                description: handleHTTPError(error).message
            })
        }
    }
    return {
        teams,
        options,
        fetchTeams,
    }
})