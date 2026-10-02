import { computed, nextTick, onScopeDispose, ref, watch } from 'vue'
import { useDebounceFn, useEventListener } from '@vueuse/core'
import { useConversationStore } from '@main/stores/conversation'
import { getTextFromHTML } from '@shared-ui/utils/string.js'

const keyFor = (uuid, type) => `${uuid}::${type}`
const emptyRecipients = () => ({ to: '', cc: '', bcc: '' })
const copy = (value) => JSON.parse(JSON.stringify(value))
const isEmpty = (draft) =>
  !getTextFromHTML(draft?.content || '').length &&
  !/<img\b/i.test(draft?.content || '') &&
  !draft?.meta?.attachments?.length &&
  !draft?.meta?.recipients
const sameDraft = (left, right) =>
  (isEmpty(left) && isEmpty(right)) ||
  JSON.stringify({ content: left?.content || '', meta: left?.meta || {} }) ===
    JSON.stringify({ content: right?.content || '', meta: right?.meta || {} })

export function useDraftManager(
  conversationUUID,
  messageType,
  uploadedFiles = null,
  recipientDefaults = null
) {
  const store = useConversationStore()
  const htmlContent = ref('')
  const textContent = ref('')
  const isLoading = ref(false)
  const loadedAttachments = ref([])
  const recipients = ref(emptyRecipients())
  const loadedKey = ref(null)
  let recipientsInitialized = false
  let recipientsEdited = false
  let applying = false
  let loadSequence = 0
  const currentKey = () => keyFor(conversationUUID.value, messageType.value)

  function applyRecipients(value) {
    applying = true
    recipients.value = { ...emptyRecipients(), ...value }
    applying = false
  }

  function prefillRecipients() {
    const defaults = recipientDefaults?.value
    if (
      recipientsInitialized ||
      recipientsEdited ||
      !defaults?.ready ||
      defaults.uuid !== conversationUUID.value ||
      loadedKey.value !== currentKey()
    )
      return
    applyRecipients(defaults.recipients)
    recipientsInitialized = true
  }

  function buildDraft() {
    const meta = {}
    if (uploadedFiles?.value?.length) {
      meta.attachments = uploadedFiles.value.map((file) => ({
        id: file.id,
        url: file.url,
        size: file.size,
        uuid: file.uuid,
        filename: file.filename || file.name,
        content_type: file.content_type,
        disposition: file.disposition
      }))
    }
    // Recipient-only edits are a draft too; untouched defaults do not create one.
    if (
      messageType.value === 'reply' &&
      (recipientsEdited || htmlContent.value || meta.attachments)
    ) {
      meta.recipients = { ...recipients.value }
    }
    return { content: htmlContent.value, meta }
  }

  function applyDraft(draft) {
    htmlContent.value = draft?.content || ''
    textContent.value = getTextFromHTML(htmlContent.value)
    loadedAttachments.value = (draft?.meta?.attachments || []).filter((a) => a.id && a.uuid)
    recipientsEdited = Boolean(draft?.meta?.recipients)
    recipientsInitialized = recipientsEdited
    applyRecipients(draft?.meta?.recipients || emptyRecipients())
  }

  function persist(uuid, type, draft) {
    if (sameDraft(draft, store.getDraft(uuid, type))) return
    if (isEmpty(draft)) {
      store.removeDraft(uuid, type)
      store.syncDraft(uuid, type, null)
    } else {
      store.setDraft(uuid, type, draft)
      store.syncDraft(uuid, type, draft)
    }
  }

  function save(uuid = conversationUUID.value, type = messageType.value) {
    if (uuid) persist(uuid, type, buildDraft())
  }

  const debouncedSave = useDebounceFn(() => {
    if (loadedKey.value === currentKey()) save()
  }, 500)

  watch(
    recipients,
    () => {
      if (applying || isLoading.value) return
      recipientsEdited = true
      recipientsInitialized = true
      debouncedSave()
    },
    { deep: true, flush: 'sync' }
  )

  watch(
    [htmlContent, textContent, ...(uploadedFiles ? [uploadedFiles] : [])],
    () => {
      if (!isLoading.value && loadedKey.value === currentKey()) debouncedSave()
    },
    { deep: true }
  )

  watch(
    [conversationUUID, messageType],
    async ([uuid, type], old = []) => {
      const [prevUuid, prevType] = old
      if (prevUuid && loadedKey.value === keyFor(prevUuid, prevType)) {
        // Use the old type when capturing a switch from reply to private note.
        const draft = buildDraft()
        if (prevType === 'reply' && (recipientsEdited || draft.content || draft.meta.attachments)) {
          draft.meta.recipients = { ...recipients.value }
        }
        if (prevType !== 'reply') delete draft.meta.recipients
        persist(prevUuid, prevType, draft)
      }
      const sequence = ++loadSequence
      isLoading.value = true
      loadedKey.value = null
      await store.draftsReady
      if (sequence !== loadSequence) return
      applyDraft(uuid ? store.getDraft(uuid, type) : null)
      // Allow the attachment owner to restore the loaded array before autosaving.
      await nextTick()
      if (sequence !== loadSequence) return
      loadedKey.value = uuid ? keyFor(uuid, type) : null
      isLoading.value = false
      prefillRecipients()
    },
    { immediate: true }
  )

  if (recipientDefaults) watch(recipientDefaults, prefillRecipients, { deep: true })

  function clearDraft(uuid = conversationUUID.value, type = messageType.value) {
    if (!uuid) return
    store.removeDraft(uuid, type)
    store.syncDraft(uuid, type, null)
    if (loadedKey.value === keyFor(uuid, type)) {
      applyDraft(null)
      prefillRecipients()
    }
  }

  // Re-read the open draft from the store, e.g. after a review returns a reply to it.
  function reloadDraft() {
    const uuid = conversationUUID.value
    if (!uuid || loadedKey.value !== currentKey()) return
    applyDraft(store.getDraft(uuid, messageType.value))
    prefillRecipients()
  }

  function captureDraft() {
    const snapshot = {
      uuid: conversationUUID.value,
      type: messageType.value,
      draft: copy(buildDraft())
    }
    persist(snapshot.uuid, snapshot.type, snapshot.draft)
    return snapshot
  }

  function completeSend(snapshot) {
    const isCurrent = loadedKey.value === keyFor(snapshot.uuid, snapshot.type)
    const latest = isCurrent ? buildDraft() : store.getDraft(snapshot.uuid, snapshot.type)
    if (!sameDraft(latest, snapshot.draft)) return false
    clearDraft(snapshot.uuid, snapshot.type)
    return isCurrent
  }

  const flush = () => {
    if (loadedKey.value === currentKey()) save()
  }
  useEventListener(document, 'visibilitychange', () => {
    if (document.visibilityState === 'hidden') flush()
  })
  onScopeDispose(flush)

  const saveState = computed(() => store.draftSaveStates.get(currentKey()) || '')
  const retrySave = () => store.retryDraftSave(conversationUUID.value, messageType.value)
  return {
    htmlContent,
    textContent,
    isLoading,
    loadedAttachments,
    recipients,
    clearDraft,
    reloadDraft,
    captureDraft,
    completeSend,
    saveState,
    retrySave
  }
}
