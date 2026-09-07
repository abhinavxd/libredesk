import { defineStore } from 'pinia'
import { ref, computed, reactive } from 'vue'
import api from '../api/index.js'
import { deepMerge } from '@shared-ui/utils/object.js'
import { TYPING_RECEIVE_TIMEOUT } from '@shared-ui/composables/useTypingIndicator.js'
import MessageCache from '@main/utils/conversation-message-cache.js'
import { useUserStore } from './user.js'

export const useChatStore = defineStore('chat', () => {
    const userStore = useUserStore()
    // State
    let typingTimeout = null
    const isTyping = ref(false)
    const currentConversation = ref({})
    const conversations = ref(null)
    // Conversation messages cache, evict old conversation messages after 50 conversations.
    const messageCache = reactive(new MessageCache(50))
    const isLoadingConversations = ref(false)
    const isLoadingConversation = ref(false)
    // Reactivity trigger for message cache changes this is easier than making the whole messageCache reactive.
    const messageCacheVersion = ref(0)

    // Getters
    const getCurrentConversationMessages = computed(() => {
        messageCacheVersion.value // Force reactivity tracking
        const convId = currentConversation.value?.uuid
        if (!convId) return []
        return messageCache.getAllPagesMessages(convId)
    })
    const hasConversations = computed(() => conversations.value?.length > 0)
    const getConversations = computed(() => {
        // Sort by `last_message.created_at` descending.
        if (conversations.value) {
            return conversations.value.sort((a, b) => new Date(b.last_message.created_at) - new Date(a.last_message.created_at))
        }
        return []
    })

    const updateConversationListLastMessage = (conversationUUID, message, incrementUnread = false) => {
        if (!conversations.value || !Array.isArray(conversations.value)) return

        // Find conversation in the list
        const conv = conversations.value.find(c => c.uuid === conversationUUID)
        if (!conv) return

        // Update last_message in the conversation
        conv.last_message = {
            content: message.text_content !== '' ? message.text_content : message.content,
            created_at: message.created_at,
            status: message.status,
            author: {
                id: message.author.id,
                first_name: message.author.first_name || '',
                last_name: message.author.last_name || '',
                avatar_url: message.author.avatar_url || '',
                availability_status: message.author.availability_status || '',
                type: message.author.type || '',
                active_at: message.author.active_at || null
            }
        }

        // Increment unread count if needed
        if (incrementUnread) {
            conv.unread_message_count = (conv.unread_message_count || 0) + 1
        }
    }

    const addMessageToConversation = (conversationUUID, message) => {
        messageCache.addMessage(conversationUUID, message)
        messageCacheVersion.value++ // Trigger reactivity
        const shouldIncrementUnread = message.author?.type === 'agent'
        updateConversationListLastMessage(conversationUUID, message, shouldIncrementUnread)
    }

    const addPendingMessage = (conversationUUID, messageText, authorType, authorId, files = []) => {
        // Pending message is a temporary message that will be replaced with actual message later after sending.
        const pendingMessage = {
            content: messageText,
            content_type: 'text',
            author: {
                type: authorType,
                id: authorId,
                first_name: userStore.firstName || '',
                last_name: userStore.lastName || '',
                avatar_url: '',
                availability_status: '',
                active_at: null
            },
            attachments: [],
            uuid: `pending-${Date.now()}`,
            status: files.length > 0 ? 'uploading' : 'sending',
            created_at: new Date().toISOString()
        }
        messageCache.addMessage(conversationUUID, pendingMessage)
        messageCacheVersion.value++ // Trigger reactivity

        // Update conversations list with pending message
        updateConversationListLastMessage(conversationUUID, pendingMessage)

        return pendingMessage.uuid
    }

    const replaceMessage = (conversationUUID, msgID, actualMessage) => {
        messageCache.updateMessage(conversationUUID, msgID, actualMessage)
        messageCacheVersion.value++ // Trigger reactivity
        updateConversationListLastMessage(conversationUUID, actualMessage)
    }

    const removeMessage = (conversationUUID, msgID) => {
        messageCache.removeMessage(conversationUUID, msgID)
        messageCacheVersion.value++ // Trigger reactivity
    }

    const loadConversation = async (conversationUUID, force = false, silent = false) => {
        if (!conversationUUID) return false

        // If the conversation is already loaded, do not fetch again unless forced.
        if (currentConversation.value?.uuid === conversationUUID && !force) {
            return true
        }

        try {
            if (!silent) isLoadingConversation.value = true
            const resp = await api.getChatConversation(conversationUUID)
            const conversation = resp.data.data.conversation
            conversation.business_hours_id = resp.data.data.business_hours_id
            conversation.working_hours_utc_offset = resp.data.data.working_hours_utc_offset
            setCurrentConversation(conversation)
            replaceMessages(resp.data.data.messages)
            if (resp.data.data.messages.length > 0) {
                updateConversationListLastMessage(conversationUUID, resp.data.data.messages[0], false)
            }
        } catch (error) {
            console.error('Error fetching conversation:', error)
            return false
        } finally {
            isLoadingConversation.value = false
        }
        return true
    }

    const replaceMessages = (newMessages) => {
        const convId = currentConversation.value?.uuid
        if (!convId) return
        if (Array.isArray(newMessages) && newMessages.length > 0) {
            // Purge and then add messages.
            messageCache.purgeConversation(convId)
            messageCache.addMessages(convId, newMessages, 1, 1)
        }
        messageCacheVersion.value++ // Trigger reactivity
    }

    const clearMessages = () => {
        const convId = currentConversation.value?.uuid
        if (!convId) return
        // Clear messages for current conversation by setting empty values.
        messageCache.addMessages(convId, [], 1, 1)
        messageCacheVersion.value++ // Trigger reactivity
    }

    const setTypingStatus = (conversationUUID, status) => {
        if (!conversationUUID) return
        if (currentConversation.value?.uuid !== conversationUUID) return

        if (typingTimeout) {
            clearTimeout(typingTimeout)
            typingTimeout = null
        }

        isTyping.value = status

        if (status) {
            typingTimeout = setTimeout(() => {
                isTyping.value = false
                typingTimeout = null
            }, TYPING_RECEIVE_TIMEOUT)
        }
    }

    const setCurrentConversation = (conversation) => {
        if (conversation === null) {
            conversation = {}
        }
        // Clear messages if conversation is null or empty.
        if (!conversation) {
            clearMessages()
        }
        isTyping.value = false
        if (typingTimeout) {
            clearTimeout(typingTimeout)
            typingTimeout = null
        }
        currentConversation.value = conversation
    }

    const fetchConversations = async (force = false, silent = false) => {
        // No session token means visitor, no conversations to fetch.
        if (!userStore.userSessionToken) {
            return true
        }

        // If conversations are already loaded and is an array, do not fetch again unless forced.
        if (!force && Array.isArray(conversations.value)) {
            return true
        }

        try {
            if (!silent) isLoadingConversations.value = true
            const response = await api.getChatConversations()
            conversations.value = response.data.data || []
            return true
        } catch {
            return false
        } finally {
            isLoadingConversations.value = false
        }
    }

    const updateCurrentConversationLastSeen = async () => {
        const conversationUUID = currentConversation.value?.uuid
        if (!conversationUUID) return
        try {
            await api.updateConversationLastSeen(conversationUUID)
            // Reset unread count for current conversation
            if (conversations.value && Array.isArray(conversations.value)) {
                const conv = conversations.value.find(c => c.uuid === conversationUUID)
                if (conv) {
                    conv.unread_message_count = 0
                }
            }
        } catch (error) {
            console.error('Error updating last seen:', error)
        }
    }

    const updateCurrentConversation = (data) => {
        if (currentConversation.value?.uuid === data.uuid) {
            deepMerge(currentConversation.value, data)
        }

        if (conversations.value && Array.isArray(conversations.value)) {
            const index = conversations.value.findIndex(c => c.uuid === data.uuid)
            if (index >= 0) {
                deepMerge(conversations.value[index], data)
            }
        }
    }

    const addConversationToList = (conversation) => {
        conversation.unread_message_count = 0
        if (!conversations.value) {
            conversations.value = []
        }
        const existingIndex = conversations.value.findIndex(c => c.uuid === conversation.uuid)
        if (existingIndex >= 0) {
            conversations.value[existingIndex] = conversation
            return
        }
        conversations.value.push(conversation)
    }

    return {
        // State
        messageCache,
        isTyping,
        conversations,
        currentConversation,
        isLoadingConversations,
        isLoadingConversation,

        // Getters
        getCurrentConversationMessages,
        hasConversations,
        getConversations,

        // Actions
        addMessageToConversation,
        addPendingMessage,
        replaceMessage,
        removeMessage,
        replaceMessages,
        clearMessages,
        setTypingStatus,
        setCurrentConversation,
        fetchConversations,
        loadConversation,
        updateCurrentConversationLastSeen,
        updateConversationListLastMessage,
        updateCurrentConversation,
        addConversationToList
    }
})
