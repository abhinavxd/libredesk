import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { SquarePen } from 'lucide-vue-next'
import { navIconMap } from '@main/constants/navIcons'
import { adminNavItems } from '@main/constants/navigation'
import { permissions as perms } from '@main/constants/permissions'
import { useComposeStore } from '@main/stores/compose'
import { SECTIONS } from '../sections'

// Compose New plus the administrative creation flows. There are no saved views.
export function useCreateCommands() {
  const router = useRouter()
  const { t } = useI18n()
  const composeStore = useComposeStore()
  const compose = () => ({
    id: 'create.compose',
    label: t('compose.button'),
    keywords: [t('globals.messages.create'), t('compose.title')],
    section: SECTIONS.CREATE,
    icon: SquarePen,
    permission: perms.CONVERSATIONS_CREATE,
    run: () => composeStore.open()
  })
  const adminCreates = () =>
    adminNavItems
      .flatMap((group) => group.children)
      .filter((item) => item.createRouteName)
      .map((item) => ({
        id: `create.${item.createRouteName}`,
        label: t(router.resolve({ name: item.createRouteName }).meta.titleKey),
        keywords: [t('globals.messages.create'), t('globals.terms.admin')],
        section: SECTIONS.CREATE,
        icon: navIconMap[item.icon],
        permission: item.permission,
        run: () => router.push({ name: item.createRouteName })
      }))
  return computed(() => [compose(), ...adminCreates()])
}
