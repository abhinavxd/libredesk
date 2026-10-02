import { ref } from 'vue'
import { defineStore } from 'pinia'

// Opens the Compose New dialog from anywhere. A prefill carries a returned
// submission so its author can edit and resubmit it.
export const useComposeStore = defineStore('compose', () => {
  const isOpen = ref(false)
  const prefill = ref(null)

  function open(initial = null) {
    prefill.value = initial
    isOpen.value = true
  }

  function close() {
    isOpen.value = false
    prefill.value = null
  }

  return { isOpen, prefill, open, close }
})
