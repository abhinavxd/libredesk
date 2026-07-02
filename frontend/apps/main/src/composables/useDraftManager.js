import { ref, watch } from 'vue'
import { watchDebounced, useStorage, useEventListener } from '@vueuse/core'
import { useConversationStore } from '@main/stores/conversation'
import { MACRO_CONTEXT } from '@main/constants/conversation'
import { getTextFromHTML } from '@shared-ui/utils/string.js'
import api from '@main/api'

/**
 * Validate macro actions have required structure
 */
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

/**
 * Validate attachments have required structure
 */
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

/**
 * Check if draft has no meaningful content
 */
const isDraftEmpty = (draft) => {
  if (!draft) return true
  const content = draft.content || ''
  const textContent = getTextFromHTML(content)
  const hasInlineImage = /<img\b/i.test(content)
  const hasAttachments = draft.meta?.attachments?.length > 0
  const hasMacroActions = draft.meta?.macro_actions?.length > 0
  return textContent.length === 0 && !hasInlineImage && !hasAttachments && !hasMacroActions
}

const draftKey = (uuid, type) => `${uuid}::${type}`

/**
 * Composable for managing draft state and persistence, keyed per conversation and message type
 * (reply / private note). Saves to localStorage immediately, syncs to backend on switch/send/unload.
 *
 * @param conversationUUID - Reactive reference to the current conversation UUID
 * @param messageType - Reactive reference to the current message type ('reply' | 'private_note')
 * @param uploadedFiles - Optional reactive reference to uploaded files array
 */
export function useDraftManager (conversationUUID, messageType, uploadedFiles = null) {
  const conversationStore = useConversationStore()
  const htmlContent = ref('')
  const textContent = ref('')
  const isLoading = ref(false)
  const isDirty = ref(false)
  const skipNextSave = ref(false)
  const loadedAttachments = ref([])
  const loadedMacroActions = ref([])
  const isTransitioning = ref(false)

  // Reactive localStorage for all drafts
  const localDrafts = useStorage('libredesk_drafts', {})

  /**
   * Save draft to localStorage only
   */
  const saveDraftLocal = (uuid, type) => {
    if (!uuid) return
    const macroActions = conversationStore.getMacro(MACRO_CONTEXT.REPLY)?.actions || []
    const draftMeta = {}
    if (macroActions.length > 0) {
      draftMeta.macro_actions = macroActions
    }

    // Set only required attachment fields
    if (uploadedFiles?.value?.length > 0) {
      draftMeta.attachments = uploadedFiles.value.map(file => ({
        id: file.id,
        size: file.size,
        uuid: file.uuid,
        filename: file.filename,
        content_type: file.content_type
      }))
    }

    localDrafts.value[draftKey(uuid, type)] = { content: htmlContent.value, meta: draftMeta }
    isDirty.value = true
  }

  const getLocalDraft = (uuid, type) => localDrafts.value[draftKey(uuid, type)] || null

  const removeLocalDraft = (uuid, type) => {
    delete localDrafts.value[draftKey(uuid, type)]
  }

  /**
   * Sync localStorage draft to backend
   */
  const syncDraftToBackend = async (uuid, type) => {
    if (!uuid || !isDirty.value) return
    const localDraft = getLocalDraft(uuid, type)
    if (!localDraft) return

    try {
      if (isDraftEmpty(localDraft)) {
        await api.deleteDraft(uuid, type)
        conversationStore.removeDraft(uuid, type)
      } else {
        await api.saveDraft(uuid, type, localDraft)
        conversationStore.setDraft(uuid, type, localDraft)
      }
      isDirty.value = false
    } catch (error) {
      // Silent fail - will retry on next sync
    }
  }

  /**
   * Reset all draft state to initial values
   */
  const resetState = () => {
    htmlContent.value = ''
    textContent.value = ''
    isLoading.value = false
    isDirty.value = false
    loadedAttachments.value = []
    loadedMacroActions.value = []
  }

  /**
   * Load draft from store (pre-fetched on app init)
   */
  const loadDraft = async (uuid, type) => {
    if (!uuid) return
    isLoading.value = true
    isDirty.value = false
    skipNextSave.value = true
    try {
      // Check if there's an unsynced localStorage draft - sync it first
      const localDraft = getLocalDraft(uuid, type)
      if (localDraft && !isDraftEmpty(localDraft)) {
        await api.saveDraft(uuid, type, localDraft)
        conversationStore.setDraft(uuid, type, localDraft)
      }
      removeLocalDraft(uuid, type)

      // Load from store (drafts pre-fetched on app init)
      const draft = conversationStore.getDraft(uuid, type)
      if (!draft) {
        resetState()
        return
      }

      const content = draft.content || ''
      const meta = draft.meta || {}

      // Check if draft is empty - if so, delete it and return
      if (isDraftEmpty({ content, meta })) {
        await api.deleteDraft(uuid, type)
        conversationStore.removeDraft(uuid, type)
        resetState()
        return
      }

      htmlContent.value = content
      textContent.value = ''
      loadedAttachments.value = validateAttachments(meta.attachments)
      loadedMacroActions.value = validateMacroActions(meta.macro_actions)
    } catch (error) {
      resetState()
    } finally {
      isLoading.value = false
    }
  }

  /**
   * Clear draft from both localStorage and backend
   */
  const clearDraft = async (uuid = conversationUUID.value, type = messageType.value) => {
    if (!uuid) return
    removeLocalDraft(uuid, type)
    try {
      await api.deleteDraft(uuid, type)
      conversationStore.removeDraft(uuid, type)
      resetState()
    } catch (error) {
      // Silent fail
    }
  }

  // Watch for conversation/type changes - sync old draft to backend before loading the new one
  watch(
    [conversationUUID, messageType],
    async (newVals, oldVals) => {
      const [newUuid, newType] = newVals
      const [oldUuid, oldType] = oldVals || []
      const changed = newUuid !== oldUuid || newType !== oldType
      if (!changed) return

      // Block saves during transition to prevent race
      isTransitioning.value = true

      if (oldUuid && isDirty.value) {
        await syncDraftToBackend(oldUuid, oldType)
        removeLocalDraft(oldUuid, oldType)
      }

      if (newUuid) {
        await loadDraft(newUuid, newType)
      } else {
        resetState()
      }

      // Allow saves after debounce window passes (200ms > 100ms debounce)
      setTimeout(() => {
        isTransitioning.value = false
      }, 200)
    },
    { immediate: true }
  )

  // Watch changes in draft content/meta to save locally
  const watchSources = [
    htmlContent,
    textContent,
    () => conversationStore.macros[MACRO_CONTEXT.REPLY]
  ]
  if (uploadedFiles) {
    watchSources.push(uploadedFiles)
  }

  watchDebounced(
    watchSources,
    () => {
      if (skipNextSave.value) {
        skipNextSave.value = false
        return
      }

      // Need to make sure not loading or transitioning, as during transition the key will change
      if (!isLoading.value && !isTransitioning.value && conversationUUID.value) {
        saveDraftLocal(conversationUUID.value, messageType.value)
      }
    },
    { debounce: 100, deep: true }
  )

  // Sync to backend when page is hidden (tab switch)
  useEventListener(document, 'visibilitychange', async () => {
    if (document.visibilityState === 'hidden' && isDirty.value && conversationUUID.value) {
      await syncDraftToBackend(conversationUUID.value, messageType.value)
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
