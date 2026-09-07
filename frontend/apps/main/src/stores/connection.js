import { ref } from 'vue'
import { defineStore } from 'pinia'

export const useConnectionStore = defineStore('connection', () => {
  const connecting = ref(false)
  const connectionFailed = ref(false)

  const setConnecting = (value) => {
    connecting.value = value
  }

  const setConnectionFailed = (failed) => {
    connectionFailed.value = failed
  }

  return {
    connecting,
    connectionFailed,

    setConnecting,
    setConnectionFailed
  }
})
