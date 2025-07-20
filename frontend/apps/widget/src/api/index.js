import axios from 'axios'

function getInboxIDFromQuery () {
    const params = new URLSearchParams(window.location.search)
    const inboxId = params.get('inbox_id')
    return inboxId ? parseInt(inboxId, 10) : null
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
        const libredeskSession = localStorage.getItem('libredesk_session')
        const inboxId = getInboxIDFromQuery()
        
        // Add JWT to Authorization header
        if (libredeskSession) {
            request.headers['Authorization'] = `Bearer ${libredeskSession}`
        }
        
        // Add inbox ID to custom header
        if (inboxId) {
            request.headers['X-Libredesk-Inbox-ID'] = inboxId.toString()
        }
    }

    return request
})

const getWidgetSettings = (inboxID) => http.get('/api/v1/widget/chat/settings', {
    params: { inbox_id: inboxID }
})
const getLanguage = (lang) => http.get(`/api/v1/lang/${lang}`)
const initChatConversation = (data) => http.post('/api/v1/widget/chat/conversations/init', data)
const getChatConversations = () => http.post('/api/v1/widget/chat/conversations')
const getChatConversation = (uuid) => http.post(`/api/v1/widget/chat/conversations/${uuid}`)
const sendChatMessage = (uuid, data) => http.post(`/api/v1/widget/chat/conversations/${uuid}/message`, data)
const closeChatConversation = (uuid) => http.post(`/api/v1/widget/chat/conversations/${uuid}/close`)
const uploadMedia = (conversationUUID, files) => {
    const formData = new FormData()
    
    // Only add conversation UUID to form data now
    formData.append('conversation_uuid', conversationUUID)

    // Add files
    for (let i = 0; i < files.length; i++) {
        formData.append('files', files[i])
    }

    // Get authentication data for headers
    const libredeskSession = localStorage.getItem('libredesk_session')
    const inboxId = getInboxIDFromQuery()

    const headers = {
        'Content-Type': 'multipart/form-data'
    }

    // Add authentication headers
    if (libredeskSession) {
        headers['Authorization'] = `Bearer ${libredeskSession}`
    }
    if (inboxId) {
        headers['X-Libredesk-Inbox-ID'] = inboxId.toString()
    }

    return axios.post('/api/v1/widget/media/upload', formData, {
        headers,
        timeout: 30000
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
    initChatConversation,
    getChatConversations,
    getChatConversation,
    sendChatMessage,
    closeChatConversation,
    uploadMedia,
    updateConversationLastSeen,
    submitCSATResponse
}
