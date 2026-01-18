import { computed } from 'vue'
import { useConversationStore } from '@/stores/conversation'
import { useInboxStore } from '@/stores/inbox'
import { useUsersStore } from '@/stores/users'
import { useTeamStore } from '@/stores/team'
import { useSlaStore } from '@/stores/sla'
import { useCustomAttributeStore } from '@/stores/customAttributes'
import { useTagStore } from '@/stores/tag'
import { FIELD_TYPE, FIELD_OPERATORS } from '@/constants/filterConfig'
import { useI18n } from 'vue-i18n'

export function useConversationFilters () {
    const cStore = useConversationStore()
    const iStore = useInboxStore()
    const uStore = useUsersStore()
    const tStore = useTeamStore()
    const slaStore = useSlaStore()
    const customAttributeStore = useCustomAttributeStore()
    const tagStore = useTagStore()
    const { t } = useI18n()

    const customAttributeDataTypeToFieldType = {
        'text': FIELD_TYPE.TEXT,
        'number': FIELD_TYPE.NUMBER,
        'checkbox': FIELD_TYPE.BOOLEAN,
        'date': FIELD_TYPE.DATE,
        'link': FIELD_TYPE.TEXT,
        'list': FIELD_TYPE.SELECT,
    }

    const customAttributeDataTypeToFieldOperators = {
        'text': FIELD_OPERATORS.TEXT,
        'number': FIELD_OPERATORS.NUMBER,
        'checkbox': FIELD_OPERATORS.BOOLEAN,
        'date': FIELD_OPERATORS.DATE,
        'link': FIELD_OPERATORS.TEXT,
        'list': FIELD_OPERATORS.SELECT,
    }

    const conversationsListFilters = computed(() => ({
        status_id: {
            label: t('globals.terms.status'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: cStore.statusOptions
        },
        priority_id: {
            label: t('globals.terms.priority'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: cStore.priorityOptions
        },
        assigned_team_id: {
            label: t('globals.messages.assign', {
                name: t('globals.terms.team').toLowerCase()
            }),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: tStore.options
        },
        assigned_user_id: {
            label: t('globals.messages.assign', {
                name: t('globals.terms.agent').toLowerCase()
            }),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: uStore.options
        },
        inbox_id: {
            label: t('globals.terms.inbox'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: iStore.options
        },
        tags: {
            label: t('globals.terms.tag', 2),
            type: FIELD_TYPE.MULTI_SELECT,
            operators: FIELD_OPERATORS.MULTI_SELECT,
            options: tagStore.tagOptions
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
            label: t('globals.terms.email'),
            type: FIELD_TYPE.TEXT,
            operators: FIELD_OPERATORS.TEXT
        },
        content: {
            label: t('globals.terms.content'),
            type: FIELD_TYPE.TEXT,
            operators: FIELD_OPERATORS.TEXT
        },
        subject: {
            label: t('globals.terms.subject'),
            type: FIELD_TYPE.TEXT,
            operators: FIELD_OPERATORS.TEXT
        },
        status: {
            label: t('globals.terms.status'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: cStore.statusOptions
        },
        priority: {
            label: t('globals.terms.priority'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: cStore.priorityOptions
        },
        assigned_team: {
            label: t('globals.messages.assign', {
                name: t('globals.terms.team').toLowerCase()
            }),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: tStore.options
        },
        assigned_user: {
            label: t('globals.messages.assign', {
                name: t('globals.terms.agent').toLowerCase()
            }),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: uStore.options
        },
        inbox: {
            label: t('globals.terms.inbox'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: iStore.options
        }
    }))

    const conversationFilters = computed(() => ({
        status: {
            label: t('globals.terms.status'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: cStore.statusOptions
        },
        priority: {
            label: t('globals.terms.priority'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: cStore.priorityOptions
        },
        assigned_team: {
            label: t('globals.messages.assign', {
                name: t('globals.terms.team').toLowerCase()
            }),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: tStore.options
        },
        assigned_user: {
            label: t('globals.messages.assign', {
                name: t('globals.terms.agent').toLowerCase()
            }),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: uStore.options
        },
        hours_since_created: {
            label: t('globals.messages.hoursSinceCreated'),
            type: FIELD_TYPE.NUMBER,
            operators: FIELD_OPERATORS.NUMBER
        },
        hours_since_first_reply: {
            label: t('globals.messages.hoursSinceFirstReply'),
            type: FIELD_TYPE.NUMBER,
            operators: FIELD_OPERATORS.NUMBER
        },
        hours_since_last_reply: {
            label: t('globals.messages.hoursSinceLastReply'),
            type: FIELD_TYPE.NUMBER,
            operators: FIELD_OPERATORS.NUMBER
        },
        hours_since_resolved: {
            label: t('globals.messages.hoursSinceResolved'),
            type: FIELD_TYPE.NUMBER,
            operators: FIELD_OPERATORS.NUMBER
        },
        inbox: {
            label: t('globals.terms.inbox'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: iStore.options
        }
    }))

    const conversationActions = computed(() => ({
        assign_team: {
            label: t('globals.messages.assign', {
                name: t('globals.terms.team').toLowerCase()
            }),
            type: FIELD_TYPE.SELECT,
            options: tStore.options
        },
        assign_user: {
            label: t('globals.messages.assign', {
                name: t('globals.terms.agent').toLowerCase()
            }),
            type: FIELD_TYPE.SELECT,
            options: uStore.options
        },
        set_status: {
            label: t('globals.messages.set', {
                name: t('globals.terms.status').toLowerCase()
            }),
            type: FIELD_TYPE.SELECT,
            options: cStore.statusOptionsNoSnooze
        },
        set_priority: {
            label: t('globals.messages.set', {
                name: t('globals.terms.priority').toLowerCase()
            }),
            type: FIELD_TYPE.SELECT,
            options: cStore.priorityOptions
        },
        send_private_note: {
            label: t('globals.messages.send', {
                name: t('globals.terms.privateNote').toLowerCase()
            }),
            type: FIELD_TYPE.RICHTEXT
        },
        send_reply: {
            label: t('globals.messages.send', {
                name: t('globals.terms.reply').toLowerCase()
            }),
            type: FIELD_TYPE.RICHTEXT
        },
        send_csat: {
            label: t('globals.messages.send', {
                name: t('globals.terms.csat').toLowerCase()
            }),
        },
        set_sla: {
            label: t('globals.messages.set', {
                name: t('globals.terms.sla').toLowerCase()
            }),
            type: FIELD_TYPE.SELECT,
            options: slaStore.options
        },
        add_tags: {
            label: t('globals.messages.add', {
                name: t('globals.terms.tag', 2).toLowerCase()
            }),
            type: FIELD_TYPE.TAG
        },
        set_tags: {
            label: t('globals.messages.set', {
                name: t('globals.terms.tag', 2).toLowerCase()
            }),
            type: FIELD_TYPE.TAG
        },
        remove_tags: {
            label: t('globals.messages.remove', {
                name: t('globals.terms.tag', 2).toLowerCase()
            }),
            type: FIELD_TYPE.TAG
        }
    }))

    const macroActions = computed(() => ({
        assign_team: {
            label: t('globals.messages.assign', {
                name: t('globals.terms.team').toLowerCase()
            }),
            type: FIELD_TYPE.SELECT,
            options: tStore.options
        },
        assign_user: {
            label: t('globals.messages.assign', {
                name: t('globals.terms.agent').toLowerCase()
            }),
            type: FIELD_TYPE.SELECT,
            options: uStore.options
        },
        set_status: {
            label: t('globals.messages.set', {
                name: t('globals.terms.status').toLowerCase()
            }),
            type: FIELD_TYPE.SELECT,
            options: cStore.statusOptionsNoSnooze
        },
        set_priority: {
            label: t('globals.messages.set', {
                name: t('globals.terms.priority').toLowerCase()
            }),
            type: FIELD_TYPE.SELECT,
            options: cStore.priorityOptions
        },
        add_tags: {
            label: t('globals.messages.add', {
                name: t('globals.terms.tag', 2).toLowerCase()
            }),
            type: FIELD_TYPE.TAG
        },
        set_tags: {
            label: t('globals.messages.set', {
                name: t('globals.terms.tag', 2).toLowerCase()
            }),
            type: FIELD_TYPE.TAG
        },
        remove_tags: {
            label: t('globals.messages.remove', {
                name: t('globals.terms.tag', 2).toLowerCase()
            }),
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