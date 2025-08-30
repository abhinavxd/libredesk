import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { handleHTTPError } from '@/utils/http'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import api from '@/api'

export const useInboxStore = defineStore('inbox', () => {
  const inboxes = ref([])
  const emitter = useEmitter()
  const options = computed(() => inboxes.value.map(inb => ({
    label: inb.name,
    value: String(inb.id)
  })))
  const fetchInboxes = async (force = false) => {
    if (!force && inboxes.value.length) return
    try {
      const response = await api.getInboxes()
      inboxes.value = response?.data?.data || []
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        title: 'Error',
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }
  return {
    inboxes,
    options,
    fetchInboxes,
  }
})