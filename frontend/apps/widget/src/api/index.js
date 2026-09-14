import axios from 'axios'

let _sessionToken = ''
let _visitorToken = ''

function postToParent (data) {
    if (window.parent && window.parent !== window) {
        window.parent.postMessage(data, '*')
    }
}

function getInboxIDFromQuery () {
    const params = new URLSearchParams(window.location.search)
    return params.get('inbox_id') || null
}

export function setApiSessionToken (token) {
    _sessionToken = token || ''
}

export function setVisitorToken (token) {
    _visitorToken = token
    postToParent({ type: 'STORE_VISITOR_TOKEN', token })
}

export function clearVisitorToken () {
    _visitorToken = ''
    postToParent({ type: 'CLEAR_VISITOR_TOKEN' })
}

export function getVisitorToken () {
    return _visitorToken || null
}

export function initVisitorToken (token) {
    _visitorToken = token || ''
}

// Stores registered by App.vue for use in the response interceptor.
let _stores = null
export function registerStores (stores) {
    _stores = stores
}

// Clears all session state, cookies, and closes widget on 401/session expiry.
function handleSessionExpired () {
    if (!_stores) return
    const { userStore, chatStore, widgetStore } = _stores
    userStore.clearSessionToken()
    clearVisitorToken()
    postToParent({ type: 'CLEAR_SESSION_TOKEN' })
    chatStore.setCurrentConversation(null)
    chatStore.conversations = null
    widgetStore.closeWidget()
}

// Saves session token and user metadata from a server response.
// When isNewVisitor is true, also stores the token as the visitor token (for merge flow).
export function saveSession (sessionToken, user, userStore, isNewVisitor = false) {
    userStore.setSessionToken(sessionToken)
    setApiSessionToken(sessionToken)
    if (user) userStore.setUserMeta(user)
    if (isNewVisitor) setVisitorToken(sessionToken)
    postToParent({ type: 'STORE_SESSION', token: sessionToken })
}

// Returns visitor token if current user is a verified contact (for merge).
function getVisitorTokenForMerge () {
    const vt = getVisitorToken()
    if (!vt || !_sessionToken || vt === _sessionToken) {
        return null
    }
    return vt
}

const http = axios.create({
    timeout: 10000,
    responseType: 'json'
})

// Set content type and authentication headers
http.interceptors.request.use((request) => {
    if ((request.method === 'post' || request.method === 'put') && !request.headers['Content-Type']) {
        request.headers['Content-Type'] = 'application/json'
    }

    // Add authentication headers for widget API endpoints
    if (request.url && request.url.includes('/api/v1/widget/')) {
        const inboxId = getInboxIDFromQuery()

        if (_sessionToken) {
            request.headers['Authorization'] = `Bearer ${_sessionToken}`
        }

        if (inboxId) {
            request.headers['X-Libredesk-Inbox-ID'] = inboxId.toString()
        }

        const visitorTokenForMerge = getVisitorTokenForMerge()
        if (visitorTokenForMerge) {
            request.headers['X-Libredesk-Visitor-Token'] = visitorTokenForMerge
        }
    }

    return request
})

http.interceptors.response.use(
    (response) => {
        if (response.headers['x-libredesk-clear-visitor']) {
            clearVisitorToken()
        }
        return response
    },
    (error) => {
        if (error.response?.status === 401) {
            // Only cleanup if the failed request used the current session token.
            // Prevents clearing a valid new session when a stale request returns 401.
            const reqAuth = error.config?.headers?.Authorization
            if (reqAuth && reqAuth === `Bearer ${_sessionToken}`) {
                handleSessionExpired()
            }
        }
        return Promise.reject(error)
    }
)

const getWidgetSettings = (inboxID) => http.get('/api/v1/widget/chat/settings', {
    params: { inbox_id: inboxID }
})
const getLanguage = (lang) => http.get(`/api/v1/lang/${lang}`)
const getAvailableLanguages = () => http.get('/api/v1/lang')
const exchangeJWTForSession = (jwt) => http.post('/api/v1/widget/chat/auth/exchange', { jwt })
const getAuthMe = () => http.get('/api/v1/widget/chat/auth/me')
const initChatConversation = (data) => http.post('/api/v1/widget/chat/conversations/init', data)
const getChatConversations = () => http.get('/api/v1/widget/chat/conversations')
const getChatConversation = (uuid) => http.get(`/api/v1/widget/chat/conversations/${uuid}`)
const sendChatMessage = (uuid, data) => http.post(`/api/v1/widget/chat/conversations/${uuid}/message`, data)
const closeChatConversation = (uuid) => http.post(`/api/v1/widget/chat/conversations/${uuid}/close`)
const uploadMedia = (conversationUUID, files) => {
    const formData = new FormData()
    formData.append('conversation_uuid', conversationUUID)
    for (let i = 0; i < files.length; i++) {
        formData.append('files', files[i])
    }
    return http.post('/api/v1/widget/media/upload', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
        timeout: 120000
    })
}
const updateConversationLastSeen = (uuid) => http.post(`/api/v1/widget/chat/conversations/${uuid}/update-last-seen`)
const submitCSATResponse = (csatUuid, rating, feedback) =>
    http.post(`/api/v1/csat/${csatUuid}/response`, {
        rating,
        feedback,
    })

export default {
    getWidgetSettings,
    getLanguage,
    getAvailableLanguages,
    exchangeJWTForSession,
    getAuthMe,
    initChatConversation,
    getChatConversations,
    getChatConversation,
    sendChatMessage,
    closeChatConversation,
    uploadMedia,
    updateConversationLastSeen,
    submitCSATResponse
}
