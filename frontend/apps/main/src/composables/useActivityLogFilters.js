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
                label: t('activityLog.type.agentLogin'),
                value: 'agent_login'
            }, {
                label: t('activityLog.type.agentLogout'),
                value: 'agent_logout'
            }, {
                label: t('activityLog.type.agentAway'),
                value: 'agent_away'
            }, {
                label: t('activityLog.type.agentAwayReassigned'),
                value: 'agent_away_reassigned'
            }, {
                label: t('activityLog.type.agentOnline'),
                value: 'agent_online'
            }, {
                label: t('activityLog.type.agentPasswordSet'),
                value: 'agent_password_set'
            }, {
                label: t('activityLog.type.agentRolePermissionsChanged'),
                value: 'agent_role_permissions_changed'
            }]
        },
    }))
    return {
        activityLogListFilters
    }
}