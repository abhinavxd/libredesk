import axios from 'axios'
import qs from 'qs'

const http = axios.create({
  timeout: 10000,
  responseType: 'json'
})

// LLM calls can take 30-40s+, well past the default request timeout.

function getCSRFToken() {
  const name = 'csrf_token='
  const cookies = document.cookie.split(';')
  for (let i = 0; i < cookies.length; i++) {
    let c = cookies[i].trim()
    if (c.indexOf(name) === 0) {
      return c.substring(name.length, c.length)
    }
  }
  return ''
}

// Route-scoped abort, opt-in via { abortOnRoute: true }. Default no-abort protects in-flight saves.
let routeAbort = new AbortController()
export function abortRouteScope() {
  routeAbort.abort()
  routeAbort = new AbortController()
}

http.interceptors.request.use((request) => {
  const token = getCSRFToken()
  if (token) {
    request.headers['X-CSRFTOKEN'] = token
  }

  if ((request.method === 'post' || request.method === 'put') && !request.headers['Content-Type']) {
    request.headers['Content-Type'] = 'application/json'
  }

  if (request.headers['Content-Type'] === 'application/x-www-form-urlencoded') {
    request.data = qs.stringify(request.data)
  }

  if (request.abortOnRoute && request.signal === undefined) {
    request.signal = routeAbort.signal
  }

  return request
})



const getTemplate = (id) => http.get(`/api/v1/templates/${id}`)
const getTemplates = (type) => http.get('/api/v1/templates', { params: { type: type } })
const createTemplate = (data) =>
  http.post('/api/v1/templates', data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const deleteTemplate = (id) => http.delete(`/api/v1/templates/${id}`)
const updateTemplate = (id, data) =>
  http.put(`/api/v1/templates/${id}`, data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })

const createOIDC = (data) =>
  http.post('/api/v1/oidc', data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const getConfig = () => http.get('/api/v1/config')
const getResourceUsage = () => http.get('/api/v1/system/resource-usage')
const getResourceLimits = () => http.get('/api/v1/system/resource-limits')
const updateResourceLimits = (data) => http.put('/api/v1/system/resource-limits', data)
const getAllOIDC = () => http.get('/api/v1/oidc')
const getOIDC = (id) => http.get(`/api/v1/oidc/${id}`)
const updateOIDC = (id, data) =>
  http.put(`/api/v1/oidc/${id}`, data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const deleteOIDC = (id) => http.delete(`/api/v1/oidc/${id}`)
const updateSettings = (key, data) =>
  http.put(`/api/v1/settings/${key}`, data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const getSettings = (key) => http.get(`/api/v1/settings/${key}`)
const login = (data) =>
  http.post(`/api/v1/auth/login`, data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })

const updateUser = (id, data) =>
  http.put(`/api/v1/agents/${id}`, data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const getUsers = () => http.get('/api/v1/agents')
const getUsersCompact = (params) => http.get('/api/v1/agents/compact', { params })

const getUser = (id) => http.get(`/api/v1/agents/${id}`)

const getCurrentUser = () => http.get('/api/v1/agents/me')

const updateCurrentUserAvailability = (data) =>
  http.put('/api/v1/agents/me/availability', data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const resetPassword = (data) =>
  http.post('/api/v1/agents/reset-password', data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const setPassword = (data) =>
  http.post('/api/v1/agents/set-password', data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const deleteUser = (id) => http.delete(`/api/v1/agents/${id}`)
const importAgents = (data) =>
  http.post('/api/v1/agents/import', data, {
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  })
const getAgentImportStatus = () => http.get('/api/v1/agents/import/status')
const createUser = (data) =>
  http.post('/api/v1/agents', data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })

const updateAssigneeLastSeen = (uuid) => http.put(`/api/v1/conversations/${uuid}/last-seen`)
const markConversationAsUnread = (uuid) => http.put(`/api/v1/conversations/${uuid}/mark-unread`)
const getConversationMessage = (cuuid, uuid) =>
  http.get(`/api/v1/conversations/${cuuid}/messages/${uuid}`)
const allowMessageImages = (cuuid, uuid, scope) =>
  http.post(`/api/v1/conversations/${cuuid}/messages/${uuid}/images/allow/${scope}`)
const getResourcePolicy = () => http.get('/api/v1/settings/resource-policy')
const updateResourcePolicy = (data) => http.put('/api/v1/settings/resource-policy', data)
const retryMessage = (cuuid, uuid) =>
  http.put(`/api/v1/conversations/${cuuid}/messages/${uuid}/retry`)
const deleteMessage = (cuuid, uuid) =>
  http.delete(`/api/v1/conversations/${cuuid}/messages/${uuid}`)
const getConversationMessages = (uuid, params) =>
  http.get(`/api/v1/conversations/${uuid}/messages`, { params, abortOnRoute: true })
const sendMessage = (uuid, data) =>
  http.post(`/api/v1/conversations/${uuid}/messages`, data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const getConversation = (uuid) => http.get(`/api/v1/conversations/${uuid}`, { abortOnRoute: true })
const markAddressAsRead = (id) => http.post(`/api/v1/addresses/${id}/mark-read`)
const getConversationTranscript = (uuid) =>
  http.get(`/api/v1/conversations/${uuid}/transcript`, { responseType: 'blob' })

const getAddresses = () => http.get('/api/v1/addresses')
const getAddressConversations = (id, params) =>
  http.get(`/api/v1/addresses/${id}/conversations`, { params, abortOnRoute: true })
const getSidebarCounts = () => http.get('/api/v1/conversations/sidebar-counts')
const getAdminAddresses = () => http.get('/api/v1/admin/addresses')
const getAddressPrincipals = () => http.get('/api/v1/admin/address-principals')
const getAdminAddress = (id) => http.get(`/api/v1/admin/addresses/${id}`)
const getAddressAccess = (id) => http.get(`/api/v1/admin/addresses/${id}/access`)
const createAddress = (data) => http.post('/api/v1/admin/addresses', data)
const updateAddress = (id, data) => http.put(`/api/v1/admin/addresses/${id}`, data)
const deleteAddress = (id) => http.delete(`/api/v1/admin/addresses/${id}`)
// Compose New and the review queue.
const composeEmail = (data) => http.post('/api/v1/compose', data)
const submitReplyForReview = (uuid, data) => http.post(`/api/v1/conversations/${uuid}/reviews`, data)
const getConversationReviews = (uuid) => http.get(`/api/v1/conversations/${uuid}/reviews`)
const getReviews = () => http.get('/api/v1/reviews')
const getReviewCounts = () => http.get('/api/v1/reviews/counts')
const getReview = (uuid) => http.get(`/api/v1/reviews/${uuid}`)
const approveReview = (uuid) => http.post(`/api/v1/reviews/${uuid}/approve`)
const denyReview = (uuid, note) => http.post(`/api/v1/reviews/${uuid}/deny`, { note })
const withdrawReview = (uuid) => http.post(`/api/v1/reviews/${uuid}/withdraw`)
const dismissReview = (uuid) => http.post(`/api/v1/reviews/${uuid}/dismiss`)
const resubmitReview = (uuid, data) => http.put(`/api/v1/reviews/${uuid}`, data)
const discardReview = (uuid) => http.delete(`/api/v1/reviews/${uuid}`)

// Users settings screen.
const getManagedUsers = () => http.get('/api/v1/admin/users')
const createManagedUser = (data) => http.post('/api/v1/admin/users', data)
const updateManagedUserAccess = (id, data) => http.put(`/api/v1/admin/users/${id}/access`, data)
const deleteManagedUser = (id) => http.delete(`/api/v1/admin/users/${id}`)

const uploadMedia = (data) =>
  http.post('/api/v1/media', data, {
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  })
const uploadSiteLogo = (data) =>
  http.post('/api/v1/settings/general/logo', data, {
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  })

const getLanguage = (lang) => http.get(`/api/v1/lang/${lang}`)
const getAvailableLanguages = () => http.get('/api/v1/lang')
const createInbox = (data) =>
  http.post('/api/v1/inboxes', data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const getInboxes = () => http.get('/api/v1/inboxes')
const getInbox = (id) => http.get(`/api/v1/inboxes/${id}`)
const toggleInbox = (id) => http.put(`/api/v1/inboxes/${id}/toggle`)
const updateInbox = (id, data) =>
  http.put(`/api/v1/inboxes/${id}`, data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const deleteInbox = (id) => http.delete(`/api/v1/inboxes/${id}`)
const saveDraft = (uuid, type, data) =>
  http.post(
    `/api/v1/conversations/${uuid}/draft`,
    { ...data, type },
    {
      headers: {
        'Content-Type': 'application/json'
      }
    }
  )

const getAllDrafts = () => http.get('/api/v1/drafts')

const deleteDraft = (uuid, type) =>
  http.delete(`/api/v1/conversations/${uuid}/draft`, { params: { type } })
const getWebhooksCompact = () => http.get('/api/v1/webhooks/compact')
const getWebhooks = () => http.get('/api/v1/webhooks')
const getWebhook = (id) => http.get(`/api/v1/webhooks/${id}`)
const createWebhook = (data) =>
  http.post('/api/v1/webhooks', data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const updateWebhook = (id, data) =>
  http.put(`/api/v1/webhooks/${id}`, data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })
const deleteWebhook = (id) => http.delete(`/api/v1/webhooks/${id}`)
const toggleWebhook = (id) => http.put(`/api/v1/webhooks/${id}/toggle`)
const testWebhook = (id) => http.post(`/api/v1/webhooks/${id}/test`)

const generateAPIKey = (id) =>
  http.post(
    `/api/v1/agents/${id}/api-key`,
    {},
    {
      headers: {
        'Content-Type': 'application/json'
      }
    }
  )

const revokeAPIKey = (id) => http.delete(`/api/v1/agents/${id}/api-key`)

const initiateOAuthFlow = (provider, data) =>
  http.post(`/api/v1/inboxes/oauth/${provider}/authorize`, data, {
    headers: {
      'Content-Type': 'application/json'
    }
  })

export default {
  composeEmail,
  submitReplyForReview,
  getConversationReviews,
  getReviews,
  getReviewCounts,
  getReview,
  approveReview,
  denyReview,
  withdrawReview,
  dismissReview,
  resubmitReview,
  discardReview,
  getManagedUsers,
  createManagedUser,
  updateManagedUserAccess,
  deleteManagedUser,
  markAddressAsRead,
  login,
  deleteUser,
  importAgents,
  getAgentImportStatus,

  resetPassword,
  setPassword,

  getUser,

  getUsers,
  getInbox,
  getInboxes,
  getLanguage,
  getAvailableLanguages,
  getConversation,

  getAddresses,
  getAddressConversations,
  getSidebarCounts,
  getAdminAddresses,
  getAddressPrincipals,
  getAdminAddress,
  getAddressAccess,
  createAddress,
  updateAddress,
  deleteAddress,

  getConversationMessage,
  allowMessageImages,
  getResourcePolicy,
  updateResourcePolicy,
  getConversationMessages,
  getConversationTranscript,
  getCurrentUser,


  uploadMedia,
  uploadSiteLogo,
  updateAssigneeLastSeen,
  markConversationAsUnread,
  updateUser,
  updateCurrentUserAvailability,

  sendMessage,
  retryMessage,
  deleteMessage,
  createUser,
  createInbox,
  updateInbox,
  deleteInbox,
  toggleInbox,

  getSettings,
  updateSettings,
  createOIDC,
  getAllOIDC,
  getConfig,
  getResourceUsage,
  getResourceLimits,
  updateResourceLimits,
  getOIDC,
  updateOIDC,
  deleteOIDC,
  getTemplate,
  getTemplates,
  createTemplate,
  updateTemplate,
  deleteTemplate,



  getUsersCompact,

  saveDraft,
  getAllDrafts,
  deleteDraft,

  getWebhooksCompact,
  getWebhooks,
  getWebhook,
  createWebhook,
  updateWebhook,
  deleteWebhook,
  toggleWebhook,
  testWebhook,

  generateAPIKey,
  revokeAPIKey,
  initiateOAuthFlow
}
