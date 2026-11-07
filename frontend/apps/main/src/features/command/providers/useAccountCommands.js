import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useColorMode } from '@vueuse/core'
import {
  CircleCheck,
  Clock,
  Keyboard,
  LogOut,
  Moon,
  Monitor,
  Sun,
  UserRoundCog
} from 'lucide-vue-next'
import { useUserStore } from '@main/stores/user'
import { useKeyboardShortcutsDialog } from '@main/composables/useKeyboardShortcutsDialog'
import { useLogout } from '@main/composables/useLogout'
import { SECTIONS } from '@main/features/command/sections'

const AWAY_STATUSES = ['away_manual', 'away_and_reassigning']

export function useAccountCommands() {
  const { t } = useI18n()
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
        label: t(isDark ? 'command.switchToLightMode' : 'command.switchToDarkMode'),
        keywords: [t('navigation.darkMode'), 'theme'],
        section: SECTIONS.ACCOUNT,
        icon: isDark ? Sun : Moon,
        run: () => {
          mode.value = isDark ? 'light' : 'dark'
        }
      },
      {
        id: 'account.themeSystem',
        label: t('command.useSystemTheme'),
        keywords: [t('navigation.darkMode'), 'theme', 'system', 'auto'],
        section: SECTIONS.ACCOUNT,
        icon: Monitor,
        run: () => {
          mode.value = 'auto'
        }
      },
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
