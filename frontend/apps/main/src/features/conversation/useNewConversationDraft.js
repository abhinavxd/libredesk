import { useStorage, StorageSerializers } from '@vueuse/core'

const DRAFT_KEYS = {
  email: 'newConversationDraftEmail',
  whatsapp: 'newConversationDraftWhatsApp'
}

// Sync flush: a queued write is dropped when the form unmounts right after clearing the draft.
export const useNewConversationDraft = (channel) =>
  useStorage(DRAFT_KEYS[channel], null, undefined, { serializer: StorageSerializers.object, flush: 'sync' })

export const hasNewConversationDraft = (storage) =>
  Object.values(DRAFT_KEYS).some((key) => storage?.getItem(key))

export const clearNewConversationDrafts = (storage) => {
  Object.values(DRAFT_KEYS).forEach((key) => storage?.removeItem(key))
}
