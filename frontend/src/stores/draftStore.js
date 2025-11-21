import { defineStore } from 'pinia'
import { useStorage } from '@vueuse/core'

const STORAGE_KEY = 'libredesk-conversation-drafts'

export const useDraftStore = defineStore('drafts', () => {
  // Reactive ref that auto-syncs with localStorage
  const drafts = useStorage(STORAGE_KEY, {}, localStorage, {
    serializer: {
      read: (v) => {
        try {
          return v ? JSON.parse(v) : {}
        } catch (error) {
          console.error('Failed to parse drafts:', error)
          return {}
        }
      },
      write: (v) => JSON.stringify(v)
    }
  })

  const getDraft = (uuid) => {
    if (!uuid) return { htmlContent: '', textContent: '' }
    return drafts.value[uuid] || { htmlContent: '', textContent: '' }
  }

  const setDraft = (uuid, htmlContent, textContent) => {
    if (!uuid) return
    
    drafts.value[uuid] = {
      htmlContent,
      textContent,
      timestamp: Date.now()
    }
  }

  const clearDraft = (uuid) => {
    if (!uuid) return
    delete drafts.value[uuid]
  }

  const clearAllDrafts = () => {
    drafts.value = {}
  }

  return {
    drafts,
    getDraft,
    setDraft,
    clearDraft,
    clearAllDrafts
  }
})