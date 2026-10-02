import { defineStore } from 'pinia'
import { computed, onScopeDispose, reactive, ref, watch, watchEffect } from 'vue'
import { useRouter } from 'vue-router'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { TYPING_RECEIVE_TIMEOUT } from '@shared-ui/composables/useTypingIndicator.js'
import { deepMerge } from '@shared-ui/utils/object.js'
import { computeRecipientsFromMessage } from '@main/utils/email-recipients'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import { subscribeToConversation, sendTypingIndicator, subscribeListReplace } from '@main/websocket'

import MessageCache from '@main/utils/conversation-message-cache'
import { createDraftSync } from '@main/utils/draft-sync'
import { getI18n } from '@main/i18n'
import { CONVERSATION_LIST_TYPE } from '@/constants/conversation'
import { useThrottleFn, useStorage } from '@vueuse/core'

import { delayedLoading } from '@/utils/delayed-loading'
import api from '@main/api'

export const useConversationStore = defineStore('conversation', () => {
  const CONV_LIST_PAGE_SIZE = 25
  const MESSAGE_LIST_PAGE_SIZE = 30

  const currentTo = ref([])
  const currentBCC = ref([])
  const currentCC = ref([])

  const drafts = ref(new Map())
  const draftSaveStates = reactive(new Map())
  const draftSync = createDraftSync({
    write: (uuid, type, draft) => draft
      ? api.saveDraft(uuid, type, draft)
      : api.deleteDraft(uuid, type),
    onState: (key, state) => draftSaveStates.set(key, state)
  })
  onScopeDispose(() => draftSync.dispose())
  const syncDraft = (uuid, type, draft) => draftSync.save(uuid, type, draft)
  const retryDraftSave = (uuid, type) => draftSync.retry(uuid, type)
  // In-memory, resets on reload.
  const selectedDraftType = ref(new Map())
  let resolveDraftsReady
  const draftsReady = new Promise((resolve) => {
    resolveDraftsReady = resolve
  })

  const router = useRouter()
  const isViewingConversation = (uuid) => router.currentRoute.value.params.uuid === uuid

  const sidebarCounts = reactive({
    unread: 0,
    addresses: {}
  })

  // Route changes reuse a count younger than the TTL; mutations and WS events pass force.
  const SIDEBAR_COUNTS_TTL = 45_000
  let sidebarCountsRequest = null
  let sidebarCountsFetchedAt = 0
  let sidebarCountsForceQueued = false

  async function fetchSidebarCounts({ force = false } = {}) {
    if (sidebarCountsRequest) {
      // A request already in flight may have read the DB before this caller's mutation committed.
      if (force) sidebarCountsForceQueued = true
      return sidebarCountsRequest
    }
    if (!force && Date.now() - sidebarCountsFetchedAt < SIDEBAR_COUNTS_TTL) return

    sidebarCountsRequest = (async () => {
      try {
        const resp = await api.getSidebarCounts()
        const data = resp?.data?.data
        if (!data) return
        sidebarCounts.unread = data.unread || 0
        sidebarCounts.addresses = data.addresses || {}
        sidebarCountsFetchedAt = Date.now()
      } catch {
        // The sidebar works without counts.
      } finally {
        sidebarCountsRequest = null
        if (sidebarCountsForceQueued) {
          sidebarCountsForceQueued = false
          fetchSidebarCounts({ force: true })
        }
      }
    })()

    return sidebarCountsRequest
  }

  // WS events burst one per conversation; leading + trailing keeps it to two requests per burst.
  const SIDEBAR_COUNTS_EVENT_THROTTLE = 45_000
  const refreshSidebarCounts = useThrottleFn(
    () => fetchSidebarCounts({ force: true }),
    SIDEBAR_COUNTS_EVENT_THROTTLE,
    true
  )

  // TODO: Move to constants.
  const sortFieldMap = {
    oldest: {
      model: 'conversations',
      field: 'last_message_at',
      order: 'asc'
    },
    newest: {
      model: 'conversations',
      field: 'last_message_at',
      order: 'desc'
    },
    started_first: {
      model: 'conversations',
      field: 'created_at',
      order: 'asc'
    },
    started_last: {
      model: 'conversations',
      field: 'created_at',
      order: 'desc'
    },
    waiting_longest: {
      model: 'conversations',
      field: 'waiting_since',
      order: 'asc'
    }
  }

  const sortFieldI18nKeys = {
    oldest: 'conversation.sort.oldestActivity',
    newest: 'conversation.sort.newestActivity',
    started_first: 'conversation.sort.startedFirst',
    started_last: 'conversation.sort.startedLast',
    waiting_longest: 'conversation.sort.waitingLongest'
  }

  const persistedListSortField = useStorage('conversationListSortField', 'newest')
  if (!sortFieldMap[persistedListSortField.value]) {
    persistedListSortField.value = 'newest'
  }

  const typingByUUID = reactive({})
  const typingAgentsByUUID = reactive({})
  const typingTimeoutsByUUID = new Map()

  const conversations = reactive({
    data: [],
    listType: null,
    sortField: persistedListSortField.value,
    addressID: 0,
    loading: false,
    fetching: false,
    initialized: false,
    page: 1,
    hasMore: false,
    total: 0,
    errorMessage: ''
  })

  const conversation = reactive({
    data: null,
    loading: false,
    errorMessage: '',
    isTyping: false
  })

  const messages = reactive({
    data: new MessageCache(),
    loading: false,
    fetching: false,
    page: 1,
    // To trigger reactivity on the messages cache, simpler than making MessageCache reactive.
    version: 0
  })

  // Convos whose message cache is stale; drained lazily by fetchMessages on next open.
  let staleConversationUUIDs = new Set()
  const CONVERSATION_CACHE_MAX = 50
  const conversationDataCache = new Map()

  function cacheConversationData(uuid, data) {
    if (!uuid || !data) return
    if (conversationDataCache.has(uuid)) conversationDataCache.delete(uuid)
    conversationDataCache.set(uuid, data)
    while (conversationDataCache.size > CONVERSATION_CACHE_MAX) {
      const oldest = conversationDataCache.keys().next().value
      if (oldest === conversation.data?.uuid) break
      conversationDataCache.delete(oldest)
    }
  }
  // Bumped on resetConversations() so in-flight requests can drop stale responses.
  let contextSeq = 0
  const emitter = useEmitter()

  const incrementMessageVersion = () => setTimeout(() => messages.version++, 0)

  function setListSortField(field) {
    if (conversations.sortField === field) return
    conversations.sortField = field
    persistedListSortField.value = field
    resetConversations()
    reFetchConversationsList()
  }

  const getListSortField = computed(() => {
    const i18n = getI18n()
    const t = i18n?.global?.t || ((key) => key.split('.').pop())
    return t(sortFieldI18nKeys[conversations.sortField])
  })

  const conversationsList = computed(() => {
    if (!conversations.data) return []
    return [...conversations.data].sort((a, b) => {
      const field = sortFieldMap[conversations.sortField]?.field
      if (!a[field] && !b[field]) return 0
      if (!a[field]) return 1 // null goes last
      if (!b[field]) return -1
      const order = sortFieldMap[conversations.sortField]?.order
      return order === 'asc'
        ? new Date(a[field]) - new Date(b[field])
        : new Date(b[field]) - new Date(a[field])
    })
  })

  const currentConversationHasMoreMessages = computed(() => {
    return messages.data.hasMore(conversation.data?.uuid)
  })

  const conversationMessages = computed(() => {
    return messages.data.getAllPagesMessages(conversation.data?.uuid)
  })

  function markConversationAsRead(uuid) {
    if (!isViewingConversation(uuid) || document.hidden) return
    const row = conversations.data.find((conv) => conv.uuid === uuid)
    if (row) {
      const unread = row.unread_message_count || 0
      sidebarCounts.unread = Math.max(0, sidebarCounts.unread - unread)
      adjustAddressUnread(row.address_id, -unread)
      row.unread_message_count = 0
    }
  }

  function adjustAddressUnread(addressID, delta) {
    if (!addressID || !delta) return
    const current = sidebarCounts.addresses[addressID] || 0
    const next = Math.max(0, current + delta)
    if (next) sidebarCounts.addresses[addressID] = next
    else delete sidebarCounts.addresses[addressID]
  }

  async function markAsUnread(uuid) {
    try {
      await api.markConversationAsUnread(uuid)
      const index = conversations.data.findIndex((conv) => conv.uuid === uuid)
      if (index !== -1) {
        const row = conversations.data[index]
        const previousUnread = row.unread_message_count || 0
        sidebarCounts.unread = Math.max(0, sidebarCounts.unread - previousUnread) + 1
        adjustAddressUnread(row.address_id, 1 - previousUnread)
        conversations.data[index].unread_message_count = 1
      }
      fetchSidebarCounts({ force: true })
    } catch (err) {
      handleHTTPError(err)
    }
  }

  function incrementUnread(uuid) {
    const row = conversations.data.find((c) => c.uuid === uuid)
    if (!row) return
    row.unread_message_count = Math.min((row.unread_message_count || 0) + 1, 10)
    adjustAddressUnread(row.address_id, 1)
  }

  function incrementAddressUnread(addressID) {
    adjustAddressUnread(addressID, 1)
  }

  const currentSenderName = computed(() => {
    if (!conversation.data?.correspondent) return ''
    const sender = conversation.data.correspondent
    return `${sender.first_name || ''} ${sender.last_name || ''}`.trim() || sender.email || ''
  })

  function getContactFullName(uuid) {
    if (conversations?.data) {
      const conv = conversations.data.find((conv) => conv.uuid === uuid)
      return conv
        ? `${conv.correspondent.first_name || ''} ${conv.correspondent.last_name || ''}`.trim() ||
            conv.correspondent.email
        : ''
    }
  }

  const current = computed(() => {
    return conversation.data || {}
  })

  const isConversationOpen = computed(() => {
    return Object.keys(conversation.data || {}).length > 0
  })

  watchEffect(async () => {
    const _ = messages.version // eslint-disable-line no-unused-vars
    const conv = conversation.data
    const msgData = messages.data
    const inboxEmail = conv?.inbox_mail

    if (!conv || !msgData || !inboxEmail) return

    // Use the last human message to prefill recipients.
    const latestMessage = msgData.getLatestMessage(conv.uuid, ['incoming', 'outgoing'], true, true)
    if (!latestMessage) {
      currentTo.value = []
      currentCC.value = []
      currentBCC.value = []
      return
    }

    const { to, cc, bcc } = computeRecipientsFromMessage(
      latestMessage,
      conv.correspondent?.email || '',
      inboxEmail,
      conv?.inbox_reply_to || ''
    )
    currentTo.value = to
    currentCC.value = cc
    currentBCC.value = bcc
  })

  function resetTypingState() {
    conversation.isTyping = Boolean(typingByUUID[conversation.data?.uuid])
  }

  let conversationRequestSeq = 0
  async function fetchConversation(uuid) {
    const requestSeq = ++conversationRequestSeq
    const cached = conversationDataCache.get(uuid)
    if (cached) {
      conversation.data = cached
      resetTypingState()
      subscribeToConversation(uuid)
      silentRefetchConversation(uuid)
      return
    }
    const guard = delayedLoading(conversation, 'loading')
    try {
      const resp = await api.getConversation(uuid)
      if (requestSeq !== conversationRequestSeq) return
      conversation.data = resp.data.data
      conversation.errorMessage = ''
      resetTypingState()
      subscribeToConversation(uuid)
      cacheConversationData(uuid, conversation.data)
    } catch (error) {
      if (requestSeq !== conversationRequestSeq || error.code === 'ERR_CANCELED') return
      conversation.data = null
      conversation.errorMessage = handleHTTPError(error).message
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: conversation.errorMessage
      })
    } finally {
      guard.release()
    }
  }

  async function silentRefetchConversation(uuid) {
    try {
      const resp = await api.getConversation(uuid)
      if (conversation.data?.uuid === uuid) {
        deepMerge(conversation.data, resp.data.data)
        cacheConversationData(uuid, conversation.data)
      }
    } catch (error) {
      if ([403, 404].includes(error.response?.status)) {
        conversationDataCache.delete(uuid)
        messages.data.purgeConversation(uuid)
        if (conversation.data?.uuid === uuid) conversation.data = null
        incrementMessageVersion()
      }
      console.warn('silent conversation refetch failed', error)
    }
  }

  function invalidateImageDisplays() {
    messages.data = new MessageCache()
    staleConversationUUIDs.clear()
    incrementMessageVersion()
  }

  async function updateImagePermissions(message, scope) {
    if (scope === 'message') {
      mergeMessageUpdate(message)
      return
    }
    invalidateImageDisplays()
    await fetchMessages(message.conversation_uuid)
  }

  async function fetchMessages(uuid, fetchNextPage = false) {
    if (staleConversationUUIDs.has(uuid)) {
      try {
        const response = await api.getConversationMessages(uuid, {
          page: 1,
          page_size: MESSAGE_LIST_PAGE_SIZE
        })
        const result = response.data?.data || {}
        // A missed burst can exceed one page. Replace pagination as well as
        // content, otherwise the middle of the thread can stay missing forever.
        messages.data.purgeConversation(uuid)
        messages.data.addMessages(uuid, result.results || [], result.page, result.total_pages)
        staleConversationUUIDs.delete(uuid)
        incrementMessageVersion()
        markConversationAsRead(uuid)
        return true
      } catch (error) {
        if (error.code === 'ERR_CANCELED') return false
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
          variant: 'destructive',
          description: handleHTTPError(error).message
        })
        return false
      }
    }

    if (!fetchNextPage && messages.data.getAllPagesMessages(uuid).length > 0) {
      markConversationAsRead(uuid)
      return true
    }

    const guard = fetchNextPage ? null : delayedLoading(messages, 'loading')
    messages.fetching = true
    const page = messages.data.getLastFetchedPage(uuid) + 1
    try {
      const response = await api.getConversationMessages(uuid, {
        page,
        page_size: MESSAGE_LIST_PAGE_SIZE
      })
      const result = response.data?.data || {}
      markConversationAsRead(uuid)
      messages.data.addMessages(uuid, result.results || [], result.page, result.total_pages)
      incrementMessageVersion()
      return true
    } catch (error) {
      if (error.code === 'ERR_CANCELED') return false
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
      return false
    } finally {
      if (guard) guard.release()
      messages.fetching = false
    }
  }

  async function fetchNextMessages() {
    return fetchMessages(conversation.data.uuid, true)
  }

  async function fetchMessage(conversationUUID, messageUUID) {
    try {
      const response = await api.getConversationMessage(conversationUUID, messageUUID)
      if (response?.data?.data) {
        const newMsg = response.data.data
        if (messages.data.hasMessage(conversationUUID, newMsg.uuid)) {
          messages.data.updateMessage(conversationUUID, newMsg.uuid, newMsg)
        } else {
          messages.data.addMessage(conversationUUID, newMsg)
        }
        incrementMessageVersion()
        return newMsg
      }
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  async function deleteMessage(conversationUUID, messageUUID) {
    try {
      const resp = await api.deleteMessage(conversationUUID, messageUUID)
      const deletedText = resp.data.data.content
      const existing = messages.data
        .getAllPagesMessages(conversationUUID)
        .find((m) => m.uuid === messageUUID)
      messages.data.updateMessage(conversationUUID, messageUUID, {
        content: deletedText,
        text_content: deletedText,
        meta: { ...(existing?.meta || {}), deleted_at: new Date().toISOString() }
      })
      incrementMessageVersion()
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  function fetchNextConversations() {
    if (conversations.fetching || !conversations.hasMore) return
    fetchConversationsList(false, conversations.addressID, conversations.page + 1)
  }

  function reFetchConversationsList(showLoader = true) {
    fetchConversationsList(showLoader, conversations.addressID, conversations.page)
  }

  async function fetchFirstPageConversations() {
    await fetchConversationsList(false, conversations.addressID, 1)
  }

  async function fetchConversationsList(showLoader = true, addressID = 0, page = 0, replace = false) {
    if (!addressID) return
    if (conversations.listType !== CONVERSATION_LIST_TYPE.ADDRESS || conversations.addressID !== addressID) {
      resetConversations()
    }
    conversations.listType = CONVERSATION_LIST_TYPE.ADDRESS
    conversations.addressID = addressID
    const guard = showLoader ? delayedLoading(conversations, 'loading') : null
    conversations.fetching = true
    if (page === 0) page = conversations.page
    const seq = contextSeq
    const isStale = () => seq !== contextSeq
    try {
      conversations.errorMessage = ''
      const response = await makeConversationListRequest(addressID, page)
      if (isStale()) return
      if (replace) {
        conversations.data = []
        conversations.page = 1
      }
      processConversationListResponse(response)
    } catch (error) {
      if (isStale()) return
      if (conversations.data.length === 0) {
        conversations.errorMessage = handleHTTPError(error).message
        conversations.total = 0
      }
    } finally {
      if (guard) {
        if (isStale()) guard.cancel()
        else guard.release()
      }
      if (!isStale()) {
        conversations.initialized = true
        conversations.fetching = false
      }
    }
  }

  async function makeConversationListRequest(addressID, page) {
    return api.getAddressConversations(addressID, {
      page,
      page_size: CONV_LIST_PAGE_SIZE,
      order_by:
        sortFieldMap[conversations.sortField].model +
        '.' +
        sortFieldMap[conversations.sortField].field,
      order: sortFieldMap[conversations.sortField].order
    })
  }

  function trimListToCurrentPage() {
    const maxLen = conversations.page * CONV_LIST_PAGE_SIZE
    if (conversations.data.length > maxLen) {
      conversations.data.splice(maxLen)
    }
  }

  function mergeIntoList(uuid, payload) {
    const existing = conversations.data?.find((c) => c.uuid === uuid)
    if (existing) deepMerge(existing, payload)
    return existing
  }

  function processConversationListResponse(response) {
    const apiResponse = response.data.data
    const newConversations = []
    for (const conv of apiResponse.results) {
      if (!mergeIntoList(conv.uuid, conv)) newConversations.push(conv)
    }
    conversations.page = Math.max(conversations.page, apiResponse.page)
    conversations.hasMore = apiResponse.total_pages > conversations.page
    if (!conversations.data) conversations.data = []
    if (apiResponse.page === 1) {
      conversations.data.unshift(...newConversations)
    } else {
      conversations.data.push(...newConversations)
    }
    conversations.total = apiResponse.total

    trimListToCurrentPage()
  }

  let resyncRequest = null
  function resyncMail() {
    if (resyncRequest) return resyncRequest
    for (const uuid of messages.data.cache.keys()) staleConversationUUIDs.add(uuid)
    retryDraftSave()
    const uuid = router.currentRoute.value.params.uuid
    const addressID = Number(router.currentRoute.value.params.addressID)
    const requests = [fetchSidebarCounts({ force: true })]
    if (addressID) requests.push(fetchConversationsList(false, addressID, 1, true))
    if (uuid) requests.push(silentRefetchConversation(uuid), fetchMessages(uuid).then((loaded) => {
      if (loaded && !document.hidden) return updateAssigneeLastSeen(uuid)
    }))
    resyncRequest = Promise.allSettled(requests).finally(() => { resyncRequest = null })
    return resyncRequest
  }

  async function markAddressAsRead(addressID) {
    const response = await api.markAddressAsRead(addressID)
    const markedAt = response.data.data.marked_at
    for (const row of conversations.data) {
      if (Number(row.address_id) === Number(addressID) &&
          (!row.last_message_at || new Date(row.last_message_at) <= new Date(markedAt))) {
        row.unread_message_count = 0
      }
    }
    const requests = [fetchSidebarCounts({ force: true })]
    if (Number(conversations.addressID) === Number(addressID)) {
      requests.push(fetchConversationsList(false, Number(addressID), 1, true))
    }
    if (Number(conversation.data?.address_id) === Number(addressID)) {
      requests.push(silentRefetchConversation(conversation.data.uuid))
    }
    await Promise.allSettled(requests)
  }

  async function updateAssigneeLastSeen(uuid) {
    if (!isViewingConversation(uuid)) return
    markConversationAsRead(uuid)
    try {
      await api.updateAssigneeLastSeen(uuid)
      // Reconcile the total, including threads outside the visible page and counts capped at 9+.
      fetchSidebarCounts({ force: true })
    } catch {
      // Leave read-state retries to the next visit or focus event.
    }
  }

  function isConversationInList(uuid) {
    return Boolean(conversations.data?.find((c) => c.uuid === uuid))
  }

  // trailing=true: fires one final refresh after a burst so the list converges to latest state.
  const throttledFetchFirstPage = useThrottleFn(fetchFirstPageConversations, 60000, true)

  function refreshConversationList() {
    throttledFetchFirstPage()
  }

  function updateConversationLastMessage(uuid, message) {
    const conv = conversations.data?.find((c) => c.uuid === uuid)
    if (!conv) return
    conv.last_message =
      message.text_content || message.content || getMediaPreview(message.attachments)
    conv.last_message_at = message.created_at
    conv.last_message_sender = message.sender_type
  }

  async function updateConversationMessage(message) {
    if (conversation.data?.uuid !== message.conversation_uuid) {
      // Lazy invalidation: refresh the cache when the user next opens this convo, not on every WS event.
      if (messages.data.hasConversation(message.conversation_uuid)) {
        staleConversationUUIDs.add(message.conversation_uuid)
      }
      return
    }

    if (!messages.data.hasMessage(message.conversation_uuid, message.uuid)) {
      const echoId = message.echo_id
      if (echoId && messages.data.hasMessage(message.conversation_uuid, echoId)) {
        messages.data.updateMessage(message.conversation_uuid, echoId, { uuid: message.uuid })
        incrementMessageVersion()
        updateAssigneeLastSeen(message.conversation_uuid)
        return
      }

      if (message.type === 'activity') {
        const activityMessage = {
          uuid: message.uuid,
          conversation_uuid: message.conversation_uuid,
          type: 'activity',
          content: message.preview,
          created_at: message.created_at,
          updated_at: message.created_at,
          sender_type: message.sender_type
        }
        messages.data.addMessage(message.conversation_uuid, activityMessage)
        incrementMessageVersion()
        setTimeout(() => {
          emitter.emit(EMITTER_EVENTS.NEW_MESSAGE, {
            conversation_uuid: message.conversation_uuid,
            message: activityMessage
          })
        }, 100)
        if (!document.hidden) {
          updateAssigneeLastSeen(message.conversation_uuid)
        }
        return
      }

      const fetchedMessage = await fetchMessage(message.conversation_uuid, message.uuid)
      if (fetchedMessage) {
        updateConversationLastMessage(message.conversation_uuid, fetchedMessage)
        setTimeout(() => {
          emitter.emit(EMITTER_EVENTS.NEW_MESSAGE, {
            conversation_uuid: message.conversation_uuid,
            message: fetchedMessage
          })
        }, 100)
      }

      if (!document.hidden) {
        updateAssigneeLastSeen(message.conversation_uuid)
      }
    }
  }

  function addPendingMessage(
    conversationUUID,
    content,
    isPrivate,
    author,
    attachments = [],
    textContent = '',
    meta = {}
  ) {
    const pendingMessage = {
      uuid: `pending-${Date.now()}`,
      type: 'outgoing',
      status: 'pending',
      content,
      text_content: textContent,
      content_type: 'html',
      private: isPrivate,
      sender_type: 'agent',
      sender_id: author.id,
      conversation_uuid: conversationUUID,
      created_at: new Date().toISOString(),
      author,
      attachments: attachments.map((a) => ({
        uuid: a.uuid,
        name: a.filename || a.name,
        size: a.size,
        content_type: a.content_type,
        url: a.url,
        disposition: a.disposition
      })),
      meta
    }
    messages.data.addMessage(conversationUUID, pendingMessage)
    incrementMessageVersion()
    setTimeout(() => {
      emitter.emit(EMITTER_EVENTS.NEW_MESSAGE, {
        conversation_uuid: conversationUUID,
        message: pendingMessage
      })
    }, 0)

    return pendingMessage.uuid
  }

  function replacePendingMessage(conversationUUID, tempUUID, realMessage) {
    if (messages.data.hasMessage(conversationUUID, realMessage.uuid)) {
      messages.data.removeMessage(conversationUUID, tempUUID)
    } else if (messages.data.hasMessage(conversationUUID, tempUUID)) {
      messages.data.updateMessage(conversationUUID, tempUUID, realMessage)
    } else {
      messages.data.addMessage(conversationUUID, realMessage)
    }
    incrementMessageVersion()
  }

  function removePendingMessage(conversationUUID, tempUUID) {
    messages.data.removeMessage(conversationUUID, tempUUID)
    incrementMessageVersion()
  }

  function addNewConversation(conversation) {
    if (!isConversationInList(conversation.uuid)) {
      refreshConversationList()
    }
  }

  function mergeMessageUpdate(data) {
    const { conversation_uuid, uuid, ...fields } = data
    if (!messages.data.hasMessage(conversation_uuid, uuid)) return
    messages.data.updateMessage(conversation_uuid, uuid, fields)
    incrementMessageVersion()
  }

  function canPushInsert(payload) {
    return (
      conversations.listType === CONVERSATION_LIST_TYPE.ADDRESS &&
      Number(payload.address_id) === Number(conversations.addressID)
    )
  }

  function handleConvPush(payload) {
    if (!payload || !payload.uuid) return
    if (mergeIntoList(payload.uuid, payload)) {
      if (conversation.data?.uuid === payload.uuid) {
        deepMerge(conversation.data, payload)
      }
      return
    }
    if (!canPushInsert(payload)) return
    if (!conversations.data) conversations.data = []
    conversations.data.unshift(payload)
    conversations.total += 1
    trimListToCurrentPage()
  }

  function mergeConversationUpdate(update) {
    if (conversation.data?.uuid === update.uuid) {
      deepMerge(conversation.data, update)
    }
    mergeIntoList(update.uuid, update)
  }

  function resetConversations() {
    conversations.data = []
    conversations.page = 1
    conversations.initialized = false
    conversations.hasMore = false
    conversations.total = 0
    contextSeq++

  }

  function updateTypingStatus(typingData) {
    const { conversation_uuid: uuid, is_typing, user_id: userID, user_name: name } = typingData
    const actor = String(userID || 'unknown')
    const key = `${uuid}::${actor}`
    const previous = typingTimeoutsByUUID.get(key)
    if (previous) clearTimeout(previous)
    const clearActor = () => {
      if (typingAgentsByUUID[uuid]) {
        delete typingAgentsByUUID[uuid][actor]
        if (!Object.keys(typingAgentsByUUID[uuid]).length) {
          delete typingAgentsByUUID[uuid]
          delete typingByUUID[uuid]
        }
      }
      typingTimeoutsByUUID.delete(key)
      if (conversation.data?.uuid === uuid) resetTypingState()
    }
    if (is_typing) {
      if (!typingAgentsByUUID[uuid]) typingAgentsByUUID[uuid] = {}
      typingAgentsByUUID[uuid][actor] = name || ''
      typingByUUID[uuid] = true
      typingTimeoutsByUUID.set(key, setTimeout(clearActor, TYPING_RECEIVE_TIMEOUT))
      if (conversation.data?.uuid === uuid) resetTypingState()
    } else clearActor()
  }

  const typingNames = (uuid) => Object.values(typingAgentsByUUID[uuid] || {}).filter(Boolean).join(', ')
  onScopeDispose(() => { for (const timer of typingTimeoutsByUUID.values()) clearTimeout(timer) })

  function sendTyping(isTyping, otherAttributes = {}) {
    if (conversation.data?.uuid) {
      sendTypingIndicator(conversation.data.uuid, isTyping, otherAttributes.isPrivateMessage)
    }
  }

  function draftMapKey(uuid, type) {
    return `${uuid}::${type}`
  }

  async function fetchAllDrafts() {
    try {
      const resp = await api.getAllDrafts()
      const newDrafts = new Map()
      if (resp.data?.data) {
        for (const draft of resp.data.data) {
          newDrafts.set(draftMapKey(draft.conversation_uuid, draft.type), draft)
        }
      }
      // Do not overwrite edits made while the initial fetch was in flight.
      for (const [key, draft] of drafts.value) newDrafts.set(key, draft)
      drafts.value = newDrafts
    } catch (e) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(e).message
      })
    } finally {
      resolveDraftsReady()
    }
  }

  function getDraft(uuid, type) {
    return drafts.value.get(draftMapKey(uuid, type))
  }

  function setDraft(uuid, type, draft) {
    drafts.value.set(draftMapKey(uuid, type), draft)
    drafts.value = new Map(drafts.value)
  }

  function removeDraft(uuid, type) {
    drafts.value.delete(draftMapKey(uuid, type))
    drafts.value = new Map(drafts.value)
  }

  function hasDraft(uuid, type) {
    return drafts.value.has(draftMapKey(uuid, type))
  }

  function conversationHasDraft(uuid) {
    return hasDraft(uuid, 'reply') || hasDraft(uuid, 'private_note')
  }

  function setSelectedDraftType(uuid, type) {
    selectedDraftType.value.set(uuid, type)
    selectedDraftType.value = new Map(selectedDraftType.value)
  }

  function resolveDraftType(uuid) {
    const last = selectedDraftType.value.get(uuid)
    if (last && hasDraft(uuid, last)) return last
    if (hasDraft(uuid, 'reply')) return 'reply'
    if (hasDraft(uuid, 'private_note')) return 'private_note'
    return last || 'reply'
  }

  function conversationDraftPreview(uuid) {
    return getDraft(uuid, resolveDraftType(uuid))
  }

  function getMediaPreview(attachments) {
    if (!attachments?.length) return ''
    const contentType = attachments[0].content_type || ''
    const i18n = getI18n()
    const t = i18n?.global?.t || ((key) => key.split('.').pop())

    if (contentType.startsWith('image/')) return t('globals.terms.image')
    if (contentType.startsWith('video/')) return t('globals.terms.video')
    if (contentType.startsWith('audio/')) return t('globals.terms.audio')
    return t('globals.terms.file')
  }

  // On new conversation uuids, subscribere user to those conversations.
  watch(
    () =>
      conversations.data
        ?.map((c) => c.uuid)
        .sort()
        .join(',') ?? '',
    () => subscribeListReplace(conversations.data?.map((c) => c.uuid) || [])
  )

  return {
    isViewingConversation,
    typingNames,
    resyncMail,
    markAddressAsRead,
    draftSaveStates,
    syncDraft,
    retryDraftSave,
    conversations,
    conversation,
    messages,
    conversationsList,
    conversationMessages,
    currentConversationHasMoreMessages,
    isConversationOpen,
    current,
    currentSenderName,
    currentTo,
    currentBCC,
    currentCC,
    isConversationInList,

    mergeConversationUpdate,
    handleConvPush,

    addNewConversation,
    getContactFullName,
    fetchNextMessages,
    fetchNextConversations,
    mergeMessageUpdate,
    invalidateImageDisplays,
    updateImagePermissions,
    updateAssigneeLastSeen,
    markAsUnread,
    incrementUnread,
    incrementAddressUnread,
    updateConversationMessage,
    fetchConversation,
    fetchConversationsList,
    fetchMessages,

    refreshConversationList,
    resetConversations,
    updateConversationLastMessage,
    fetchFirstPageConversations,

    setListSortField,

    getListSortField,
    sortFieldI18nKeys,

    updateTypingStatus,
    typingByUUID,
    sendTyping,
    drafts,
    draftsReady,
    fetchAllDrafts,
    getDraft,
    setDraft,
    removeDraft,
    hasDraft,
    deleteMessage,
    conversationHasDraft,
    conversationDraftPreview,
    getMediaPreview,
    setSelectedDraftType,
    resolveDraftType,
    addPendingMessage,
    replacePendingMessage,
    removePendingMessage,
    sidebarCounts,
    fetchSidebarCounts,
    refreshSidebarCounts
  }
})
