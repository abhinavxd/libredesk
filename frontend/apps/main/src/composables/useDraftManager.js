import { ref, watch } from 'vue'
import { useDebounceFn, useEventListener } from '@vueuse/core'
import { useConversationStore } from '@main/stores/conversation'
import { MACRO_CONTEXT } from '@main/constants/conversation'
import { getTextFromHTML } from '@shared-ui/utils/string.js'
import api from '@main/api'

const hasKeys = (obj, keys) => Boolean(obj) && keys.every(key => key in obj)

const validateMacroActions = (actions) => {
  if (!Array.isArray(actions)) return []
  return actions.filter(action =>
    hasKeys(action, ['type', 'value', 'display_value']) &&
    Array.isArray(action.value) &&
    Array.isArray(action.display_value)
  )
}

const validateAttachments = (attachments) => {
  if (!Array.isArray(attachments)) return []
  return attachments.filter(attachment =>
    hasKeys(attachment, ['id', 'size', 'uuid', 'filename', 'content_type'])
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

const metaSignature = (meta) =>
  JSON.stringify({
    macro_actions: meta?.macro_actions || [],
    attachments: (meta?.attachments || []).map(a => a.uuid)
  })

const sameDraft = (a, b) => {
  if (isDraftEmpty(a) && isDraftEmpty(b)) return true
  return (a?.content || '') === (b?.content || '') && metaSignature(a?.meta) === metaSignature(b?.meta)
}

export function useDraftManager (conversationUUID, messageType, uploadedFiles = null) {
  const conversationStore = useConversationStore()
  const htmlContent = ref('')
  const textContent = ref('')
  const isLoading = ref(false)
  const loadedAttachments = ref([])
  const loadedMacroActions = ref([])

  // Live-key guard: the editor is transiently empty during open/switch and must not clobber a stored draft.
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
    // Prefetch may still be in flight on a fresh page load; reading the store now would apply an empty draft.
    const before = htmlContent.value
    await conversationStore.draftsReady
    // If the user typed while drafts were still loading, keep their input instead of clobbering it.
    if (htmlContent.value === before) {
      applyDraft(conversationStore.getDraft(uuid, type))
    }
    loadedKey.value = draftKey(uuid, type)
    isLoading.value = false
  }

  // Serialize every server write so a delete can never land before an earlier save and resurrect the row.
  let syncChain = Promise.resolve()
  const queueSync = (fn) => {
    syncChain = syncChain.then(fn, fn)
  }

  // Update the store synchronously; server sync runs in the background, never awaited.
  const save = (uuid, type) => {
    const draft = buildDraft()
    if (sameDraft(draft, conversationStore.getDraft(uuid, type))) return
    if (isDraftEmpty(draft)) {
      conversationStore.removeDraft(uuid, type)
      queueSync(() => api.deleteDraft(uuid, type).catch(() => {}))
    } else {
      conversationStore.setDraft(uuid, type, draft)
      queueSync(() => api.saveDraft(uuid, type, draft).catch(() => {}))
    }
  }

  const debouncedSave = useDebounceFn(() => {
    if (loadedKey.value === currentKey()) save(conversationUUID.value, messageType.value)
  }, 500)

  const clearDraft = (uuid = conversationUUID.value, type = messageType.value) => {
    if (!uuid) return
    conversationStore.removeDraft(uuid, type)
    queueSync(() => api.deleteDraft(uuid, type).catch(() => {}))
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

  // Serialize switches: a rapid A->B->A must not interleave save and load.
  let chain = Promise.resolve()
  watch(
    [conversationUUID, messageType],
    ([uuid, type], oldVals) => {
      const [prevUuid, prevType] = oldVals || []
      chain = chain.then(async () => {
        if (prevUuid && loadedKey.value === draftKey(prevUuid, prevType)) save(prevUuid, prevType)
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
