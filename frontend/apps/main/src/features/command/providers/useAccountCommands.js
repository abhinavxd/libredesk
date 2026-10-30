const awayMessage = msg({ message: "Away", context: "Conversation navigation" })
import { defineMessage as msg } from '@lingui/core/macro'


import { computed } from 'vue'
import { useI18n } from '@shared-ui/i18n'
import { useColorMode } from '@vueuse/core'
import {
  CircleCheck,
  Clock,
  Keyboard,
  LogOut,
  Moon,
  Sun,
  UserRoundCog
} from 'lucide-vue-next'
import { useUserStore } from '@main/stores/user'
import { useKeyboardShortcutsDialog } from '@main/composables/useKeyboardShortcutsDialog'
import { useLogout } from '@main/composables/useLogout'
import { SECTIONS } from '@main/features/command/sections'

const AWAY_STATUSES = ['away_manual', 'away_and_reassigning']

export function useAccountCommands() {
  const i18n = useI18n()
  const mode = useColorMode()
  const userStore = useUserStore()
  const shortcutsDialog = useKeyboardShortcutsDialog()
  const logout = useLogout()

  return computed(() => {
    const status = userStore.user.availability_status
    const isAway = AWAY_STATUSES.includes(status)
    const isReassigning = status === 'away_and_reassigning'
    const isDark = mode.value === 'dark'

    return [
      {
        id: 'account.theme',
        label: isDark ? i18n._('Switch to light mode') : i18n._('Switch to dark mode'),
        keywords: [i18n._('Dark Mode'), 'theme'],
        section: SECTIONS.ACCOUNT,
        icon: isDark ? Sun : Moon,
        run: () => {
          mode.value = isDark ? 'light' : 'dark'
        }
      },
      {
        id: 'account.availability',
        label: isAway ? i18n._('Set status to online') : i18n._('Set status to away'),
        keywords: [i18n._(awayMessage), i18n._('{count, plural, one {Online} other {Online}}'), i18n._('{count, plural, one {Status} other {Statuses}}')],
        section: SECTIONS.ACCOUNT,
        icon: isAway ? CircleCheck : Clock,
        run: () => userStore.updateUserAvailability(isAway ? 'online' : 'away_manual')
      },
      {
        id: 'account.reassign',
        label: isReassigning ? i18n._('Disable reassign replies') : i18n._('Enable reassign replies'),
        keywords: [i18n._('Reassign replies'), i18n._(awayMessage)],
        section: SECTIONS.ACCOUNT,
        icon: UserRoundCog,
        run: () =>
          userStore.updateUserAvailability(isReassigning ? 'away_manual' : 'away_and_reassigning')
      },
      {
        id: 'account.shortcuts',
        label: i18n._('Keyboard shortcuts'),
        section: SECTIONS.ACCOUNT,
        icon: Keyboard,
        shortcut: ['Ctrl', '/'],
        run: shortcutsDialog.show
      },
      {
        id: 'account.logout',
        label: i18n._('Logout'),
        keywords: ['sign out'],
        section: SECTIONS.ACCOUNT,
        icon: LogOut,
        run: logout
      }
    ]
  })
}
