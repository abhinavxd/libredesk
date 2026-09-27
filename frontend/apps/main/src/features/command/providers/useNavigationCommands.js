import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
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
  { type: 'assigned', labelKey: 'conversation.myInbox', icon: Inbox },
  { type: 'mentioned', labelKey: 'conversation.mentions', icon: AtSign },
  { type: 'unassigned', labelKey: 'globals.terms.unassigned', icon: UserX },
  { type: 'all', labelKey: 'globals.messages.all', icon: Layers }
]

export function useNavigationCommands() {
  const router = useRouter()
  const { t } = useI18n()
  const userStore = useUserStore()
  const viewStore = useViewStore()
  const sharedViewStore = useSharedViewStore()
  const { navigateToInbox, navigateToTeamInbox, navigateToViewInbox } = useInboxNavigation()

  const goTo = (route) => () => router.push(route)

  const navItemLabel = (item) => t(item.titleKey, item.isTitleKeyPlural ? 2 : 1)

  const adminCommands = () =>
    adminNavItems.flatMap((group) =>
      group.children.map((item) => ({
        id: `goto.admin.${item.href}`,
        label: navItemLabel(item),
        hint: navItemLabel(group),
        keywords: [navItemLabel(group), t('globals.terms.admin')],
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
      hint: t('globals.terms.report', 2),
      keywords: [t('globals.terms.report', 2)],
      section: SECTIONS.GOTO,
      icon: navIconMap[item.icon],
      permission: item.permission,
      run: goTo(item.href)
    }))

  return computed(() => [
    ...INBOX_TYPES.map(({ type, labelKey, icon }) => ({
      id: `goto.inbox.${type}`,
      label: t(labelKey),
      hint: t('globals.terms.inbox', 2),
      keywords: [t('globals.terms.inbox', 2)],
      section: SECTIONS.GOTO,
      icon,
      run: () => navigateToInbox(type)
    })),
    ...userStore.teams.map((team) => ({
      id: `goto.team.${team.id}`,
      label: team.name,
      hint: t('globals.terms.teamInbox'),
      keywords: [t('globals.terms.teamInbox'), t('globals.terms.team')],
      section: SECTIONS.GOTO,
      icon: Users,
      run: () => navigateToTeamInbox(team.id)
    })),
    ...viewStore.views.map((view) => ({
      id: `goto.view.${view.id}`,
      label: view.name,
      hint: t('globals.terms.view'),
      keywords: [t('globals.terms.view')],
      section: SECTIONS.GOTO,
      icon: Eye,
      permission: permissions.VIEW_MANAGE,
      run: () => navigateToViewInbox(view.id)
    })),
    ...sharedViewStore.sharedViewList.map((view) => ({
      id: `goto.shared-view.${view.id}`,
      label: view.name,
      hint: t('globals.terms.sharedView'),
      keywords: [t('globals.terms.sharedView'), t('globals.terms.view')],
      section: SECTIONS.GOTO,
      icon: Eye,
      run: () => navigateToViewInbox(view.id)
    })),
    {
      id: 'goto.search',
      label: t('conversation.search'),
      section: SECTIONS.GOTO,
      icon: Search,
      run: goTo({ name: 'search' })
    },
    {
      id: 'goto.contacts',
      label: t('contact.allContacts'),
      keywords: [t('globals.terms.contact', 2)],
      section: SECTIONS.GOTO,
      icon: BookUser,
      permission: permissions.CONTACTS_READ_ALL,
      run: goTo({ name: 'contacts' })
    },
    ...reportCommands(),
    ...adminCommands(),
    {
      id: 'goto.profile',
      label: t('globals.terms.profile'),
      hint: t('globals.terms.account'),
      keywords: [t('globals.terms.account')],
      section: SECTIONS.GOTO,
      icon: CircleUser,
      run: goTo({ name: 'profile' })
    }
  ])
}
