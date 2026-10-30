

import { computed } from 'vue'
import { FIELD_TYPE, FIELD_OPERATORS } from '../constants/filterConfig'
import { useI18n } from '@shared-ui/i18n'

export function useActivityLogFilters () {
    const i18n = useI18n()
    const activityLogListFilters = computed(() => ({
        actor_id: {
            label: i18n._('{count, plural, one {Actor} other {Actors}}'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            entity: 'agent'
        },
        activity_type: {
            label: i18n._('Log entry type'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: [{
                label: i18n._('Agent login'),
                value: 'agent_login'
            }, {
                label: i18n._('Agent logout'),
                value: 'agent_logout'
            }, {
                label: i18n._('Agent away'),
                value: 'agent_away'
            }, {
                label: i18n._('Agent away reassigned'),
                value: 'agent_away_reassigned'
            }, {
                label: i18n._('Agent online'),
                value: 'agent_online'
            }, {
                label: i18n._('Agent password set'),
                value: 'agent_password_set'
            }, {
                label: i18n._('Agent role permissions changed'),
                value: 'agent_role_permissions_changed'
            }, {
                label: i18n._('Contact deleted'),
                value: 'contact_deleted'
            }, {
                label: i18n._('Contact data exported'),
                value: 'contact_data_exported'
            }]
        },
    }))
    return {
        activityLogListFilters
    }
}
