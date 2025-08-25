import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useWidgetStore = defineStore('widget', () => {
    // State
    const isOpen = ref(false)
    const currentView = ref('home')
    const config = ref({})
    const isInChatView = ref(false)
    const isMobileFullScreen = ref(false)
    const isExpanded = ref(false)
    const wasExpandedBeforeLeaving = ref(false)


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
        // Clear expanded state memory when widget is closed
        wasExpandedBeforeLeaving.value = false
        
        isOpen.value = false
        currentView.value = 'home'
        isInChatView.value = false
        // Auto-collapse when closing widget
        if (isExpanded.value) {
            collapseWidget()
        }
    }

    const navigateToChat = () => {
        currentView.value = 'messages'
        isInChatView.value = true
        // Restore expanded state if it was expanded before leaving
        if (wasExpandedBeforeLeaving.value && !isMobileFullScreen.value) {
            setTimeout(() => {
                expandWidget()
            }, 100)
        }
    }

    const navigateToMessages = () => {
        // Remember expanded state before leaving chat view
        wasExpandedBeforeLeaving.value = isExpanded.value
        
        currentView.value = 'messages'
        isInChatView.value = false
        // Auto-collapse when leaving chat view
        if (isExpanded.value) {
            collapseWidget()
        }
    }

    const navigateToHome = () => {
        // Remember expanded state before leaving chat view
        wasExpandedBeforeLeaving.value = isExpanded.value
        
        currentView.value = 'home'
        isInChatView.value = false
        // Auto-collapse when leaving chat view
        if (isExpanded.value) {
            collapseWidget()
        }
    }

    const updateConfig = (newConfig) => {
        config.value = { ...newConfig }
    }

    const setMobileFullScreen = (isMobile) => {
        isMobileFullScreen.value = isMobile
    }

    const toggleExpand = () => {
        if (isExpanded.value) {
            collapseWidget()
        } else {
            expandWidget()
        }
    }

    const expandWidget = () => {
        if (!isMobileFullScreen.value) {
            isExpanded.value = true
            window.parent.postMessage({ type: 'EXPAND_WIDGET' }, '*')
        }
    }

    const collapseWidget = () => {
        if (!isMobileFullScreen.value) {
            isExpanded.value = false
            window.parent.postMessage({ type: 'COLLAPSE_WIDGET' }, '*')
        }
    }

    const setExpanded = (expanded) => {
        isExpanded.value = expanded
    }

    return {
        // State
        isOpen,
        currentView,
        config,
        isInChatView,
        isMobileFullScreen,
        isExpanded,
        wasExpandedBeforeLeaving,

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
        toggleExpand,
        expandWidget,
        collapseWidget,
        setExpanded,
    }
})
