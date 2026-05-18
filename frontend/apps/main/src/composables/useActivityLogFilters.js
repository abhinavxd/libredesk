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
            label: t('activityLog.entryType'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: [{
                label: t('activityLog.entryType.agentLogin'),
                value: 'agent_login'
            }, {
                label: t('activityLog.entryType.agentLogout'),
                value: 'agent_logout'
            }, {
                label: t('activityLog.entryType.agentAway'),
                value: 'agent_away'
            }, {
                label: t('activityLog.entryType.agentAwayReassigned'),
                value: 'agent_away_reassigned'
            }, {
                label: t('activityLog.entryType.agentOnline'),
                value: 'agent_online'
            }, {
                label: t('activityLog.entryType.agentPasswordSet'),
                value: 'agent_password_set'
            }, {
                label: t('activityLog.entryType.agentRolePermissionsChanged'),
                value: 'agent_role_permissions_changed'
            }]
        },
    }))
    return {
        activityLogListFilters
    }
}
