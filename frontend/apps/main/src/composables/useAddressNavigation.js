import { useRouter } from 'vue-router'
import { useIsMobile } from '@shared-ui/composables'
import { useConversationStore } from '@main/stores/conversation'

export function useAddressNavigation() {
  const router = useRouter()
  const isMobile = useIsMobile()
  const conversationStore = useConversationStore()

  const navigateToAddress = (addressID) => {
    const uuid =
      !isMobile.value &&
      conversationStore.isConversationOpen &&
      Number(conversationStore.conversation.data?.address_id) === Number(addressID)
        ? conversationStore.conversation.data?.uuid
        : null
    if (uuid) {
      return router.push({ name: 'address-inbox-conversation', params: { addressID, uuid } })
    }
    return router.push({ name: 'address-inbox', params: { addressID } })
  }

  return { navigateToAddress }
}
