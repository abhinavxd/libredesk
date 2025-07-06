import { defineStore } from 'pinia'
import { ref, computed, reactive } from 'vue'
import { initWidgetWS } from '../websocket.js'
import api from '../api/index.js'
import MessageCache from '@main/utils/conversation-message-cache.js'
import { useUserStore } from './user.js'

export const useChatStore = defineStore('chat', () => {
    const userStore = useUserStore()
    // State
    const isTyping = ref(false)
    const currentConversation = ref({})
    const conversations = ref(null)
    // Conversation messages cache, evict old conversation messages after 50 conversations.
    const messageCache = reactive(new MessageCache(50))
    const isLoadingConversations = ref(false)

    // Getters
    const getCurrentConversationMessages = () => {
        const convId = currentConversation.value?.uuid
        if (!convId) return []
        return messageCache.getAllPagesMessages(convId)
    }
    const hasConversations = computed(() => conversations.value?.length > 0)
    const getConversations = computed(() => {
        // Sort by `last_message_at` descending.
        if (conversations.value) {
            return conversations.value.sort((a, b) => new Date(b.last_message_at) - new Date(a.last_message_at))
        }
        return []
    })

    // Actions
    const addMessageToConversation = (conversationUUID, message) => {
        messageCache.addMessage(conversationUUID, message)
        // Update `last_message` for the conversations list.
        const conv = conversations.value.find(c => c.uuid === conversationUUID)
        if (conv) {
            conv.last_message = message.text_content
            conv.last_message_at = message.created_at
        }
    }

    const fetchCurrentConversation = async () => {
        const conversationUUID = currentConversation.value?.uuid
        if (!conversationUUID) return

        // Set unread message count to 0 for the current conversation
        const conv = conversations.value.find(c => c.uuid === conversationUUID)
        if (conv) {
            conv.unread_message_count = 0
        }

        // If messages are already loaded, do nothing.
        if (messageCache.hasConversation(conversationUUID)) {
            return
        }

        // Fetch entire conversation and replace messages and conversation data
        try {
            const resp = await api.getChatConversation(conversationUUID)
            replaceMessages(resp.data.data.messages)
            currentConversation.value = resp.data.data.conversation
        } catch (error) {
            console.error('Error fetching conversation:', error)
        }
    }

    const fetchAndReplaceConversationAndMessages = async () => {
        const conversationUUID = currentConversation.value?.uuid
        if (!conversationUUID) return
        // Fetch entire conversation and replace messages
        try {
            const resp = await api.getChatConversation(conversationUUID)
            replaceMessages(resp.data.data.messages)
            currentConversation.value = resp.data.data.conversation
            // Set last message for the conversation
            const conv = conversations.value.find(c => c.uuid === conversationUUID)
            if (conv) {
                conv.last_message = resp.data.data.messages[0]?.content || ''
                conv.last_message_at = resp.data.data.messages[0]?.created_at || new Date().toISOString()
            }
        } catch (error) {
            console.error('Error fetching conversation:', error)
        }
    }

    const replaceMessages = (newMessages) => {
        const convId = currentConversation.value?.uuid
        if (!convId) return
        if (Array.isArray(newMessages) && newMessages.length > 0) {
            // Purge and then add messages.
            messageCache.purgeConversation(convId)
            messageCache.addMessages(convId, newMessages, 1, 1)
        }
    }

    const clearMessages = () => {
        const convId = currentConversation.value?.uuid
        if (!convId) return
        // Clear messages for current conversation by setting empty array.
        messageCache.addMessages(convId, [], 1, 1)
    }

    const setTypingStatus = (status) => {
        isTyping.value = status
    }

    const setCurrentConversation = (conversation) => {
        if (conversation === null) {
            conversation = {}
        }
        currentConversation.value = conversation
    }

    const openConversation = (conversation) => {
        // Set the current conversation
        setCurrentConversation(conversation)

        // Init WebSocket connection if not already initialized.
        const jwt = userStore.userSessionToken
        if (jwt) {
            initWidgetWS(jwt)
        }
    }

    const fetchConversations = async () => {
        if (!userStore.userSessionToken) {
            conversations.value = []
            return
        }

        if (conversations.value !== null) {
            return
        }

        try {
            isLoadingConversations.value = true
            const response = await api.getChatConversations()
            conversations.value = response.data.data || []
        } catch (error) {
            // On 401, clear session from user store.
            if (error.response && error.response.status === 401) {
                userStore.clearSessionToken()
                conversations.value = null
                return
            }
            console.error('Error fetching conversations:', error)
        } finally {
            isLoadingConversations.value = false
        }
    }

    const updateCurrentConversationLastSeen = async () => {
        const conversationUUID = currentConversation.value?.uuid
        if (!conversationUUID) return
        api.updateConversationLastSeen(conversationUUID)
    }

    return {
        // State
        messageCache,
        isTyping,
        conversations,
        currentConversation,
        isLoadingConversations,

        // Getters
        getCurrentConversationMessages,
        hasConversations,
        getConversations,

        // Actions
        addMessageToConversation,
        fetchCurrentConversation,
        replaceMessages,
        clearMessages,
        setTypingStatus,
        setCurrentConversation,
        openConversation,
        fetchConversations,
        fetchAndReplaceConversationAndMessages,
        updateCurrentConversationLastSeen
    }
})
