export default class MessageCache {
    /**
     * Cache for conversation messages with eviction of old conversations
     * @param {number} maxConvs - Max conversations to store before eviction
     */
    constructor(maxConvs = 100) {
        this.cache = new Map()
        this.maxConvs = maxConvs
        this.recentConvs = []
    }

    /**
     * Adds or updates messages for a conversation page
     * Updates cache metadata like lastFetchedPage and hasMore
     */
    addMessages (convId, messages, page, totalPages) {
        const conv = this.cache.get(convId)
        // Filter out messages already present in cache.
        const uniqueMsgs = messages.filter(m => !this.hasMessage(convId, m.uuid))

        if (conv) {
            conv.lastFetchedPage = Math.max(page, conv.lastFetchedPage)
            conv.hasMore = totalPages > conv.lastFetchedPage
            conv.totalPages = totalPages
            conv.pages.set(page, uniqueMsgs)
        } else {
            this.cache.set(convId, {
                pages: new Map([[page, uniqueMsgs]]),
                totalPages,
                lastFetchedPage: page,
                hasMore: totalPages > page,
            })
            this.pruneOldConversations(convId)
        }
    }

    /**
    * Checks if message exists in conversation
    * @returns {boolean} 
    */
    hasMessage (convId, msgId) {
        const conv = this.cache.get(convId)
        if (!conv) return false
        return Array.from(conv.pages.values()).some(msgs => msgs.some(m => m.uuid === msgId))
    }

    /**
    * Adds single message to a conversation if not already present
    * 
    * @param {string} convId - Conversation ID
    * @param {object} message - Message with uuid field
    */
    addMessage (convId, message) {
        const conv = this.cache.get(convId)
        if (!conv || this.hasMessage(convId, message.uuid)) return
        if (!conv.pages.has(1)) {
            conv.pages.set(1, [message])
        } else {
            conv.pages.get(1).push(message)
        }
    }

    /**
     * Returns all cached messages for a conversation sorted by creation time
     */
    getAllPagesMessages (convId) {
        return Array.from(this.cache.get(convId)?.pages.values() || [])
            .flat()
            .sort((a, b) => new Date(a.created_at) - new Date(b.created_at))
    }

    /**
     * Returns latest message for a conversation
     * @param {string} convId - Conversation ID
     * @param {string[]} type - Array of message types to filter - outgoing, incoming, etc.
     * @param {boolean} excludePrivate - Exclude private messages
     * 
     * @returns {object} - Latest message object or null if not found
     */
    getLatestMessage (convId, type = [], excludePrivate = false) {
        const conv = this.cache.get(convId)
        if (!conv) return null

        // Get all messages from all pages
        let allMessages = Array.from(conv.pages.values()).flat()

        // Apply filters
        if (type.length > 0) {
            allMessages = allMessages.filter(msg => type.includes(msg.type))
        }
        if (excludePrivate) {
            allMessages = allMessages.filter(msg => !msg.private)
        }

        // Sort messages by created_at in descending order (newest first)
        allMessages.sort((a, b) => new Date(b.created_at) - new Date(a.created_at))

        return allMessages.length ? allMessages[0] : null
    }
    /**
     * Updates message fields by applying update object
     */
    updateMessage (convId, msgId, updates) {
        const conv = this.cache.get(convId)
        if (!conv) return
        conv.pages.forEach(msgs => {
            const msg = msgs.find(m => m.uuid === msgId)
            if (msg) Object.assign(msg, updates)
        })
    }

    /**
     * Updates a single field in a message
     */
    updateMessageField (convId, msgId, field, value) {
        const conv = this.cache.get(convId)
        if (!conv) return
        conv.pages.forEach(msgs => {
            const msg = msgs.find(m => m.uuid === msgId)
            if (msg) msg[field] = value
        })
    }

    /**
     * Checks if conversation has more pages to fetch
     */
    hasMore (convId) {
        return this.cache.get(convId)?.hasMore || false
    }

    /**
     * Returns last fetched page number for a conversation
     */
    getLastFetchedPage (convId) {
        return this.cache.get(convId)?.lastFetchedPage || 0
    }

    /**
     * pruneOldConversations - Evicts old conversations from cache
     */
    pruneOldConversations (convId) {
        this.recentConvs = [convId, ...this.recentConvs.filter(id => id !== convId)]
        if (this.recentConvs.length > this.maxConvs) {
            const removed = this.recentConvs.pop()
            this.cache.delete(removed)
        }
    }
}