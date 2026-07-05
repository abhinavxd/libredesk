import { ref, watch } from 'vue'
import { useDebounceFn, useEventListener } from '@vueuse/core'
import { useConversationStore } from '@main/stores/conversation'
import { MACRO_CONTEXT } from '@main/constants/conversation'
import { getTextFromHTML } from '@shared-ui/utils/string.js'
import api from '@main/api'

const validateMacroActions = (actions) => {
  if (!Array.isArray(actions)) return []
  return actions.filter(action =>
    action &&
    'type' in action &&
    'value' in action &&
    Array.isArray(action.value) &&
    'display_value' in action &&
    Array.isArray(action.display_value)
  )
}

const validateAttachments = (attachments) => {
  if (!Array.isArray(attachments)) return []
  return attachments.filter(attachment =>
    attachment &&
    'id' in attachment &&
    'size' in attachment &&
    'uuid' in attachment &&
    'filename' in attachment &&
    'content_type' in attachment
  )
}

const isDraftEmpty = (draft) => {
  const content = draft?.content || ''
  const hasText = getTextFromHTML(content).length > 0
  const hasInlineImage = /<img\b/i.test(content)
  const hasAttachments = draft?.meta?.attachments?.length > 0
  const hasMacroActions = draft?.meta?.macro_actions?.length > 0
  return !hasText && !hasInlineImage && !hasAttachments && !hasMacroActions
}

const draftKey = (uuid, type) => `${uuid}::${type}`

// Compare only the meaningful bits so equal drafts don't churn the backend on every load.
const metaSignature = (meta) =>
  JSON.stringify({
    macro_actions: meta?.macro_actions || [],
    attachments: (meta?.attachments || []).map(a => a.uuid)
  })

const sameDraft = (a, b) => {
  if (isDraftEmpty(a) && isDraftEmpty(b)) return true
  return (a?.content || '') === (b?.content || '') && metaSignature(a?.meta) === metaSignature(b?.meta)
}

// Per-conversation, per-type draft state; the server-backed store is the single source of truth and edits debounce back to it.
export function useDraftManager (conversationUUID, messageType, uploadedFiles = null) {
  const conversationStore = useConversationStore()
  const htmlContent = ref('')
  const textContent = ref('')
  const isLoading = ref(false)
  const loadedAttachments = ref([])
  const loadedMacroActions = ref([])

  // Saves fire only while this equals the live key, keeping the transient empty editor during open/switch from clobbering a real draft.
  const loadedKey = ref(null)
  const currentKey = () => draftKey(conversationUUID.value, messageType.value)

  const buildDraft = () => {
    const meta = {}
    const macroActions = conversationStore.getMacro(MACRO_CONTEXT.REPLY)?.actions || []
    if (macroActions.length > 0) meta.macro_actions = macroActions
    if (uploadedFiles?.value?.length > 0) {
      meta.attachments = uploadedFiles.value.map(file => ({
        id: file.id,
        size: file.size,
        uuid: file.uuid,
        filename: file.filename,
        content_type: file.content_type
      }))
    }
    return { content: htmlContent.value, meta }
  }

  const applyDraft = (draft) => {
    htmlContent.value = draft?.content || ''
    textContent.value = ''
    loadedAttachments.value = validateAttachments(draft?.meta?.attachments)
    loadedMacroActions.value = validateMacroActions(draft?.meta?.macro_actions)
  }

  const load = async (uuid, type) => {
    isLoading.value = true
    loadedKey.value = null
    // Prefetch may still be in flight on a fresh page load; wait rather than read an empty store.
    await conversationStore.draftsReady
    applyDraft(conversationStore.getDraft(uuid, type))
    loadedKey.value = draftKey(uuid, type)
    isLoading.value = false
  }

  const save = async (uuid, type) => {
    const draft = buildDraft()
    if (sameDraft(draft, conversationStore.getDraft(uuid, type))) return
    try {
      if (isDraftEmpty(draft)) {
        await api.deleteDraft(uuid, type)
        conversationStore.removeDraft(uuid, type)
      } else {
        await api.saveDraft(uuid, type, draft)
        conversationStore.setDraft(uuid, type, draft)
      }
    } catch (error) {
      // Keep the editor content; the next save retries.
    }
  }

  const debouncedSave = useDebounceFn(() => {
    if (loadedKey.value === currentKey()) save(conversationUUID.value, messageType.value)
  }, 500)

  const clearDraft = async (uuid = conversationUUID.value, type = messageType.value) => {
    if (!uuid) return
    try {
      await api.deleteDraft(uuid, type)
      conversationStore.removeDraft(uuid, type)
    } catch (error) {
      // Silent fail
    }
    if (uuid === conversationUUID.value && type === messageType.value) applyDraft(null)
  }

  const watchSources = [
    htmlContent,
    textContent,
    () => conversationStore.macros[MACRO_CONTEXT.REPLY]
  ]
  if (uploadedFiles) watchSources.push(uploadedFiles)

  watch(
    watchSources,
    () => {
      if (!isLoading.value && loadedKey.value === currentKey()) debouncedSave()
    },
    { deep: true }
  )

  // Flush the outgoing draft, then load the incoming one. Serialized so rapid switches can't interleave.
  let chain = Promise.resolve()
  watch(
    [conversationUUID, messageType],
    ([uuid, type], oldVals) => {
      const [prevUuid, prevType] = oldVals || []
      chain = chain.then(async () => {
        if (prevUuid && loadedKey.value === draftKey(prevUuid, prevType)) await save(prevUuid, prevType)
        if (uuid) await load(uuid, type)
        else applyDraft(null)
      }).catch(() => {})
    },
    { immediate: true }
  )

  useEventListener(document, 'visibilitychange', () => {
    if (document.visibilityState === 'hidden' && conversationUUID.value && loadedKey.value === currentKey()) {
      save(conversationUUID.value, messageType.value)
    }
  })

  return {
    htmlContent,
    textContent,
    isLoading,
    clearDraft,
    loadedAttachments,
    loadedMacroActions
  }
}
