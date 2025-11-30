import { defineStore } from 'pinia'
import { useStorage } from '@vueuse/core'

const STORAGE_KEY = 'libredesk-conversation-drafts'
const MAX_ENTRIES = 10

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
    
    const isEmpty = (!htmlContent || htmlContent.trim() === '') &&
                  (!textContent || textContent.trim() === '')

    if (isEmpty) return
    drafts.value[uuid] = {
      htmlContent,
      textContent,
      timestamp: Date.now()
    }
    const keys = Object.keys(drafts.value)
    if (keys.length > MAX_ENTRIES) {
      const sorted = keys
        .map(k => [k, drafts.value[k].timestamp])
        .sort((a, b) => a[1] - b[1])
      const removeCount = keys.length - MAX_ENTRIES
      for (let i = 0; i < removeCount; i++) {
        delete drafts.value[sorted[i][0]]
      }
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