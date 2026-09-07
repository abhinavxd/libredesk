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
    const connectionFailed = ref(false)
    const connecting = ref(false)
    const connected = ref(false)


    // Getters
    const isChatView = computed(() => isInChatView.value)

    // Actions
    const setOpen = (open) => {
        isOpen.value = open
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
        // Tell the parent loader to hide the iframe.
        window.parent.postMessage({ type: 'CLOSE_WIDGET' }, '*')
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
        // Only remember expanded state when leaving from chat view
        if (isInChatView.value) {
            wasExpandedBeforeLeaving.value = isExpanded.value
        }

        currentView.value = 'messages'
        isInChatView.value = false
        if (isExpanded.value) {
            collapseWidget()
        }
    }

    const navigateToHome = () => {
        // Only remember expanded state when leaving from chat view
        if (isInChatView.value) {
            wasExpandedBeforeLeaving.value = isExpanded.value
        }

        currentView.value = 'home'
        isInChatView.value = false
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

    const setConnectionFailed = (failed) => {
        connectionFailed.value = failed
    }

    const setConnecting = (value) => {
        connecting.value = value
    }

    const setConnected = (value) => {
        connected.value = value
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
        connectionFailed,
        connecting,
        connected,

        // Getters
        isChatView,

        // Actions
        setOpen,
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
        setConnectionFailed,
        setConnecting,
        setConnected,
    }
})
