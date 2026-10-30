

import { computed } from 'vue'
import { useConversationStore } from '@/stores/conversation'
import { useInboxStore } from '@/stores/inbox'
import { useSlaStore } from '@/stores/sla'
import { useCustomAttributeStore } from '@/stores/customAttributes'
import { FIELD_TYPE, FIELD_OPERATORS } from '@/constants/filterConfig'
import { useI18n } from '@shared-ui/i18n'

export function useConversationFilters () {
    const cStore = useConversationStore()
    const iStore = useInboxStore()
    const slaStore = useSlaStore()
    const customAttributeStore = useCustomAttributeStore()
    const i18n = useI18n()

    const customAttributeDataTypeToFieldType = {
        'text': FIELD_TYPE.TEXT,
        'number': FIELD_TYPE.NUMBER,
        'checkbox': FIELD_TYPE.BOOLEAN,
        'date': FIELD_TYPE.DATE,
        'link': FIELD_TYPE.TEXT,
        'list': FIELD_TYPE.SELECT,
    }

    const customAttributeDataTypeToFieldOperators = {
        'text': FIELD_OPERATORS.TEXT_AUTOMATION,
        'number': FIELD_OPERATORS.NUMBER,
        'checkbox': FIELD_OPERATORS.BOOLEAN,
        'date': FIELD_OPERATORS.DATE_AUTOMATION,
        'link': FIELD_OPERATORS.TEXT_AUTOMATION,
        'list': FIELD_OPERATORS.SELECT,
    }

    const conversationsListFilters = computed(() => ({
        status_id: {
            label: i18n._('{count, plural, one {Status} other {Statuses}}'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: cStore.statusOptions
        },
        priority_id: {
            label: i18n._('{count, plural, one {Priority} other {Priorities}}'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: cStore.priorityOptions
        },
        assigned_team_id: {
            label: i18n._('Assign team'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            entity: 'team'
        },
        assigned_user_id: {
            label: i18n._('Assign agent'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            entity: 'agent'
        },
        inbox_id: {
            label: i18n._('{count, plural, one {Inbox} other {Inboxes}}'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: iStore.options
        },
        tags: {
            label: i18n._('{count, plural, one {Tag} other {Tags}}', 2),
            type: FIELD_TYPE.MULTI_SELECT,
            operators: FIELD_OPERATORS.MULTI_SELECT,
            entity: 'tag'
        },
        created_at: {
            label: i18n._('Created at'),
            type: FIELD_TYPE.DATE,
            operators: FIELD_OPERATORS.DATE
        },
        waiting_since: {
            label: i18n._('Waiting since'),
            type: FIELD_TYPE.DATE,
            operators: FIELD_OPERATORS.DATE
        },
        snoozed_until: {
            label: i18n._('Snoozed until'),
            type: FIELD_TYPE.DATE,
            operators: FIELD_OPERATORS.DATE
        },
        last_message_at: {
            label: i18n._('Last message at'),
            type: FIELD_TYPE.DATE,
            operators: FIELD_OPERATORS.DATE
        },
        last_interaction_at: {
            label: i18n._('Last interaction at'),
            type: FIELD_TYPE.DATE,
            operators: FIELD_OPERATORS.DATE
        },
        next_sla_deadline_at: {
            label: i18n._('Next SLA deadline'),
            type: FIELD_TYPE.DATE,
            operators: FIELD_OPERATORS.DATE
        },
        email: {
            label: i18n._('Contact email'),
            type: FIELD_TYPE.TEXT,
            operators: FIELD_OPERATORS.TEXT,
            model: 'users'
        },
        external_user_id: {
            label: i18n._('Contact external ID'),
            type: FIELD_TYPE.TEXT,
            operators: FIELD_OPERATORS.TEXT_EXACT,
            model: 'users'
        },
        last_interaction_sender: {
            label: i18n._('Last interaction by'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: [
                { label: i18n._('{count, plural, one {Contact} other {Contacts}}'), value: 'contact' },
                { label: i18n._('{count, plural, one {Agent} other {Agents}}'), value: 'agent' }
            ]
        },
        sla_policy_id: {
            label: i18n._('{count, plural, one {SLA policy} other {SLA policies}}'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: slaStore.options
        },
        channel: {
            label: i18n._('Channel'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: [
                { label: i18n.ngettext('Email', 'Emails', 1, {}, 'Communication channel'), value: 'email' },
                { label: i18n._('Live Chat'), value: 'livechat' }
            ],
            model: 'inboxes'
        }
    }))

    const contactCustomAttributes = computed(() => {
        return customAttributeStore.contactAttributeOptions
            .filter(attribute => attribute.applies_to === 'contact')
            .reduce((acc, attribute) => {
                acc[attribute.key] = {
                    label: attribute.label,
                    type: customAttributeDataTypeToFieldType[attribute.data_type] || FIELD_TYPE.TEXT,
                    operators: customAttributeDataTypeToFieldOperators[attribute.data_type] || FIELD_OPERATORS.TEXT,
                    allowCaseSensitive: ['text', 'link'].includes(attribute.data_type),
                    options: attribute.values.map(value => ({
                        label: value,
                        value: value
                    })) || [],
                }
                return acc
            }, {})
    })

    const newConversationFilters = computed(() => ({
        contact_email: {
            label: i18n._('{count, plural, one {Email} other {Emails}}'),
            type: FIELD_TYPE.TEXT,
            operators: FIELD_OPERATORS.TEXT_AUTOMATION
        },
        to: {
            label: i18n._('{count, plural, one {To email address} other {To email addresses}}'),
            type: FIELD_TYPE.TEXT,
            operators: FIELD_OPERATORS.TEXT
        },
        content: {
            label: i18n._('{count, plural, one {Content} other {Contents}}'),
            type: FIELD_TYPE.TEXT,
            operators: FIELD_OPERATORS.TEXT_AUTOMATION,
            allowCaseSensitive: true
        },
        subject: {
            label: i18n._('{count, plural, one {Subject} other {Subjects}}'),
            type: FIELD_TYPE.TEXT,
            operators: FIELD_OPERATORS.TEXT_AUTOMATION,
            allowCaseSensitive: true
        },
        status: {
            label: i18n._('{count, plural, one {Status} other {Statuses}}'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: cStore.statusOptions
        },
        priority: {
            label: i18n._('{count, plural, one {Priority} other {Priorities}}'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: cStore.priorityOptions
        },
        assigned_team: {
            label: i18n._('Assign team'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            entity: 'team'
        },
        assigned_user: {
            label: i18n._('Assign agent'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            entity: 'agent'
        },
        inbox: {
            label: i18n._('{count, plural, one {Inbox} other {Inboxes}}'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: iStore.options
        }
    }))

    const conversationFilters = computed(() => ({
        status: {
            label: i18n._('{count, plural, one {Status} other {Statuses}}'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: cStore.statusOptions
        },
        priority: {
            label: i18n._('{count, plural, one {Priority} other {Priorities}}'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: cStore.priorityOptions
        },
        assigned_team: {
            label: i18n._('Assign team'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            entity: 'team'
        },
        assigned_user: {
            label: i18n._('Assign agent'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            entity: 'agent'
        },
        hours_since_created: {
            label: i18n._('Hours since created'),
            type: FIELD_TYPE.NUMBER,
            operators: FIELD_OPERATORS.NUMBER
        },
        hours_since_first_reply: {
            label: i18n._('Hours since first reply'),
            type: FIELD_TYPE.NUMBER,
            operators: FIELD_OPERATORS.NUMBER
        },
        hours_since_last_reply: {
            label: i18n._('Hours since last reply'),
            type: FIELD_TYPE.NUMBER,
            operators: FIELD_OPERATORS.NUMBER
        },
        hours_since_resolved: {
            label: i18n._('Hours since resolved'),
            type: FIELD_TYPE.NUMBER,
            operators: FIELD_OPERATORS.NUMBER
        },
        hours_since_last_resolved: {
            label: i18n._('Hours since last resolved'),
            type: FIELD_TYPE.NUMBER,
            operators: FIELD_OPERATORS.NUMBER
        },
        inbox: {
            label: i18n._('{count, plural, one {Inbox} other {Inboxes}}'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: iStore.options
        },
        previous_status: {
            label: i18n._('Previous status'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: cStore.statusOptions
        },
        previous_priority: {
            label: i18n._('Previous priority'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: cStore.priorityOptions
        },
        previous_assigned_user: {
            label: i18n._('Previous assigned user'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            entity: 'agent'
        },
        previous_assigned_team: {
            label: i18n._('Previous assigned team'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            entity: 'team'
        }
    }))

    const conversationActions = computed(() => ({
        assign_team: {
            label: i18n._('Assign team'),
            type: FIELD_TYPE.SELECT,
            entity: 'team'
        },
        assign_user: {
            label: i18n._('Assign agent'),
            type: FIELD_TYPE.SELECT,
            entity: 'agent'
        },
        set_status: {
            label: i18n._('Set status'),
            type: FIELD_TYPE.SELECT,
            options: cStore.statusOptionsNoSnooze
        },
        set_priority: {
            label: i18n._('Set priority'),
            type: FIELD_TYPE.SELECT,
            options: cStore.priorityOptions
        },
        send_private_note: {
            label: i18n._('Add private note'),
            type: FIELD_TYPE.RICHTEXT
        },
        send_reply: {
            label: i18n._('Send reply'),
            type: FIELD_TYPE.RICHTEXT
        },
        send_csat: {
            label: i18n._('Send CSAT'),
        },
        set_sla: {
            label: i18n._('Set SLA'),
            type: FIELD_TYPE.SELECT,
            options: slaStore.options
        },
        add_tags: {
            label: i18n._('Add tags'),
            type: FIELD_TYPE.TAG
        },
        set_tags: {
            label: i18n._('Set tags'),
            type: FIELD_TYPE.TAG
        },
        remove_tags: {
            label: i18n._('Remove tags'),
            type: FIELD_TYPE.TAG
        },
        notify: {
            label: i18n._('Notify agents'),
            type: FIELD_TYPE.RECIPIENTS
        },
        snooze: {
            label: i18n._('Snooze'),
            type: FIELD_TYPE.TEXT
        },
        trigger_webhook: {
            label: i18n._('Trigger webhook'),
            type: FIELD_TYPE.WEBHOOK
        }
    }))

    const macroActions = computed(() => ({
        assign_team: {
            label: i18n._('Assign team'),
            type: FIELD_TYPE.SELECT,
            entity: 'team'
        },
        assign_user: {
            label: i18n._('Assign agent'),
            type: FIELD_TYPE.SELECT,
            entity: 'agent'
        },
        set_status: {
            label: i18n._('Set status'),
            type: FIELD_TYPE.SELECT,
            options: cStore.statusOptionsNoSnooze
        },
        set_priority: {
            label: i18n._('Set priority'),
            type: FIELD_TYPE.SELECT,
            options: cStore.priorityOptions
        },
        add_tags: {
            label: i18n._('Add tags'),
            type: FIELD_TYPE.TAG
        },
        set_tags: {
            label: i18n._('Set tags'),
            type: FIELD_TYPE.TAG
        },
        remove_tags: {
            label: i18n._('Remove tags'),
            type: FIELD_TYPE.TAG
        }
    }))


    return {
        conversationsListFilters,
        conversationFilters,
        newConversationFilters,
        conversationActions,
        macroActions,
        contactCustomAttributes,
    }
}
