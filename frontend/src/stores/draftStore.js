import { defineStore } from 'pinia'
import { ref } from 'vue'
import { useDebounceFn } from '@vueuse/core'

const STORAGE_KEY = 'libredesk-conversation-drafts'

export const useDraftStore = defineStore('drafts', () => {
  // State
  const drafts = ref({})

  // Load from localStorage
  const loadDrafts = () => {
    try {
      const saved = localStorage.getItem(STORAGE_KEY)
      if (saved) {
        drafts.value = JSON.parse(saved)
      }
    } catch (error) {
      console.error('Failed to load drafts:', error)
      drafts.value = {}
    }
  }

  // Save to localStorage (immediate)
  const saveDrafts = () => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(drafts.value))
    } catch (error) {
      console.error('Failed to save drafts:', error)
    }
  }

  // Debounced save (500ms delay)
  const debouncedSave = useDebounceFn(saveDrafts, 500)

  // Get draft for a conversation
  const getDraft = (uuid) => {
    if (!uuid) return { htmlContent: '', textContent: '' }
    return drafts.value[uuid] || { htmlContent: '', textContent: '' }
  }

  // Set draft for a conversation
  const setDraft = (uuid, htmlContent, textContent) => {
    if (!uuid) return
    
    drafts.value[uuid] = {
      htmlContent,
      textContent,
      timestamp: Date.now()
    }
    
    debouncedSave()
  }

  // Clear draft for a conversation
  const clearDraft = (uuid) => {
    if (!uuid) return
    
    delete drafts.value[uuid]
    saveDrafts() // Immediate save for deletions
  }

  // Clear all drafts
  const clearAllDrafts = () => {
    drafts.value = {}
    saveDrafts()
  }

  return {
    drafts,
    loadDrafts,
    getDraft,
    setDraft,
    clearDraft,
    clearAllDrafts
  }
})