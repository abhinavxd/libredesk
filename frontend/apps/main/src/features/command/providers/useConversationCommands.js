
import { defineMessage as msg } from '@lingui/core/macro'


import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from '@shared-ui/i18n'
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
  { minutes: 60, unitMessage: msg({ message: '{count, plural, one {Hour} other {Hours}}' }), count: 1 },
  { minutes: 180, unitMessage: msg({ message: '{count, plural, one {Hour} other {Hours}}' }), count: 3 },
  { minutes: 360, unitMessage: msg({ message: '{count, plural, one {Hour} other {Hours}}' }), count: 6 },
  { minutes: 720, unitMessage: msg({ message: '{count, plural, one {Hour} other {Hours}}' }), count: 12 },
  { minutes: 1440, unitMessage: msg({ message: '{count, plural, one {Day} other {Days}}' }), count: 1 },
  { minutes: 2880, unitMessage: msg({ message: '{count, plural, one {Day} other {Days}}' }), count: 2 },
  { minutes: 4320, unitMessage: msg({ message: '{count, plural, one {Day} other {Days}}' }), count: 3 },
  { minutes: 10080, unitMessage: msg({ message: '{count, plural, one {Week} other {Weeks}}' }), count: 1 }
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
  const i18n = useI18n()
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
      label: i18n._('Snooze'),
      section,
      icon: CalendarClock,
      permission: perms.CONVERSATIONS_UPDATE_STATUS,
      group: true,
      shortcut: ['Alt', 'Z'],
      placeholder: i18n._('Snooze for')
    },
    ...SNOOZE_PRESETS.map(({ minutes, unitMessage, count }) => ({
      id: `conv.snooze.${minutes}`,
      label: `${count} ${i18n._(unitMessage, count)}`,
      parent: SNOOZE_COMMAND,
      icon: CalendarClock,
      run: () => conversationStore.snoozeConversation(formatSnoozeDuration(minutes))
    })),
    {
      id: SNOOZE_CUSTOM_COMMAND,
      label: i18n._('Pick date and time'),
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
        label: i18n._('Resolve'),
        keywords: ['close', 'done'],
        section,
        icon: CheckCircle2,
        permission: perms.CONVERSATIONS_UPDATE_STATUS,
        shortcut: ['Alt', 'E'],
        run: () => conversationStore.updateStatus(conv.uuid, CONVERSATION_DEFAULT_STATUSES.RESOLVED)
      })
    }
    if (REOPENABLE_STATUSES.includes(current)) {
      commands.push({
        id: 'conv.reopen',
        label: i18n._('Reopen'),
        keywords: [i18n._('Open')],
        section,
        icon: RotateCcw,
        permission: perms.CONVERSATIONS_UPDATE_STATUS,
        shortcut: ['Alt', 'O'],
        run: () => conversationStore.updateStatus(conv.uuid, CONVERSATION_DEFAULT_STATUSES.OPEN)
      })
    }
    commands.push(...snoozeCommands())
    commands.push({
      id: 'conv.status',
      label: i18n._('Set status'),
      keywords: [i18n._('{count, plural, one {Status} other {Statuses}}')],
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
        run: isSnooze ? undefined : () => conversationStore.updateStatus(conv.uuid, status.label)
      })
    }
    return commands
  }

  const priorityCommands = (conv) => [
    {
      id: 'conv.priority',
      label: i18n._('Set priority'),
      keywords: [i18n._('{count, plural, one {Priority} other {Priorities}}')],
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
        label: i18n._('Assign to me'),
        keywords: ['self', 'me'],
        section,
        icon: UserCheck,
        permission: perms.CONVERSATIONS_UPDATE_USER_ASSIGNEE,
        run: () => assignAgent(userStore.userID)
      })
    }
    commands.push({
      id: 'conv.assign-agent',
      label: i18n._('Assign agent'),
      keywords: [i18n._('{count, plural, one {Agent} other {Agents}}')],
      section,
      icon: UserPlus,
      permission: perms.CONVERSATIONS_UPDATE_USER_ASSIGNEE,
      group: true,
      shortcut: ['Alt', 'A'],
      placeholder: i18n._('Search agents...'),
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
        label: i18n._('Unassign agent'),
        keywords: [i18n._('{count, plural, one {Agent} other {Agents}}'), 'remove'],
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
        label: i18n._('Assign team'),
        keywords: [i18n._('{count, plural, one {Team} other {Teams}}')],
        section,
        icon: Users,
        permission: perms.CONVERSATIONS_UPDATE_TEAM_ASSIGNEE,
        group: true,
        placeholder: i18n._('Search teams...'),
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
        label: i18n._('Suggest tags'),
        keywords: ['ai', i18n._('{count, plural, one {Tag} other {Tags}}')],
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
        label: i18n._('Unassign team'),
        keywords: [i18n._('{count, plural, one {Team} other {Teams}}'), 'remove'],
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
        label: i18n._('Add tags'),
        keywords: [i18n._('{count, plural, one {Tag} other {Tags}}')],
        section,
        icon: Tag,
        permission: perms.CONVERSATIONS_UPDATE_TAGS,
        group: true,
        placeholder: i18n._('Search tags...'),
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
          label: i18n._('Remove tags'),
          keywords: [i18n._('{count, plural, one {Tag} other {Tags}}')],
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
      label: i18n._('Switch to reply'),
      keywords: [i18n._('{count, plural, one {Reply} other {Replies}}')],
      section,
      icon: MessageSquare,
      permission: perms.MESSAGES_WRITE,
      shortcut: ['Alt', 'R'],
      run: () => emitter.emit(EMITTER_EVENTS.REPLY_BOX_SET_TYPE, 'reply')
    },
    {
      id: 'conv.private-note',
      label: i18n._('Switch to private note'),
      keywords: [i18n._('{count, plural, one {Private note} other {Private notes}}'), i18n._('{count, plural, one {Note} other {Notes}}')],
      section,
      icon: StickyNote,
      permission: perms.MESSAGES_WRITE_PRIVATE,
      shortcut: ['Alt', 'N'],
      run: () => emitter.emit(EMITTER_EVENTS.REPLY_BOX_SET_TYPE, 'private_note')
    },
    {
      id: 'conv.focus-reply',
      label: i18n._('Focus reply box'),
      keywords: [i18n._('{count, plural, one {Reply} other {Replies}}'), 'editor', 'compose'],
      section,
      icon: PenLine,
      permission: [perms.MESSAGES_WRITE, perms.MESSAGES_WRITE_PRIVATE],
      run: () => emitter.emit(EMITTER_EVENTS.REPLY_BOX_FOCUS)
    }
  ]

  const copyLink = async () => {
    await navigator.clipboard.writeText(window.location.href)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: i18n._('Copied to clipboard') })
  }

  const miscCommands = (conv) => [
    {
      id: 'conv.macro',
      label: i18n._('Apply macro'),
      keywords: [i18n._('{count, plural, one {Macro} other {Macros}}')],
      section,
      icon: Zap,
      shortcut: ['Ctrl', 'M'],
      navigateTo: MACROS_COMMAND
    },
    {
      id: 'conv.mark-unread',
      label: i18n._('Mark as unread'),
      keywords: ['unread'],
      section,
      icon: MailOpen,
      run: () => conversationStore.markAsUnread(conv.uuid)
    },
    {
      id: 'conv.copy-link',
      label: i18n._('Copy link'),
      keywords: [i18n._('Copy'), i18n._('{count, plural, one {Link} other {Links}}'), 'url'],
      section,
      icon: Link2,
      run: copyLink
    },
    {
      id: 'conv.transcript',
      label: i18n._('Download transcript'),
      keywords: ['export'],
      section,
      icon: Download,
      run: () =>
        emitter.emit(EMITTER_EVENTS.CONVERSATION_ACTION, CONVERSATION_ACTIONS.DOWNLOAD_TRANSCRIPT)
    },
    {
      id: 'conv.summarize',
      label: i18n._('Summarize with AI'),
      keywords: ['ai', i18n._('Summary')],
      section,
      icon: Sparkles,
      permission: perms.MESSAGES_WRITE_PRIVATE,
      run: () => emitter.emit(EMITTER_EVENTS.CONVERSATION_ACTION, CONVERSATION_ACTIONS.SUMMARIZE)
    },
    ...(conv.contact_id
      ? [
          {
            id: 'conv.open-contact',
            label: i18n._('Open contact'),
            keywords: [i18n._('{count, plural, one {Contact} other {Contacts}}')],
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
