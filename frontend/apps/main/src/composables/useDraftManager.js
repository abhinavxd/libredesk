import { ref, watch } from 'vue'
import { watchDebounced, useStorage, useEventListener } from '@vueuse/core'
import { useConversationStore } from '@main/stores/conversation'
import { MACRO_CONTEXT } from '@main/constants/conversation'
import { getTextFromHTML } from '@main/utils/strings'
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
  const textContent = getTextFromHTML(draft.content || '')
  const hasAttachments = draft.meta?.attachments?.length > 0
  const hasMacroActions = draft.meta?.macro_actions?.length > 0
  return textContent.length === 0 && !hasAttachments && !hasMacroActions
}

/**
 * Composable for managing draft state and persistence
 * Saves to localStorage immediately, syncs to backend on conversation switch/send/unload
 * 
 * @param key - Reactive reference to current draft key
 * @param uploadedFiles - Optional reactive reference to uploaded files array
 */
export function useDraftManager (key, uploadedFiles = null) {
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
  const saveDraftLocal = (draftKey) => {
    if (!draftKey) return
    const macroActions = conversationStore.getMacro(MACRO_CONTEXT.REPLY)?.actions || []
    const draftMeta = {}
    if (macroActions.length > 0) {
      draftMeta.macro_actions = macroActions
    } else {
      delete draftMeta.macro_actions
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
    } else {
      delete draftMeta.attachments
    }

    // Save to localStorage
    localDrafts.value[draftKey] = { content: htmlContent.value, meta: draftMeta }

    // Mark as dirty for backend sync
    isDirty.value = true
  }

  /**
   * Get draft from localStorage
   */
  const getLocalDraft = (draftKey) => localDrafts.value[draftKey] || null

  /**
   * Remove draft from localStorage
   */
  const removeLocalDraft = (draftKey) => {
    if (localDrafts.value[draftKey]) {
      delete localDrafts.value[draftKey]
    }
  }

  /**
   * Sync localStorage draft to backend
   */
  const syncDraftToBackend = async (draftKey) => {
    if (!draftKey || !isDirty.value) return
    const localDraft = getLocalDraft(draftKey)
    if (!localDraft) return

    try {
      if (isDraftEmpty(localDraft)) {
        // Empty draft - delete instead of save
        await api.deleteDraft(draftKey)
        conversationStore.removeDraft(draftKey)
      } else {
        // Has content - save draft
        await api.saveDraft(draftKey, localDraft)
        conversationStore.setDraft(draftKey, localDraft)
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
  const loadDraft = async (draftKey) => {
    if (!draftKey) return
    isLoading.value = true
    isDirty.value = false
    skipNextSave.value = true
    try {
      // Check if there's an unsynced localStorage draft - sync it first
      const localDraft = getLocalDraft(draftKey)
      if (localDraft && !isDraftEmpty(localDraft)) {
        await api.saveDraft(draftKey, localDraft)
        conversationStore.setDraft(draftKey, localDraft)
      }
      removeLocalDraft(draftKey)

      // Load from store (drafts pre-fetched on app init)
      const draft = conversationStore.getDraft(draftKey)
      if (!draft) {
        resetState()
        return
      }

      const content = draft.content || ''
      const meta = draft.meta || {}

      // Check if draft is empty - if so, delete it and return
      if (isDraftEmpty({ content, meta })) {
        await api.deleteDraft(draftKey)
        conversationStore.removeDraft(draftKey)
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
  const clearDraft = async (draftKey) => {
    if (!draftKey) return
    removeLocalDraft(draftKey)
    try {
      await api.deleteDraft(draftKey)
      conversationStore.removeDraft(draftKey)
      resetState()
    } catch (error) {
      // Silent fail
    }
  }


  // Watch for key changes - sync to backend before switching
  watch(
    key,
    async (newKey, oldKey) => {
      // Block saves during transition to prevent race
      isTransitioning.value = true

      // Sync old draft to backend before switching
      if (newKey !== oldKey && isDirty.value) {
        await syncDraftToBackend(oldKey)
        removeLocalDraft(oldKey)
      }

      // Load new draft from backend
      if (newKey && newKey !== oldKey) {
        await loadDraft(newKey)
      } else if (!newKey && oldKey) {
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

      // Need to make sure not loading or transitioning, as during transition the `key` will change
      if (!isLoading.value && !isTransitioning.value && key.value) {
        saveDraftLocal(key.value)
      }
    },
    { debounce: 100, deep: true }
  )

  // Sync to backend when page is hidden (tab switch)
  useEventListener(document, 'visibilitychange', async () => {
    if (document.visibilityState === 'hidden' && isDirty.value && key.value) {
      await syncDraftToBackend(key.value)
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
