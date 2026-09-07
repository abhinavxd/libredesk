import { ref } from 'vue'
import { defineStore } from 'pinia'

// The contact currently open on the contact detail page.
export const useContactStore = defineStore('contact', () => {
    const current = ref(null)

    const setCurrent = (contact) => {
        current.value = contact
    }

    const clearCurrent = () => {
        current.value = null
    }

    return { current, setCurrent, clearCurrent }
})
