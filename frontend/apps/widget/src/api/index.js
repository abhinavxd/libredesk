import axios from 'axios'

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
            inbox_id: 11,
            jwt: libredeskSession
        }
    }

    return request
})

const getWidgetSettings = (inboxID) => http.get('/api/v1/widget/chat/settings', {
    params: { inbox_id: inboxID }
})
const initChatConversation = (data) => http.post('/api/v1/widget/chat/conversations/init', data)
const getChatConversations = () => http.post('/api/v1/widget/chat/conversations')
const getChatConversation = (uuid) => http.post(`/api/v1/widget/chat/conversations/${uuid}`)
const sendChatMessage = (uuid, data) => http.post(`/api/v1/widget/chat/conversations/${uuid}/message`, data)
const closeChatConversation = (uuid) => http.post(`/api/v1/widget/chat/conversations/${uuid}/close`)

export default {
    getWidgetSettings,
    initChatConversation,
    getChatConversations,
    getChatConversation,
    sendChatMessage,
    closeChatConversation
}
