import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useWidgetStore = defineStore('widget', () => {
    // State
    const isOpen = ref(false)
    const currentView = ref('home')
    const config = ref({})
    const isInChatView = ref(false)
    const isMobileFullScreen = ref(false)


    // Getters
    const isHomeView = computed(() => currentView.value === 'home')
    const isChatView = computed(() => isInChatView.value)
    const isMessagesView = computed(() => currentView.value === 'messages' && !isInChatView.value)

    // Actions
    const toggleWidget = () => {
        isOpen.value = !isOpen.value
        isInChatView.value = false
    }

    const openWidget = () => {
        isOpen.value = true
    }

    const closeWidget = () => {
        isOpen.value = false
        currentView.value = 'home'
        isInChatView.value = false
    }

    const navigateToChat = () => {
        currentView.value = 'messages'
        isInChatView.value = true
    }

    const navigateToMessages = () => {
        currentView.value = 'messages'
        isInChatView.value = false
    }

    const navigateToHome = () => {
        currentView.value = 'home'
        isInChatView.value = false
    }

    const updateConfig = (newConfig) => {
        config.value = { ...newConfig }
    }

    const setMobileFullScreen = (isMobile) => {
        isMobileFullScreen.value = isMobile
    }

    return {
        // State
        isOpen,
        currentView,
        config,
        isInChatView,
        isMobileFullScreen,

        // Getters
        isHomeView,
        isChatView,
        isMessagesView,

        // Actions
        toggleWidget,
        openWidget,
        closeWidget,
        navigateToChat,
        navigateToMessages,
        navigateToHome,
        updateConfig,
        setMobileFullScreen,
    }
})
