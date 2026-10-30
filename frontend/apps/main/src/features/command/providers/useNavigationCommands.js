
import { defineMessage as msg } from '@lingui/core/macro'


import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from '@shared-ui/i18n'
import {
  AtSign,
  BookUser,
  CircleUser,
  Eye,
  Inbox,
  Layers,
  Search,
  UserX,
  Users
} from 'lucide-vue-next'
import { navIconMap } from '@main/constants/navIcons'
import { adminNavItems, reportsNavItems } from '@main/constants/navigation'
import { permissions } from '@main/constants/permissions'
import { useUserStore } from '@main/stores/user'
import { useViewStore } from '@main/stores/view'
import { useSharedViewStore } from '@main/stores/sharedView'
import { useInboxNavigation } from '@main/composables/useInboxNavigation'
import { SECTIONS } from '../sections'

const INBOX_TYPES = [
  { type: 'assigned', labelMessage: msg({ message: 'My inbox' }), icon: Inbox },
  { type: 'mentioned', labelMessage: msg({ message: 'Mentions' }), icon: AtSign },
  { type: 'unassigned', labelMessage: msg({ message: 'Unassigned' }), icon: UserX },
  { type: 'all', labelMessage: msg({ message: 'All' }), icon: Layers }
]

export function useNavigationCommands() {
  const router = useRouter()
  const i18n = useI18n()
  const userStore = useUserStore()
  const viewStore = useViewStore()
  const sharedViewStore = useSharedViewStore()
  const { navigateToInbox, navigateToTeamInbox, navigateToViewInbox } = useInboxNavigation()

  const goTo = (route) => () => router.push(route)

  const navItemLabel = (item) => i18n._(item.titleMessage, item.isTitlePlural ? 2 : 1)

  const adminCommands = () =>
    adminNavItems.flatMap((group) =>
      group.children.map((item) => ({
        id: `goto.admin.${item.href}`,
        label: navItemLabel(item),
        hint: navItemLabel(group),
        keywords: [navItemLabel(group), i18n._('{count, plural, one {Admin} other {Admins}}')],
        section: SECTIONS.GOTO,
        icon: navIconMap[item.icon],
        permission: item.permission,
        run: goTo(item.href)
      }))
    )

  const reportCommands = () =>
    reportsNavItems.map((item) => ({
      id: `goto.reports.${item.href}`,
      label: navItemLabel(item),
      hint: i18n._('{count, plural, one {Report} other {Reports}}', 2),
      keywords: [i18n._('{count, plural, one {Report} other {Reports}}', 2)],
      section: SECTIONS.GOTO,
      icon: navIconMap[item.icon],
      permission: item.permission,
      run: goTo(item.href)
    }))

  return computed(() => [
    ...INBOX_TYPES.map(({ type, labelMessage, icon }) => ({
      id: `goto.inbox.${type}`,
      label: i18n._(labelMessage),
      hint: i18n._('{count, plural, one {Inbox} other {Inboxes}}', 2),
      keywords: [i18n._('{count, plural, one {Inbox} other {Inboxes}}', 2)],
      section: SECTIONS.GOTO,
      icon,
      run: () => navigateToInbox(type)
    })),
    ...userStore.teams.map((team) => ({
      id: `goto.team.${team.id}`,
      label: team.name,
      hint: i18n._('{count, plural, one {Team Inbox} other {Team Inboxes}}'),
      keywords: [i18n._('{count, plural, one {Team Inbox} other {Team Inboxes}}'), i18n._('{count, plural, one {Team} other {Teams}}')],
      section: SECTIONS.GOTO,
      icon: Users,
      run: () => navigateToTeamInbox(team.id)
    })),
    ...viewStore.views.map((view) => ({
      id: `goto.view.${view.id}`,
      label: view.name,
      hint: i18n._('{count, plural, one {View} other {Views}}'),
      keywords: [i18n._('{count, plural, one {View} other {Views}}')],
      section: SECTIONS.GOTO,
      icon: Eye,
      permission: permissions.VIEW_MANAGE,
      run: () => navigateToViewInbox(view.id)
    })),
    ...sharedViewStore.sharedViewList.map((view) => ({
      id: `goto.shared-view.${view.id}`,
      label: view.name,
      hint: i18n._('{count, plural, one {Shared view} other {Shared views}}'),
      keywords: [i18n._('{count, plural, one {Shared view} other {Shared views}}'), i18n._('{count, plural, one {View} other {Views}}')],
      section: SECTIONS.GOTO,
      icon: Eye,
      run: () => navigateToViewInbox(view.id)
    })),
    {
      id: 'goto.search',
      label: i18n._('Search conversations'),
      section: SECTIONS.GOTO,
      icon: Search,
      run: goTo({ name: 'search' })
    },
    {
      id: 'goto.contacts',
      label: i18n._('All contacts'),
      keywords: [i18n._('{count, plural, one {Contact} other {Contacts}}', 2)],
      section: SECTIONS.GOTO,
      icon: BookUser,
      permission: permissions.CONTACTS_READ_ALL,
      run: goTo({ name: 'contacts' })
    },
    ...reportCommands(),
    ...adminCommands(),
    {
      id: 'goto.profile',
      label: i18n._('{count, plural, one {Profile} other {Profiles}}'),
      hint: i18n._('{count, plural, one {Account} other {Accounts}}'),
      keywords: [i18n._('{count, plural, one {Account} other {Accounts}}')],
      section: SECTIONS.GOTO,
      icon: CircleUser,
      run: goTo({ name: 'profile' })
    }
  ])
}
