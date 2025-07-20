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

// Set content type if not specified and add libredesk_session to POST/PUT requests
http.interceptors.request.use((request) => {
    if ((request.method === 'post' || request.method === 'put') && !request.headers['Content-Type']) {
        request.headers['Content-Type'] = 'application/json'
    }

    // Add libredesk_session to POST/PUT request data
    if (request.method === 'post' || request.method === 'put') {
        const libredeskSession = localStorage.getItem('libredesk_session')
        request.data = {
            ...request.data,
            inbox_id: getInboxIDFromQuery(),
            jwt: libredeskSession
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

    // Add JWT token
    const libredeskSession = localStorage.getItem('libredesk_session')
    formData.append('jwt', libredeskSession)
    formData.append('conversation_uuid', conversationUUID)
    formData.append('inbox_id', getInboxIDFromQuery())

    // Add files
    for (let i = 0; i < files.length; i++) {
        formData.append('files', files[i])
    }

    return axios.post('/api/v1/widget/media/upload', formData, {
        headers: {
            'Content-Type': 'multipart/form-data'
        },
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
