import { computed } from 'vue'
import { useUsersStore } from '../stores/users'
import { FIELD_TYPE, FIELD_OPERATORS } from '../constants/filterConfig'
import { useI18n } from 'vue-i18n'

export function useActivityLogFilters () {
    const uStore = useUsersStore()
    const { t } = useI18n()
    const activityLogListFilters = computed(() => ({
        actor_id: {
            label: t('globals.terms.actor'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: uStore.options
        },
        activity_type: {
            label: t('globals.messages.type', {
                name: t('globals.terms.activityLog')
            }),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: [{
                label: 'Agent login',
                value: 'agent_login'
            }, {
                label: 'Agent logout',
                value: 'agent_logout'
            }, {
                label: 'Agent away',
                value: 'agent_away'
            }, {
                label: 'Agent away reassigned',
                value: 'agent_away_reassigned'
            }, {
                label: 'Agent online',
                value: 'agent_online'
            }]
        },
    }))
    return {
        activityLogListFilters
    }
}