import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  CalendarClock,
  CheckCircle2,
  CircleDot,
  Contact,
  Download,
  Flag,
  Link2,
  MailOpen,
  MessageSquare,
  PenLine,
  RotateCcw,
  Sparkles,
  StickyNote,
  Tag,
  TagIcon,
  UserCheck,
  UserPlus,
  UserX,
  Users,
  UsersRound,
  Zap
} from 'lucide-vue-next'
import { useConversationStore } from '@main/stores/conversation'
import { useUserStore } from '@main/stores/user'
import { useUsersStore } from '@main/stores/users'
import { useTeamStore } from '@main/stores/team'
import { useTagStore } from '@main/stores/tag'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS, CONVERSATION_ACTIONS } from '@main/constants/emitterEvents'
import { permissions as perms } from '@main/constants/permissions'
import { CONVERSATION_DEFAULT_STATUSES, TAG_ACTION } from '@main/constants/conversation'
import { MACROS_COMMAND } from '../useCommandPalette'
import { SECTIONS } from '../sections'

export const SNOOZE_COMMAND = 'conv.snooze'
export const SNOOZE_CUSTOM_COMMAND = 'conv.snooze.custom'

const SNOOZE_PRESETS = [
  { minutes: 60, unitKey: 'globals.terms.hour', count: 1 },
  { minutes: 180, unitKey: 'globals.terms.hour', count: 3 },
  { minutes: 360, unitKey: 'globals.terms.hour', count: 6 },
  { minutes: 720, unitKey: 'globals.terms.hour', count: 12 },
  { minutes: 1440, unitKey: 'globals.terms.day', count: 1 },
  { minutes: 2880, unitKey: 'globals.terms.day', count: 2 },
  { minutes: 4320, unitKey: 'globals.terms.day', count: 3 },
  { minutes: 10080, unitKey: 'globals.terms.week', count: 1 }
]

const REOPENABLE_STATUSES = [
  CONVERSATION_DEFAULT_STATUSES.RESOLVED,
  CONVERSATION_DEFAULT_STATUSES.CLOSED,
  CONVERSATION_DEFAULT_STATUSES.SNOOZED
]

export const formatSnoozeDuration = (minutes) =>
  minutes % 60 === 0 && minutes >= 60 ? `${minutes / 60}h` : `${minutes}m`

export function useConversationCommands({ openSnoozeDatePicker }) {
  const router = useRouter()
  const { t } = useI18n()
  const emitter = useEmitter()
  const conversationStore = useConversationStore()
  const userStore = useUserStore()
  const usersStore = useUsersStore()
  const teamStore = useTeamStore()
  const tagStore = useTagStore()

  const section = SECTIONS.CONVERSATION

  const snoozeCommands = () => [
    {
      id: SNOOZE_COMMAND,
      label: t('globals.terms.snooze'),
      section,
      icon: CalendarClock,
      permission: perms.CONVERSATIONS_UPDATE_STATUS,
      group: true,
      shortcut: ['Alt', 'Z'],
      placeholder: t('command.snoozeFor')
    },
    ...SNOOZE_PRESETS.map(({ minutes, unitKey, count }) => ({
      id: `conv.snooze.${minutes}`,
      label: `${count} ${t(unitKey, count)}`,
      parent: SNOOZE_COMMAND,
      icon: CalendarClock,
      run: () => conversationStore.snoozeConversation(formatSnoozeDuration(minutes))
    })),
    {
      id: SNOOZE_CUSTOM_COMMAND,
      label: t('globals.messages.pickDateAndTime'),
      parent: SNOOZE_COMMAND,
      icon: CalendarClock,
      run: openSnoozeDatePicker
    }
  ]

  const statusCommands = (conv) => {
    const commands = []
    const current = conv.status
    if (current !== CONVERSATION_DEFAULT_STATUSES.RESOLVED) {
      commands.push({
        id: 'conv.resolve',
        label: t('globals.terms.resolve'),
        keywords: ['close', 'done'],
        section,
        icon: CheckCircle2,
        permission: perms.CONVERSATIONS_UPDATE_STATUS,
        shortcut: ['Alt', 'E'],
        run: () => conversationStore.updateStatus(CONVERSATION_DEFAULT_STATUSES.RESOLVED)
      })
    }
    if (REOPENABLE_STATUSES.includes(current)) {
      commands.push({
        id: 'conv.reopen',
        label: t('globals.terms.reopen'),
        keywords: [t('globals.terms.open')],
        section,
        icon: RotateCcw,
        permission: perms.CONVERSATIONS_UPDATE_STATUS,
        shortcut: ['Alt', 'O'],
        run: () => conversationStore.updateStatus(CONVERSATION_DEFAULT_STATUSES.OPEN)
      })
    }
    commands.push(...snoozeCommands())
    commands.push({
      id: 'conv.status',
      label: t('actions.setStatus'),
      keywords: [t('globals.terms.status')],
      section,
      icon: CircleDot,
      permission: perms.CONVERSATIONS_UPDATE_STATUS,
      group: true
    })
    for (const status of conversationStore.statusOptions) {
      if (status.label === current) continue
      const isSnooze = status.label === CONVERSATION_DEFAULT_STATUSES.SNOOZED
      commands.push({
        id: `conv.status.${status.value}`,
        label: status.label,
        parent: 'conv.status',
        icon: isSnooze ? CalendarClock : CircleDot,
        navigateTo: isSnooze ? SNOOZE_COMMAND : undefined,
        run: isSnooze ? undefined : () => conversationStore.updateStatus(status.label)
      })
    }
    return commands
  }

  const priorityCommands = (conv) => [
    {
      id: 'conv.priority',
      label: t('actions.setPriority'),
      keywords: [t('globals.terms.priority')],
      section,
      icon: Flag,
      permission: perms.CONVERSATIONS_UPDATE_PRIORITY,
      group: true,
      shortcut: ['Alt', 'P']
    },
    ...conversationStore.priorityOptions
      .filter((priority) => priority.label !== conv.priority)
      .map((priority) => ({
        id: `conv.priority.${priority.value}`,
        label: priority.label,
        parent: 'conv.priority',
        icon: Flag,
        run: () => {
          conv.priority = priority.label
          conv.priority_id = priority.value
          return conversationStore.updatePriority(priority.label)
        }
      }))
  ]

  const assignAgent = (id) => {
    conversationStore.current.assigned_user_id = id
    return conversationStore.updateAssignee('user', { assignee_id: parseInt(id) })
  }

  const agentCommands = (conv) => {
    const commands = []
    if (conv.assigned_user_id !== userStore.userID) {
      commands.push({
        id: 'conv.assign-me',
        label: t('conversation.assignSelfAction'),
        keywords: ['self', 'me'],
        section,
        icon: UserCheck,
        permission: perms.CONVERSATIONS_UPDATE_USER_ASSIGNEE,
        run: () => assignAgent(userStore.userID)
      })
    }
    commands.push({
      id: 'conv.assign-agent',
      label: t('actions.assignAgent'),
      keywords: [t('globals.terms.agent')],
      section,
      icon: UserPlus,
      permission: perms.CONVERSATIONS_UPDATE_USER_ASSIGNEE,
      group: true,
      shortcut: ['Alt', 'A'],
      placeholder: t('command.searchAgents'),
      search: async (term) =>
        (await usersStore.searchUsers(term))
          .filter((agent) => String(agent.value) !== String(conv.assigned_user_id))
          .map((agent) => ({
            id: `conv.assign-agent.${agent.value}`,
            label: agent.label,
            parent: 'conv.assign-agent',
            icon: UserPlus,
            run: () => assignAgent(agent.value)
          }))
    })
    if (conv.assigned_user_id) {
      commands.push({
        id: 'conv.unassign-agent',
        label: t('command.unassignAgent'),
        keywords: [t('globals.terms.agent'), 'remove'],
        section,
        icon: UserX,
        permission: perms.CONVERSATIONS_UPDATE_USER_ASSIGNEE,
        run: () => conversationStore.removeAssignee('user')
      })
    }
    return commands
  }

  const teamCommands = (conv) => {
    const commands = [
      {
        id: 'conv.assign-team',
        label: t('actions.assignTeam'),
        keywords: [t('globals.terms.team')],
        section,
        icon: Users,
        permission: perms.CONVERSATIONS_UPDATE_TEAM_ASSIGNEE,
        group: true,
        placeholder: t('command.searchTeams'),
        search: async (term) =>
          (await teamStore.searchTeams(term))
            .filter((team) => String(team.value) !== String(conv.assigned_team_id))
            .map((team) => ({
              id: `conv.assign-team.${team.value}`,
              label: team.label,
              parent: 'conv.assign-team',
              icon: UsersRound,
              run: () =>
                conversationStore.updateAssignee('team', { assignee_id: parseInt(team.value) })
            }))
      },
      {
        id: 'conv.suggest-tags',
        label: t('conversation.sidebar.suggestTags'),
        keywords: ['ai', t('globals.terms.tag')],
        section,
        icon: Sparkles,
        permission: perms.CONVERSATIONS_UPDATE_TAGS,
        run: () =>
          emitter.emit(EMITTER_EVENTS.CONVERSATION_ACTION, CONVERSATION_ACTIONS.SUGGEST_TAGS)
      }
    ]
    if (conv.assigned_team_id) {
      commands.push({
        id: 'conv.unassign-team',
        label: t('command.unassignTeam'),
        keywords: [t('globals.terms.team'), 'remove'],
        section,
        icon: UserX,
        permission: perms.CONVERSATIONS_UPDATE_TEAM_ASSIGNEE,
        run: () => conversationStore.removeAssignee('team')
      })
    }
    return commands
  }

  const tagCommands = (conv) => {
    const currentTags = conv.tags || []
    const commands = [
      {
        id: 'conv.add-tag',
        label: t('actions.addTags'),
        keywords: [t('globals.terms.tag')],
        section,
        icon: Tag,
        permission: perms.CONVERSATIONS_UPDATE_TAGS,
        group: true,
        placeholder: t('command.searchTags'),
        search: async (term) =>
          (await tagStore.searchTagNames(term))
            .filter((tag) => !currentTags.includes(tag.value))
            .map((tag) => ({
              id: `conv.add-tag.${tag.value}`,
              label: tag.label,
              parent: 'conv.add-tag',
              icon: TagIcon,
              run: () =>
                conversationStore.updateConversationTags(conv.uuid, TAG_ACTION.ADD, [tag.value])
            }))
      }
    ]
    if (currentTags.length) {
      commands.push(
        {
          id: 'conv.remove-tag',
          label: t('actions.removeTags'),
          keywords: [t('globals.terms.tag')],
          section,
          icon: Tag,
          permission: perms.CONVERSATIONS_UPDATE_TAGS,
          group: true
        },
        ...currentTags.map((tag) => ({
          id: `conv.remove-tag.${tag}`,
          label: tag,
          parent: 'conv.remove-tag',
          icon: TagIcon,
          run: () => conversationStore.updateConversationTags(conv.uuid, TAG_ACTION.REMOVE, [tag])
        }))
      )
    }
    return commands
  }

  const composeCommands = () => [
    {
      id: 'conv.reply',
      label: t('command.switchToReply'),
      keywords: [t('globals.terms.reply')],
      section,
      icon: MessageSquare,
      permission: perms.MESSAGES_WRITE,
      shortcut: ['Alt', 'R'],
      run: () => emitter.emit(EMITTER_EVENTS.REPLY_BOX_SET_TYPE, 'reply')
    },
    {
      id: 'conv.private-note',
      label: t('command.switchToPrivateNote'),
      keywords: [t('globals.terms.privateNote'), t('globals.terms.note')],
      section,
      icon: StickyNote,
      permission: perms.MESSAGES_WRITE_PRIVATE,
      shortcut: ['Alt', 'N'],
      run: () => emitter.emit(EMITTER_EVENTS.REPLY_BOX_SET_TYPE, 'private_note')
    },
    {
      id: 'conv.focus-reply',
      label: t('command.focusReplyBox'),
      keywords: [t('globals.terms.reply'), 'editor', 'compose'],
      section,
      icon: PenLine,
      permission: [perms.MESSAGES_WRITE, perms.MESSAGES_WRITE_PRIVATE],
      run: () => emitter.emit(EMITTER_EVENTS.REPLY_BOX_FOCUS)
    }
  ]

  const copyLink = async () => {
    await navigator.clipboard.writeText(window.location.href)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: t('globals.messages.copied') })
  }

  const miscCommands = (conv) => [
    {
      id: 'conv.macro',
      label: t('actions.applyMacro'),
      keywords: [t('globals.terms.macro')],
      section,
      icon: Zap,
      shortcut: ['Ctrl', 'M'],
      navigateTo: MACROS_COMMAND
    },
    {
      id: 'conv.mark-unread',
      label: t('globals.messages.markAsUnread'),
      keywords: ['unread'],
      section,
      icon: MailOpen,
      run: () => conversationStore.markAsUnread(conv.uuid)
    },
    {
      id: 'conv.copy-link',
      label: t('globals.messages.copyLink'),
      keywords: [t('globals.terms.copy'), t('globals.terms.link'), 'url'],
      section,
      icon: Link2,
      run: copyLink
    },
    {
      id: 'conv.transcript',
      label: t('conversation.downloadTranscript'),
      keywords: ['export'],
      section,
      icon: Download,
      run: () =>
        emitter.emit(EMITTER_EVENTS.CONVERSATION_ACTION, CONVERSATION_ACTIONS.DOWNLOAD_TRANSCRIPT)
    },
    {
      id: 'conv.summarize',
      label: t('conversation.summarize'),
      keywords: ['ai', t('globals.terms.summary')],
      section,
      icon: Sparkles,
      permission: perms.MESSAGES_WRITE_PRIVATE,
      run: () => emitter.emit(EMITTER_EVENTS.CONVERSATION_ACTION, CONVERSATION_ACTIONS.SUMMARIZE)
    },
    ...(conv.contact_id
      ? [
          {
            id: 'conv.open-contact',
            label: t('command.openContact'),
            keywords: [t('globals.terms.contact')],
            section,
            icon: Contact,
            permission: perms.CONTACTS_READ,
            run: () => router.push({ name: 'contact-detail', params: { id: conv.contact_id } })
          }
        ]
      : [])
  ]

  return computed(() => {
    if (!conversationStore.isConversationOpen) return []
    const conv = conversationStore.current
    if (!conv) return []
    return [
      ...statusCommands(conv),
      ...priorityCommands(conv),
      ...agentCommands(conv),
      ...teamCommands(conv),
      ...tagCommands(conv),
      ...miscCommands(conv),
      ...composeCommands()
    ]
  })
}
