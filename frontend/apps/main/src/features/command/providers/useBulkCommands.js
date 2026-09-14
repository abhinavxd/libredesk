import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { CircleDot, Tag, TagIcon, UserPlus, Users, UsersRound, X } from 'lucide-vue-next'
import { useConversationStore } from '@main/stores/conversation'
import { useUsersStore } from '@main/stores/users'
import { useTeamStore } from '@main/stores/team'
import { useTagStore } from '@main/stores/tag'
import { useBulkActions } from '@main/composables/useBulkActions'
import { permissions as perms } from '@main/constants/permissions'
import { SECTIONS } from '../sections'

export function useBulkCommands() {
  const { t } = useI18n()
  const conversationStore = useConversationStore()
  const usersStore = useUsersStore()
  const teamStore = useTeamStore()
  const tagStore = useTagStore()
  const { bulkAssign, bulkAddTag, bulkUpdateStatus } = useBulkActions()

  const section = SECTIONS.BULK

  return computed(() => {
    if (conversationStore.selectedCount === 0) return []
    return [
      {
        id: 'bulk.status',
        label: t('actions.setStatus'),
        keywords: [t('globals.terms.status')],
        section,
        icon: CircleDot,
        permission: perms.CONVERSATIONS_UPDATE_STATUS,
        group: true
      },
      ...conversationStore.statusOptionsNoSnooze.map((status) => ({
        id: `bulk.status.${status.value}`,
        label: status.label,
        parent: 'bulk.status',
        icon: CircleDot,
        run: () => bulkUpdateStatus(status.label)
      })),
      {
        id: 'bulk.assign-agent',
        label: t('actions.assignAgent'),
        keywords: [t('globals.terms.agent')],
        section,
        icon: UserPlus,
        permission: perms.CONVERSATIONS_UPDATE_USER_ASSIGNEE,
        group: true,
        placeholder: t('command.searchAgents'),
        search: async (term) => [
          {
            id: 'bulk.assign-agent.none',
            label: t('globals.terms.none'),
            parent: 'bulk.assign-agent',
            icon: X,
            run: () => bulkAssign('user', 'none')
          },
          ...(await usersStore.searchUsers(term)).map((agent) => ({
            id: `bulk.assign-agent.${agent.value}`,
            label: agent.label,
            parent: 'bulk.assign-agent',
            icon: UserPlus,
            run: () => bulkAssign('user', agent.value)
          }))
        ]
      },
      {
        id: 'bulk.assign-team',
        label: t('actions.assignTeam'),
        keywords: [t('globals.terms.team')],
        section,
        icon: Users,
        permission: perms.CONVERSATIONS_UPDATE_TEAM_ASSIGNEE,
        group: true,
        placeholder: t('command.searchTeams'),
        search: async (term) => [
          {
            id: 'bulk.assign-team.none',
            label: t('globals.terms.none'),
            parent: 'bulk.assign-team',
            icon: X,
            run: () => bulkAssign('team', 'none')
          },
          ...(await teamStore.searchTeams(term)).map((team) => ({
            id: `bulk.assign-team.${team.value}`,
            label: team.label,
            parent: 'bulk.assign-team',
            icon: UsersRound,
            run: () => bulkAssign('team', team.value)
          }))
        ]
      },
      {
        id: 'bulk.add-tag',
        label: t('actions.addTags'),
        keywords: [t('globals.terms.tag')],
        section,
        icon: Tag,
        permission: perms.CONVERSATIONS_UPDATE_TAGS,
        group: true,
        placeholder: t('command.searchTags'),
        search: async (term) =>
          (await tagStore.searchTagNames(term)).map((tag) => ({
            id: `bulk.add-tag.${tag.value}`,
            label: tag.label,
            parent: 'bulk.add-tag',
            icon: TagIcon,
            run: () => bulkAddTag(tag.value)
          }))
      },
      {
        id: 'bulk.clear',
        label: t('conversation.bulkActions.clearSelection'),
        keywords: ['deselect'],
        section,
        icon: X,
        run: () => conversationStore.clearSelection()
      }
    ]
  })
}
