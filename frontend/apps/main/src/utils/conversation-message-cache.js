export default class MessageCache {
    // Not reactive - see widget/store/chat.js for the reactive wrapper pattern.
    constructor(maxConvs = 30) {
        this.cache = new Map()
        this.maxConvs = maxConvs
        this.recentConvs = []
    }

    addMessages (convId, messages, page, totalPages) {
        const conv = this.cache.get(convId)
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

    purgeConversation (convId) {
        return this.cache.delete(convId)
    }

    hasMessage (convId, msgId) {
        return this._allMessages(convId).some(m => m.uuid === msgId)
    }

    addMessage (convId, message) {
        const conv = this.cache.get(convId)
        if (!conv || this.hasMessage(convId, message.uuid)) return
        if (!conv.pages.has(1)) {
            conv.pages.set(1, [message])
        } else {
            conv.pages.get(1).push(message)
        }
    }

    getAllPagesMessages (convId) {
        return this._allMessages(convId)
            .sort((a, b) => new Date(a.created_at) - new Date(b.created_at))
    }

    getLatestMessage (convId, type = [], excludePrivate = false, excludeAutomated = false) {
        const filtered = this._allMessages(convId).filter(msg => {
            if (type.length > 0 && !type.includes(msg.type)) return false
            if (excludePrivate && msg.private) return false
            if (excludeAutomated && msg.meta?.is_automated) return false
            return true
        })
        filtered.sort((a, b) => new Date(b.created_at) - new Date(a.created_at))
        return filtered.length ? filtered[0] : null
    }

    updateMessage (convId, msgId, updates) {
        this._updateMessageBy(convId, msgId, msg => Object.assign(msg, updates))
    }

    updateMessageField (convId, msgId, field, value) {
        this._updateMessageBy(convId, msgId, msg => { msg[field] = value })
    }

    removeMessage (convId, msgId) {
        const conv = this.cache.get(convId)
        if (!conv) return
        conv.pages.forEach(msgs => {
            const msgIndex = msgs.findIndex(m => m.uuid === msgId)
            if (msgIndex !== -1) {
                msgs.splice(msgIndex, 1)
            }
        })
    }

    hasMore (convId) {
        return this.cache.get(convId)?.hasMore || false
    }

    getLastFetchedPage (convId) {
        return this.cache.get(convId)?.lastFetchedPage || 0
    }

    pruneOldConversations (convId) {
        this.recentConvs = [convId, ...this.recentConvs.filter(id => id !== convId)]
        if (this.recentConvs.length > this.maxConvs) {
            const removed = this.recentConvs.pop()
            this.cache.delete(removed)
        }
    }

    hasConversation (convId) {
        return this.cache.has(convId)
    }

    _allMessages (convId) {
        const conv = this.cache.get(convId)
        if (!conv) return []
        return Array.from(conv.pages.values()).flat()
    }

    _updateMessageBy (convId, msgId, mutate) {
        const conv = this.cache.get(convId)
        if (!conv) return
        conv.pages.forEach(msgs => {
            const msg = msgs.find(m => m.uuid === msgId)
            if (msg) mutate(msg)
        })
    }
}
