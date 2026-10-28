import { computed } from 'vue'
import { useConversationStore } from '@/stores/conversation'
import { useInboxStore } from '@/stores/inbox'
import { useSlaStore } from '@/stores/sla'
import { useCustomAttributeStore } from '@/stores/customAttributes'
import { FIELD_TYPE, FIELD_OPERATORS } from '@/constants/filterConfig'
import { useI18n } from 'vue-i18n'

export function useConversationFilters () {
    const cStore = useConversationStore()
    const iStore = useInboxStore()
    const slaStore = useSlaStore()
    const customAttributeStore = useCustomAttributeStore()
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
        'text': FIELD_OPERATORS.TEXT_AUTOMATION,
        'number': FIELD_OPERATORS.NUMBER,
        'checkbox': FIELD_OPERATORS.BOOLEAN,
        'date': FIELD_OPERATORS.DATE_AUTOMATION,
        'link': FIELD_OPERATORS.TEXT_AUTOMATION,
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
            label: t('actions.assignTeam'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            entity: 'team'
        },
        assigned_user_id: {
            label: t('actions.assignAgent'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            entity: 'agent'
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
            entity: 'tag'
        },
        created_at: {
            label: t('globals.terms.createdAt'),
            type: FIELD_TYPE.DATE,
            operators: FIELD_OPERATORS.DATE
        },
        waiting_since: {
            label: t('globals.terms.waitingSince'),
            type: FIELD_TYPE.DATE,
            operators: FIELD_OPERATORS.DATE
        },
        snoozed_until: {
            label: t('globals.terms.snoozedUntil'),
            type: FIELD_TYPE.DATE,
            operators: FIELD_OPERATORS.DATE
        },
        last_message_at: {
            label: t('globals.terms.lastMessageAt'),
            type: FIELD_TYPE.DATE,
            operators: FIELD_OPERATORS.DATE
        },
        last_interaction_at: {
            label: t('globals.terms.lastInteractionAt'),
            type: FIELD_TYPE.DATE,
            operators: FIELD_OPERATORS.DATE
        },
        next_sla_deadline_at: {
            label: t('globals.terms.nextSlaDeadline'),
            type: FIELD_TYPE.DATE,
            operators: FIELD_OPERATORS.DATE
        },
        email: {
            label: t('globals.terms.contactEmail'),
            type: FIELD_TYPE.TEXT,
            operators: FIELD_OPERATORS.TEXT,
            model: 'users'
        },
        external_user_id: {
            label: t('globals.terms.contactExternalId'),
            type: FIELD_TYPE.TEXT,
            operators: FIELD_OPERATORS.TEXT_EXACT,
            model: 'users'
        },
        last_interaction_sender: {
            label: t('globals.terms.lastInteractionBy'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: [
                { label: t('globals.terms.contact'), value: 'contact' },
                { label: t('globals.terms.agent'), value: 'agent' }
            ]
        },
        sla_policy_id: {
            label: t('globals.terms.slaPolicy'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: slaStore.options
        },
        channel: {
            label: t('globals.terms.channel'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: [
                { label: t('globals.terms.email'), value: 'email' },
                { label: t('globals.terms.liveChat'), value: 'livechat' }
            ],
            model: 'inboxes'
        }
    }))

    const attributeFilters = (attributes) => Object.fromEntries(attributes.map(attribute => [attribute.key, {
        label: attribute.label,
        type: customAttributeDataTypeToFieldType[attribute.data_type] || FIELD_TYPE.TEXT,
        operators: customAttributeDataTypeToFieldOperators[attribute.data_type] || FIELD_OPERATORS.TEXT,
        allowCaseSensitive: ['text', 'link'].includes(attribute.data_type),
        options: (attribute.values || []).map(value => ({ label: value, value })),
    }]))

    const contactCustomAttributes = computed(() => attributeFilters(customAttributeStore.contactAttributeOptions))
    const conversationCustomAttributes = computed(() => attributeFilters(customAttributeStore.conversationAttributeOptions))

    const baseAutomationFilters = computed(() => ({
        contact_email: {
            label: t('globals.terms.email'),
            type: FIELD_TYPE.TEXT,
            operators: FIELD_OPERATORS.TEXT_AUTOMATION
        },
        content: {
            label: t('globals.terms.content'),
            type: FIELD_TYPE.TEXT,
            operators: FIELD_OPERATORS.TEXT_AUTOMATION,
            allowCaseSensitive: true
        },
        subject: {
            label: t('globals.terms.subject'),
            type: FIELD_TYPE.TEXT,
            operators: FIELD_OPERATORS.TEXT_AUTOMATION,
            allowCaseSensitive: true
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
            label: t('actions.assignTeam'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            entity: 'team'
        },
        assigned_user: {
            label: t('actions.assignAgent'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            entity: 'agent'
        },
        inbox: {
            label: t('globals.terms.inbox'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: iStore.options
        },
        channel: {
            label: t('globals.terms.channel'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: [
                { label: t('globals.terms.email'), value: 'email' },
                { label: t('globals.terms.liveChat'), value: 'livechat' },
                { label: t('globals.terms.whatsapp'), value: 'whatsapp' }
            ]
        },
        tags: {
            label: t('globals.terms.tag', 2),
            type: FIELD_TYPE.MULTI_SELECT,
            operators: FIELD_OPERATORS.MULTI_SELECT,
            entity: 'tag'
        },
        status_category: {
            label: t('globals.terms.statusCategory'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: ['open', 'waiting', 'resolved'].map(value => ({ label: t(`globals.terms.${value}`), value }))
        },
        contact_country: {
            label: t('globals.terms.country'),
            type: FIELD_TYPE.TEXT,
            operators: FIELD_OPERATORS.TEXT_AUTOMATION
        },
        contact_external_id: {
            label: t('globals.terms.contactExternalId'),
            type: FIELD_TYPE.TEXT,
            operators: FIELD_OPERATORS.TEXT_EXACT
        },
        contact_phone: {
            label: t('globals.terms.phoneNumber'),
            type: FIELD_TYPE.TEXT,
            operators: FIELD_OPERATORS.TEXT_AUTOMATION
        },
        contact_type: {
            label: t('globals.terms.contactType'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: ['contact', 'visitor'].map(value => ({ label: t(`globals.terms.${value}`), value }))
        },
        within_business_hours: {
            label: t('admin.automation.withinBusinessHours'),
            type: FIELD_TYPE.BOOLEAN,
            operators: FIELD_OPERATORS.BOOLEAN
        }
    }))

    const newConversationFilters = computed(() => ({
        ...baseAutomationFilters.value,
        to: {
            label: t('globals.terms.toEmailAddress'),
            type: FIELD_TYPE.TEXT,
            operators: FIELD_OPERATORS.TEXT
        },
    }))

    const conversationFilters = computed(() => ({
        ...baseAutomationFilters.value,
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
        hours_since_last_resolved: {
            label: t('globals.messages.hoursSinceLastResolved'),
            type: FIELD_TYPE.NUMBER,
            operators: FIELD_OPERATORS.NUMBER
        },
        hours_waiting: {
            label: t('admin.automation.hoursWaiting'),
            type: FIELD_TYPE.NUMBER,
            operators: FIELD_OPERATORS.NUMBER_OPTIONAL
        },
        hours_since_last_message: {
            label: t('admin.automation.hoursSinceLastMessage'),
            type: FIELD_TYPE.NUMBER,
            operators: FIELD_OPERATORS.NUMBER_OPTIONAL
        },
        hours_since_last_incoming: {
            label: t('admin.automation.hoursSinceLastIncoming'),
            type: FIELD_TYPE.NUMBER,
            operators: FIELD_OPERATORS.NUMBER_OPTIONAL
        },
        hours_since_last_activity: {
            label: t('admin.automation.hoursSinceLastActivity'),
            type: FIELD_TYPE.NUMBER,
            operators: FIELD_OPERATORS.NUMBER_OPTIONAL
        },
        last_message_sender: {
            label: t('admin.automation.lastReplyBy'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: ['contact', 'agent'].map(value => ({ label: t(`globals.terms.${value}`), value }))
        },
        sla_policy: {
            label: t('globals.terms.slaPolicy'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: slaStore.options
        },
        sla_breached: {
            label: t('admin.automation.slaBreached'),
            type: FIELD_TYPE.BOOLEAN,
            operators: FIELD_OPERATORS.BOOLEAN
        },
        hours_until_sla_deadline: {
            label: t('admin.automation.hoursUntilSlaDeadline'),
            type: FIELD_TYPE.NUMBER,
            operators: FIELD_OPERATORS.NUMBER_OPTIONAL
        },
        csat_rating: {
            label: t('globals.terms.csatRating'),
            type: FIELD_TYPE.NUMBER,
            operators: FIELD_OPERATORS.NUMBER_OPTIONAL
        },
        csat_feedback: {
            label: t('globals.terms.csatFeedback'),
            type: FIELD_TYPE.TEXT,
            operators: FIELD_OPERATORS.TEXT_AUTOMATION,
            allowCaseSensitive: true
        },
        previous_status: {
            label: t('admin.automation.previousStatus'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: cStore.statusOptions
        },
        previous_priority: {
            label: t('admin.automation.previousPriority'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            options: cStore.priorityOptions
        },
        previous_assigned_user: {
            label: t('admin.automation.previousAssignedUser'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            entity: 'agent'
        },
        previous_assigned_team: {
            label: t('admin.automation.previousAssignedTeam'),
            type: FIELD_TYPE.SELECT,
            operators: FIELD_OPERATORS.SELECT,
            entity: 'team'
        }
    }))

    const conversationActions = computed(() => ({
        assign_team: {
            label: t('actions.assignTeam'),
            type: FIELD_TYPE.SELECT,
            entity: 'team'
        },
        assign_user: {
            label: t('actions.assignAgent'),
            type: FIELD_TYPE.SELECT,
            entity: 'agent'
        },
        set_status: {
            label: t('actions.setStatus'),
            type: FIELD_TYPE.SELECT,
            options: cStore.statusOptionsNoSnooze
        },
        set_priority: {
            label: t('actions.setPriority'),
            type: FIELD_TYPE.SELECT,
            options: cStore.priorityOptions
        },
        send_private_note: {
            label: t('actions.addPrivateNote'),
            type: FIELD_TYPE.RICHTEXT
        },
        send_reply: {
            label: t('actions.sendReply'),
            type: FIELD_TYPE.RICHTEXT
        },
        send_csat: {
            label: t('actions.sendCsat'),
        },
        set_sla: {
            label: t('actions.setSla'),
            type: FIELD_TYPE.SELECT,
            options: slaStore.options
        },
        add_tags: {
            label: t('actions.addTags'),
            type: FIELD_TYPE.TAG
        },
        set_tags: {
            label: t('actions.setTags'),
            type: FIELD_TYPE.TAG
        },
        remove_tags: {
            label: t('actions.removeTags'),
            type: FIELD_TYPE.TAG
        },
        notify: {
            label: t('actions.notify'),
            type: FIELD_TYPE.RECIPIENTS
        },
        snooze: {
            label: t('globals.terms.snooze'),
            type: FIELD_TYPE.TEXT
        },
        trigger_webhook: {
            label: t('actions.triggerWebhook'),
            type: FIELD_TYPE.WEBHOOK
        },
        unassign_user: { label: t('command.unassignAgent') },
        unassign_team: { label: t('command.unassignTeam') },
        clear_tags: { label: t('actions.clearTags') },
        set_subject: { label: t('actions.setSubject'), type: FIELD_TYPE.TEXT },
        set_contact_custom_attribute: { label: t('actions.setContactCustomAttribute'), type: 'custom_attribute', scope: 'contact' },
        remove_contact_custom_attribute: { label: t('actions.removeContactCustomAttribute'), type: 'custom_attribute', scope: 'contact', remove: true },
        set_conversation_custom_attribute: { label: t('actions.setConversationCustomAttribute'), type: 'custom_attribute', scope: 'conversation' },
        remove_conversation_custom_attribute: { label: t('actions.removeConversationCustomAttribute'), type: 'custom_attribute', scope: 'conversation', remove: true }
    }))

    const macroActions = computed(() => ({
        assign_team: {
            label: t('actions.assignTeam'),
            type: FIELD_TYPE.SELECT,
            entity: 'team'
        },
        assign_user: {
            label: t('actions.assignAgent'),
            type: FIELD_TYPE.SELECT,
            entity: 'agent'
        },
        set_status: {
            label: t('actions.setStatus'),
            type: FIELD_TYPE.SELECT,
            options: cStore.statusOptionsNoSnooze
        },
        set_priority: {
            label: t('actions.setPriority'),
            type: FIELD_TYPE.SELECT,
            options: cStore.priorityOptions
        },
        add_tags: {
            label: t('actions.addTags'),
            type: FIELD_TYPE.TAG
        },
        set_tags: {
            label: t('actions.setTags'),
            type: FIELD_TYPE.TAG
        },
        remove_tags: {
            label: t('actions.removeTags'),
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
        conversationCustomAttributes,
    }
}
