import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Mail, Search } from 'lucide-vue-next'
import { navIconMap } from '@main/constants/navIcons'
import { adminNavItems } from '@main/constants/navigation'
import { useAddressStore } from '@main/stores/address'
import { useAddressNavigation } from '@main/composables/useAddressNavigation'
import { SECTIONS } from '../sections'

export function useNavigationCommands() {
  const router = useRouter()
  const { t } = useI18n()
  const addressStore = useAddressStore()
  const { navigateToAddress } = useAddressNavigation()

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
        run: () => router.push(item.href)
      }))
    )

  return computed(() => [
    ...addressStore.addresses.map((address) => ({
      id: `goto.address.${address.id}`,
      label: address.address,
      hint: address.display_name || 'Address',
      keywords: [address.address, address.display_name].filter(Boolean),
      section: SECTIONS.GOTO,
      icon: Mail,
      run: () => navigateToAddress(address.id)
    })),
    {
      id: 'goto.search',
      label: t('conversation.search'),
      section: SECTIONS.GOTO,
      icon: Search,
      run: () => router.push({ name: 'search' })
    },
    ...adminCommands()
  ])
}
