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
  Monitor,
  Moon,
  Sun,
  UserRoundCog
} from 'lucide-vue-next'
import { useUserStore } from '@main/stores/user'
import { useKeyboardShortcutsDialog } from '@main/composables/useKeyboardShortcutsDialog'
import { useLogout } from '@main/composables/useLogout'
import { SECTIONS } from '@main/features/command/sections'

const AWAY_STATUSES = ['away_manual', 'away_and_reassigning']

const THEME_COMMANDS = [
  { value: 'auto', labelMessage: msg({ message: 'Switch to system theme' }), icon: Monitor },
  { value: 'light', labelMessage: msg({ message: 'Switch to light mode' }), icon: Sun },
  { value: 'dark', labelMessage: msg({ message: 'Switch to dark mode' }), icon: Moon }
]

export function useAccountCommands() {
  const i18n = useI18n()
  const { store: themePreference } = useColorMode()
  const userStore = useUserStore()
  const shortcutsDialog = useKeyboardShortcutsDialog()
  const logout = useLogout()

  return computed(() => {
    const status = userStore.user.availability_status
    const isAway = AWAY_STATUSES.includes(status)
    const isReassigning = status === 'away_and_reassigning'

    const themeCommands = THEME_COMMANDS.filter(
      (option) => option.value !== themePreference.value
    ).map((option) => ({
      id: `account.theme.${option.value}`,
      label: i18n._(option.labelMessage),
      keywords: [i18n._('Theme'), 'theme'],
      section: SECTIONS.ACCOUNT,
      icon: option.icon,
      run: () => {
        themePreference.value = option.value
      }
    }))

    return [
      ...themeCommands,
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
