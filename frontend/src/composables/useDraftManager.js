import { ref, watch } from 'vue'
import { watchDebounced } from '@vueuse/core'
import { useDraftStore } from '@/stores/draftStore'

/**
 * Composable for managing draft state and persistence
 * @param {Ref<string>} conversationKey - Reactive reference to current conversation UUID
 */
export function useDraftManager(conversationKey) {
  const draftStore = useDraftStore()
  
  const htmlContent = ref('')
  const textContent = ref('')
  const isLoadingDraft = ref(false)

  /**
   * Load draft from store for a given key
   */
  const loadDraft = (key) => {
    if (!key) return
    
    isLoadingDraft.value = true
  const draft = draftStore.getDraft(key)
    htmlContent.value = draft.htmlContent || ''
    textContent.value = draft.textContent || ''
    
    // Small delay to prevent race conditions with watchers
    setTimeout(() => {
      isLoadingDraft.value = false
    }, 600)
  }

  /**
   * Save draft to store
   */
  const saveDraft = (key) => {
    if (!key || isLoadingDraft.value) return
    draftStore.setDraft(key, htmlContent.value, textContent.value)
  }

  /**
   * Clear draft and local state
   */
  const clearDraft = (key) => {
    if (!key) return
  
    isLoadingDraft.value = true
    draftStore.clearDraft(key)
    htmlContent.value = ''
    textContent.value = ''
  
  setTimeout(() => {
    isLoadingDraft.value = false
  }, 600) 
}

  /**
   * Check if draft has content
   */
  const hasDraftContent = () => {
    return (htmlContent.value?.trim() || '') !== '' || (textContent.value?.trim() || '') !== ''
  }

// Watch for conversation key changes
watch(
  conversationKey,
  (newKey, oldKey) => {
    // Save old draft BEFORE switching (whether going to new or existing conversation)
    if (oldKey && hasDraftContent()) {
      draftStore.setDraft(oldKey, htmlContent.value, textContent.value)
    }
    
    // Only load draft if switching to an EXISTING conversation with a different key
    if (newKey && newKey !== oldKey) {
      loadDraft(newKey)
    } else if (!newKey && oldKey) {
      // Clear draft if switching to a NEW conversation
      isLoadingDraft.value = true
      
      htmlContent.value = ''
      textContent.value = ''

      setTimeout(() => {
        isLoadingDraft.value = false
      }, 600)
    }
  },
  { immediate: true }
)

  // Auto-save draft when content changes (debounced to avoid excessive writes)
  watchDebounced(
    [htmlContent, textContent],
    () => {
      if (!isLoadingDraft.value && conversationKey.value) {
        saveDraft(conversationKey.value)
      }
    },
    { debounce: 500 }
  )

  return {
    htmlContent,
    textContent,
    isLoadingDraft,
    loadDraft,
    saveDraft,
    clearDraft,
    hasDraftContent
  }
}
