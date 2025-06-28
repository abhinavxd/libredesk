import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useWidgetStore = defineStore('widget', () => {
    // State
    const isOpen = ref(false)
    const currentView = ref('welcome')
    const config = ref({})


    // Getters
    const isWelcomeView = computed(() => currentView.value === 'welcome')
    const isChatView = computed(() => currentView.value === 'chat')

    // Actions
    const toggleWidget = () => {
        isOpen.value = !isOpen.value
        if (!isOpen.value) {
            // Reset to welcome view when closing
            currentView.value = 'welcome'
        }
    }

    const openWidget = () => {
        isOpen.value = true
    }

    const closeWidget = () => {
        isOpen.value = false
        currentView.value = 'welcome'
    }

    const navigateToChat = () => {
        currentView.value = 'chat'
    }

    const navigateToWelcome = () => {
        currentView.value = 'welcome'
    }

    const updateConfig = (newConfig) => {
        config.value = { ...newConfig }
    }

    return {
        // State
        isOpen,
        currentView,
        config,

        // Getters
        isWelcomeView,
        isChatView,

        // Actions
        toggleWidget,
        openWidget,
        closeWidget,
        navigateToChat,
        navigateToWelcome,
        updateConfig,
    }
})
