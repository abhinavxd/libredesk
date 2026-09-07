import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useCopilotStore = defineStore('copilot', () => {
  // Copilot chat history keyed by conversation uuid; in-memory, resets on reload.
  const messages = ref(new Map())

  function getMessages (uuid) {
    return messages.value.get(uuid) || []
  }

  function setMessages (uuid, msgs) {
    messages.value.set(uuid, msgs)
    messages.value = new Map(messages.value)
  }

  function clearMessages (uuid) {
    messages.value.delete(uuid)
    messages.value = new Map(messages.value)
  }

  return {
    getMessages,
    setMessages,
    clearMessages
  }
})
