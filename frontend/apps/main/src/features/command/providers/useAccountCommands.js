import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
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
  { value: 'auto', label: 'command.switchToSystemTheme', icon: Monitor },
  { value: 'light', label: 'command.switchToLightMode', icon: Sun },
  { value: 'dark', label: 'command.switchToDarkMode', icon: Moon }
]

export function useAccountCommands() {
  const { t } = useI18n()
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
      label: t(option.label),
      keywords: [t('globals.terms.theme'), 'theme'],
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
        label: t(isAway ? 'command.setOnline' : 'command.setAway'),
        keywords: [t('navigation.away'), t('globals.terms.online'), t('globals.terms.status')],
        section: SECTIONS.ACCOUNT,
        icon: isAway ? CircleCheck : Clock,
        run: () => userStore.updateUserAvailability(isAway ? 'online' : 'away_manual')
      },
      {
        id: 'account.reassign',
        label: t(isReassigning ? 'command.disableReassignReplies' : 'command.enableReassignReplies'),
        keywords: [t('navigation.reassignReplies'), t('navigation.away')],
        section: SECTIONS.ACCOUNT,
        icon: UserRoundCog,
        run: () =>
          userStore.updateUserAvailability(isReassigning ? 'away_manual' : 'away_and_reassigning')
      },
      {
        id: 'account.shortcuts',
        label: t('navigation.keyboardShortcuts'),
        section: SECTIONS.ACCOUNT,
        icon: Keyboard,
        shortcut: ['Ctrl', '/'],
        run: shortcutsDialog.show
      },
      {
        id: 'account.logout',
        label: t('navigation.logout'),
        keywords: ['sign out'],
        section: SECTIONS.ACCOUNT,
        icon: LogOut,
        run: logout
      }
    ]
  })
}
