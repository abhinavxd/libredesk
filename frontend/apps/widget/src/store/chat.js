import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useChatStore = defineStore('chat', () => {
    // State
    const messages = ref([])
    const isTyping = ref(false)
    const currentConversationId = ref(null)

    // Getters
    const hasMessages = computed(() => messages.value.length > 0)
    const messageCount = computed(() => messages.value.length)
    const getMessages = () => [...messages.value].reverse()


    // Actions
    const addMessage = (message) => {
        messages.value.push(message)
    }

    const replaceMessages = (newMessages) => {
        // Clear existing messages and replace with new ones
        messages.value = []

        if (Array.isArray(newMessages)) {
            messages.value = [...newMessages]
        }
    }

    const clearMessages = () => {
        messages.value = []
    }

    const setTypingStatus = (status) => {
        isTyping.value = status
    }

    const setCurrentConversationId = (conversationId) => {
        currentConversationId.value = conversationId
    }

    const findMessageByUuid = (uuid) => {
        return messages.value.find(msg => msg.uuid === uuid)
    }

    const updateMessageStatus = (uuid, status) => {
        const message = findMessageByUuid(uuid)
        if (message) {
            message.status = status
        }
    }

    const removeMessage = (uuid) => {
        const index = messages.value.findIndex(msg => msg.uuid === uuid)
        if (index !== -1) {
            messages.value.splice(index, 1)
        }
    }

    return {
        // State
        messages,
        isTyping,
        currentConversationId,

        // Getters
        getMessages,
        hasMessages,
        messageCount,

        // Actions
        addMessage,
        replaceMessages,
        clearMessages,
        setTypingStatus,
        setCurrentConversationId,
        findMessageByUuid,
        updateMessageStatus,
        removeMessage
    }
})
