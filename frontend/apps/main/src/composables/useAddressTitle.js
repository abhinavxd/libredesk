import { watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAppSettingsStore } from '@main/stores/appSettings'
import { useAddressStore } from '@main/stores/address'
import { useConversationStore } from '@main/stores/conversation'

export function useAddressTitle() {
  const route = useRoute()
  const { t, locale } = useI18n()
  const settings = useAppSettingsStore()
  const conversations = useConversationStore()
  const addresses = useAddressStore()

  const updateTitle = () => {
    const address = addresses.get(route.params?.addressID)
    const titleKey = route.meta?.titleKey
    const page = address?.address || (titleKey ? t(titleKey, route.meta?.titleCount || 1) : 'Addresses')
    const site = settings.siteTitle
    const count = conversations.sidebarCounts.unread
    document.title = `${count > 0 ? `(${count}) ` : ''}${page} - ${site}`
  }

  watch(
    [
      () => route.fullPath,
      () => addresses.addresses,
      () => route.meta,
      () => settings.siteTitle,
      () => conversations.sidebarCounts.unread,
      locale
    ],
    updateTitle,
    { immediate: true }
  )
}
