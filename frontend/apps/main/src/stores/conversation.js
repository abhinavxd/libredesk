import { defineStore } from 'pinia'
import { computed, reactive, ref, watch, watchEffect } from 'vue'
import { useRouter } from 'vue-router'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { TYPING_RECEIVE_TIMEOUT } from '@shared-ui/composables/useTypingIndicator.js'
import { deepMerge } from '@shared-ui/utils/object.js'
import { computeRecipientsFromMessage } from '../utils/email-recipients'
import { useEmitter } from '../composables/useEmitter'
import { EMITTER_EVENTS } from '../constants/emitterEvents'
import { subscribeToConversation, sendTypingIndicator, subscribeListReplace } from '@main/websocket'
import { playNotificationSound } from '@shared-ui/composables/useNotificationSound'
import MessageCache from '../utils/conversation-message-cache'
import { getI18n } from '../i18n'
import { CONVERSATION_LIST_TYPE, CONVERSATION_DEFAULT_STATUSES, TAG_ACTION } from '@/constants/conversation'
import { useThrottleFn } from '@vueuse/core'
import { useUserStore } from '@/stores/user'
import { delayedLoading } from '@/utils/delayed-loading'
import api from '../api'

export const useConversationStore = defineStore('conversation', () => {
  const CONV_LIST_PAGE_SIZE = 25
  const MESSAGE_LIST_PAGE_SIZE = 30
  const priorities = ref([])
  const statuses = ref([])
  const currentTo = ref([])
  const currentBCC = ref([])
  const currentCC = ref([])
  const macros = ref({})
  const drafts = ref(new Map())
  const userStore = useUserStore()
  const router = useRouter()
  const isViewingConversation = (uuid) => router.currentRoute.value.params.uuid === uuid

  const selectedUUIDs = ref(new Set())

  const priorityOptions = computed(() => {
    return priorities.value.map(p => ({ label: p.name, value: p.id }))
  })
  const statusOptions = computed(() => {
    return statuses.value.map(s => ({ label: s.name, value: s.id }))
  })
  const statusOptionsNoSnooze = computed(() =>
    statuses.value.filter(s => s.name !== CONVERSATION_DEFAULT_STATUSES.SNOOZED).map(s => ({
      label: s.name,
      value: s.id
    }))
  )

  let lastClickedUUID = null

  const selectedCount = computed(() => selectedUUIDs.value.size)
  const allSelected = computed(() => {
    const list = conversationsList.value
    return list.length > 0 && selectedUUIDs.value.size === list.length
  })

  function toggleSelect (uuid, shiftKey = false) {
    const next = new Set(selectedUUIDs.value)

    if (shiftKey && lastClickedUUID && lastClickedUUID !== uuid) {
      const list = conversationsList.value
      const lastIdx = list.findIndex(c => c.uuid === lastClickedUUID)
      const curIdx = list.findIndex(c => c.uuid === uuid)
      if (lastIdx !== -1 && curIdx !== -1) {
        const start = Math.min(lastIdx, curIdx)
        const end = Math.max(lastIdx, curIdx)
        for (let i = start; i <= end; i++) {
          next.add(list[i].uuid)
        }
      }
    } else {
      if (next.has(uuid)) next.delete(uuid)
      else next.add(uuid)
    }

    lastClickedUUID = uuid
    selectedUUIDs.value = next
  }

  function selectAll () {
    selectedUUIDs.value = new Set(conversationsList.value.map(c => c.uuid))
  }

  function clearSelection () {
    selectedUUIDs.value = new Set()
    lastClickedUUID = null
  }

  function isSelected (uuid) {
    return selectedUUIDs.value.has(uuid)
  }

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
    },
    next_sla_target: {
      model: 'conversations',
      field: 'next_sla_deadline_at',
      order: 'asc'
    },
    priority_first: {
      model: 'conversations',
      field: 'priority_id',
      order: 'desc'
    }
  }

  const sortFieldI18nKeys = {
    oldest: 'conversation.sort.oldestActivity',
    newest: 'conversation.sort.newestActivity',
    started_first: 'conversation.sort.startedFirst',
    started_last: 'conversation.sort.startedLast',
    waiting_longest: 'conversation.sort.waitingLongest',
    next_sla_target: 'conversation.sort.nextSLATarget',
    priority_first: 'conversation.sort.priorityFirst'
  }

  let typingTimeout = null
  const typingByUUID = reactive({})
  const typingTimeoutsByUUID = new Map()

  const conversations = reactive({
    data: [],
    listType: null,
    status: 'Open',
    sortField: 'newest',
    listFilters: [],
    viewID: 0,
    teamID: 0,
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
    version: 0,
  })

  // Convos whose message cache is stale; drained lazily by fetchMessages on next open.
  let staleConversationUUIDs = new Set()
  const CONVERSATION_CACHE_MAX = 50
  const conversationDataCache = new Map()

  function cacheConversationData (uuid, data) {
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

  function setListStatus (status, fetch = true) {
    conversations.status = status
    if (fetch) {
      resetConversations()
      reFetchConversationsList()
    }
  }

  const getListStatus = computed(() => {
    return conversations.status
  })

  function setListSortField (field) {
    if (conversations.sortField === field) return
    conversations.sortField = field
    resetConversations()
    reFetchConversationsList()
  }

  const getListSortField = computed(() => {
    const i18n = getI18n()
    const t = i18n?.global?.t || ((key) => key.split('.').pop())
    return t(sortFieldI18nKeys[conversations.sortField])
  })


  async function fetchStatuses () {
    if (statuses.value.length > 0) return
    try {
      const response = await api.getStatuses()
      statuses.value = response.data.data.map(status => ({
        ...status,
        id: status.id.toString()
      }))
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  async function fetchPriorities () {
    if (priorities.value.length > 0) return
    try {
      const response = await api.getPriorities()
      priorities.value = response.data.data.map(priority => ({
        ...priority,
        id: priority.id.toString()
      }))
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  function matchesAssignmentScope (conv) {
    switch (conversations.listType) {
      case CONVERSATION_LIST_TYPE.ASSIGNED:
        return conv.assigned_user_id === userStore.userID
      case CONVERSATION_LIST_TYPE.UNASSIGNED:
        return !conv.assigned_user_id && !conv.assigned_team_id
      case CONVERSATION_LIST_TYPE.TEAM_UNASSIGNED:
        return Number(conv.assigned_team_id) === Number(conversations.teamID) && !conv.assigned_user_id
      default:
        return null
    }
  }

  function belongsToList (conv) {
    const matched = matchesAssignmentScope(conv)
    return matched === null ? true : matched
  }

  const conversationsList = computed(() => {
    if (!conversations.data) return []
    let filteredConversations = conversations.data
    if (conversations.status !== "") {
      filteredConversations = filteredConversations.filter(conv => conv.status === conversations.status)
    }
    filteredConversations = filteredConversations.filter(belongsToList)

    return [...filteredConversations].sort((a, b) => {
      const field = sortFieldMap[conversations.sortField]?.field
      if (!a[field] && !b[field]) return 0
      if (!a[field]) return 1       // null goes last
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

  function markConversationAsRead (uuid) {
    const index = conversations.data.findIndex(conv => conv.uuid === uuid)
    if (index !== -1) {
      setTimeout(() => {
        if (conversations.data?.[index]) {
          conversations.data[index].unread_message_count = 0
        }
      }, 3000)
    }
  }

  async function markAsUnread (uuid) {
    try {
      await api.markConversationAsUnread(uuid)
      const index = conversations.data.findIndex(conv => conv.uuid === uuid)
      if (index !== -1) {
        conversations.data[index].unread_message_count = 1
      }
    } catch (err) {
      handleHTTPError(err)
    }
  }

  function incrementUnread (uuid) {
    const row = conversations.data.find(c => c.uuid === uuid)
    if (!row) return
    row.unread_message_count = Math.min((row.unread_message_count || 0) + 1, 10)
  }

  const currentContactName = computed(() => {
    if (!conversation.data?.contact) return ''
    return conversation.data?.contact.first_name + ' ' + conversation.data?.contact.last_name
  })

  function getContactFullName (uuid) {
    if (conversations?.data) {
      const conv = conversations.data.find(conv => conv.uuid === uuid)
      return conv ? `${conv.contact.first_name} ${conv.contact.last_name}` : ''
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

    if (conv?.inbox_channel === 'livechat') {
      currentTo.value = []
      currentCC.value = []
      currentBCC.value = []
      return
    }

    if (!conv || !msgData || !inboxEmail) return

    // Skip automated messages (auto-replies, CSAT) so the prefill reflects the last human-driven recipients.
    const latestMessage = msgData.getLatestMessage(conv.uuid, ['incoming', 'outgoing'], true, true)
    if (!latestMessage) {
      currentTo.value = []
      currentCC.value = []
      currentBCC.value = []
      return
    }

    const { to, cc, bcc } = computeRecipientsFromMessage(
      latestMessage,
      conv.contact?.email || '',
      inboxEmail,
      conv?.inbox_reply_to || ''
    )
    currentTo.value = to
    currentCC.value = cc
    currentBCC.value = bcc
  })

  function resetTypingState () {
    conversation.isTyping = false
    if (typingTimeout) {
      clearTimeout(typingTimeout)
      typingTimeout = null
    }
  }

  async function fetchConversation (uuid) {
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
      conversation.data = resp.data.data
      resetTypingState()
      subscribeToConversation(uuid)
      cacheConversationData(uuid, conversation.data)
    } catch (error) {
      conversation.errorMessage = handleHTTPError(error).message
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: conversation.errorMessage
      })
    } finally {
      guard.release()
    }
  }

  async function silentRefetchConversation (uuid) {
    try {
      const resp = await api.getConversation(uuid)
      if (conversation.data?.uuid === uuid) {
        deepMerge(conversation.data, resp.data.data)
        cacheConversationData(uuid, conversation.data)
      }
    } catch (error) {
      console.warn('silent conversation refetch failed', error)
    }
  }

  async function fetchMessages (uuid, fetchNextPage = false) {
    if (staleConversationUUIDs.has(uuid) && messages.data.hasConversation(uuid)) {
      try {
        const response = await api.getConversationMessages(uuid, { page: 1, page_size: MESSAGE_LIST_PAGE_SIZE })
        const newMessages = response.data?.data?.results || []
        let lastAdded = null
        for (const m of newMessages) {
          if (!messages.data.hasMessage(uuid, m.uuid)) {
            messages.data.addMessage(uuid, m)
            lastAdded = m
          }
        }
        staleConversationUUIDs.delete(uuid)
        if (lastAdded) {
          incrementMessageVersion()
          setTimeout(() => {
            emitter.emit(EMITTER_EVENTS.NEW_MESSAGE, { conversation_uuid: uuid, message: lastAdded })
          }, 100)
        }
      } catch (error) {
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
          variant: 'destructive',
          description: handleHTTPError(error).message
        })
      }
    }

    if (!fetchNextPage && messages.data.getAllPagesMessages(uuid).length > 0) {
      markConversationAsRead(uuid)
      return
    }

    const guard = fetchNextPage ? null : delayedLoading(messages, 'loading')
    messages.fetching = true
    const page = messages.data.getLastFetchedPage(uuid) + 1
    try {
      const response = await api.getConversationMessages(uuid, { page, page_size: MESSAGE_LIST_PAGE_SIZE })
      const result = response.data?.data || {}
      markConversationAsRead(uuid)
      messages.data.addMessages(uuid, result.results || [], result.page, result.total_pages)
      incrementMessageVersion()
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    } finally {
      if (guard) guard.release()
      messages.fetching = false
    }
  }

  async function fetchNextMessages () {
    return fetchMessages(conversation.data.uuid, true)
  }

  async function fetchMessage (conversationUUID, messageUUID) {
    try {
      const response = await api.getConversationMessage(conversationUUID, messageUUID)
      if (response?.data?.data) {
        const newMsg = response.data.data
        messages.data.addMessage(conversationUUID, newMsg)
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

  function fetchNextConversations () {
    if (conversations.fetching || !conversations.hasMore) return
    fetchConversationsList(false, conversations.listType, conversations.teamID, conversations.listFilters, conversations.viewID, conversations.page + 1)
  }

  function reFetchConversationsList (showLoader = true) {
    fetchConversationsList(showLoader, conversations.listType, conversations.teamID, conversations.listFilters, conversations.viewID, conversations.page)
  }

  async function fetchFirstPageConversations () {
    await fetchConversationsList(false, conversations.listType, conversations.teamID, conversations.listFilters, conversations.viewID, 1)
  }

  async function fetchConversationsList (showLoader = true, listType = null, teamID = 0, filters = [], viewID = 0, page = 0) {
    if (!listType) return
    if (conversations.listType !== listType || conversations.teamID !== teamID || conversations.viewID !== viewID) {
      resetConversations()
    }
    conversations.listType = listType
    if (teamID) conversations.teamID = teamID
    if (viewID) conversations.viewID = viewID
    if (conversations.status) {
      filters = filters.filter(f => f.model !== 'conversation_statuses')
      filters.push({
        model: 'conversation_statuses',
        field: 'name',
        operator: 'equals',
        value: conversations.status
      })
    }
    conversations.listFilters = filters
    const guard = showLoader ? delayedLoading(conversations, 'loading') : null
    conversations.fetching = true
    if (page === 0) page = conversations.page
    const seq = contextSeq
    const isStale = () => seq !== contextSeq
    try {
      conversations.errorMessage = ''
      const response = await makeConversationListRequest(listType, teamID, viewID, filters, page)
      if (isStale()) return
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

  async function makeConversationListRequest (listType, teamID, viewID, filters, page) {
    filters = filters.length > 0 ? JSON.stringify(filters) : []
    switch (listType) {
      case CONVERSATION_LIST_TYPE.ASSIGNED:
        return await api.getAssignedConversations({
          page: page,
          page_size: CONV_LIST_PAGE_SIZE,
          order_by: sortFieldMap[conversations.sortField].model + "." + sortFieldMap[conversations.sortField].field,
          order: sortFieldMap[conversations.sortField].order,
          filters
        })
      case CONVERSATION_LIST_TYPE.UNASSIGNED:
        return await api.getUnassignedConversations({
          page: page,
          page_size: CONV_LIST_PAGE_SIZE,
          order_by: sortFieldMap[conversations.sortField].model + "." + sortFieldMap[conversations.sortField].field,
          order: sortFieldMap[conversations.sortField].order,
          filters
        })
      case CONVERSATION_LIST_TYPE.ALL:
        return await api.getAllConversations({
          page: page,
          page_size: CONV_LIST_PAGE_SIZE,
          order_by: sortFieldMap[conversations.sortField].model + "." + sortFieldMap[conversations.sortField].field,
          order: sortFieldMap[conversations.sortField].order,
          filters
        })
      case CONVERSATION_LIST_TYPE.TEAM_UNASSIGNED:
        return await api.getTeamUnassignedConversations(teamID, {
          page: page,
          page_size: CONV_LIST_PAGE_SIZE,
          order_by: sortFieldMap[conversations.sortField].model + "." + sortFieldMap[conversations.sortField].field,
          order: sortFieldMap[conversations.sortField].order,
          filters
        })
      case CONVERSATION_LIST_TYPE.VIEW:
        return await api.getViewConversations(viewID, {
          page: page,
          page_size: CONV_LIST_PAGE_SIZE,
          order_by: sortFieldMap[conversations.sortField].model + "." + sortFieldMap[conversations.sortField].field,
          order: sortFieldMap[conversations.sortField].order
        })
      case CONVERSATION_LIST_TYPE.MENTIONED:
        return await api.getMentionedConversations({
          page: page,
          page_size: CONV_LIST_PAGE_SIZE,
          order_by: sortFieldMap[conversations.sortField].model + "." + sortFieldMap[conversations.sortField].field,
          order: sortFieldMap[conversations.sortField].order,
          filters
        })
      default:
        throw new Error('Invalid conversation list type: ' + listType)
    }
  }

  function trimListToCurrentPage () {
    const maxLen = conversations.page * CONV_LIST_PAGE_SIZE
    if (conversations.data.length > maxLen) {
      conversations.data.splice(maxLen)
    }
  }

  function mergeIntoList (uuid, payload) {
    const existing = conversations.data?.find(c => c.uuid === uuid)
    if (existing) deepMerge(existing, payload)
    return existing
  }

  function processConversationListResponse (response) {
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

    // Re-check document.hidden in case the user returned while the refresh was in flight.
    if (pendingNotificationUUIDs.size > 0) {
      let shouldPlay = false
      for (const uuid of pendingNotificationUUIDs) {
        if (isConversationInList(uuid)) {
          shouldPlay = true
        }
      }
      pendingNotificationUUIDs.clear()
      if (shouldPlay && document.hidden) {
        playNotificationSound()
      }
    }
  }

  async function updatePriority (v) {
    try {
      await api.updateConversationPriority(conversation.data.uuid, { priority: v })
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  async function updateStatus (v) {
    if (!conversation.data) return
    const previous = conversation.data.status
    conversation.data.status = v
    try {
      await api.updateConversationStatus(conversation.data.uuid, { status: v })
    } catch (error) {
      if (conversation.data) conversation.data.status = previous
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  async function snoozeConversation (snoozeDuration) {
    try {
      await api.updateConversationStatus(conversation.data.uuid, { status: CONVERSATION_DEFAULT_STATUSES.SNOOZED, snoozed_until: snoozeDuration })
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  function applyTagsLocally (uuid, action, tags) {
    const targets = []
    const listConv = conversations.data?.find(c => c.uuid === uuid)
    if (listConv) targets.push(listConv)
    if (conversation.data?.uuid === uuid) targets.push(conversation.data)

    for (const conv of targets) {
      if (!Array.isArray(conv.tags)) conv.tags = []
      if (action === TAG_ACTION.ADD) {
        for (const t of tags) {
          if (!conv.tags.includes(t)) conv.tags.push(t)
        }
      } else if (action === TAG_ACTION.SET) {
        conv.tags = [...tags]
      } else if (action === TAG_ACTION.REMOVE) {
        conv.tags = conv.tags.filter(t => !tags.includes(t))
      }
    }
  }

  async function updateConversationTags (uuid, action, tags) {
    const source = conversation.data?.uuid === uuid
      ? conversation.data
      : conversations.data?.find(c => c.uuid === uuid)
    const previous = source ? [...(source.tags || [])] : null
    applyTagsLocally(uuid, action, tags)
    try {
      await api.upsertTags(uuid, { action, tags })
    } catch (error) {
      if (previous) applyTagsLocally(uuid, TAG_ACTION.SET, previous)
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
      throw error
    }
  }

  async function updateAssignee (type, v) {
    try {
      await api.updateAssignee(conversation.data.uuid, type, v)
      conversation.data[`assigned_${type}_id`] = v.assignee_id
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  async function removeAssignee (type) {
    try {
      await api.removeAssignee(conversation.data.uuid, type)
      conversation.data[`assigned_${type}_id`] = null
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  async function updateAssigneeLastSeen (uuid) {
    if (!isViewingConversation(uuid)) return
    markConversationAsRead(uuid)
    api.updateAssigneeLastSeen(uuid).catch(() => { })
  }

  function isConversationInList (uuid) {
    return Boolean(conversations.data?.find(c => c.uuid === uuid))
  }

  const pendingNotificationUUIDs = new Set()

  function addPendingNotification (uuid) {
    pendingNotificationUUIDs.add(uuid)
  }

  // trailing=true: fires one final refresh after a burst so the list converges to latest state.
  const throttledFetchFirstPage = useThrottleFn(fetchFirstPageConversations, 60000, true)

  function refreshConversationList () {
    throttledFetchFirstPage()
  }

  function updateConversationLastMessage (uuid, message) {
    const conv = conversations.data?.find(c => c.uuid === uuid)
    if (!conv) return
    conv.last_message = message.text_content || message.content || getMediaPreview(message.attachments)
    conv.last_message_at = message.created_at
    conv.last_message_sender = message.sender_type
  }

  async function updateConversationMessage (message) {
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

  function addPendingMessage (conversationUUID, content, isPrivate, author, attachments = [], textContent = '', meta = {}) {
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
      attachments: attachments.map(a => ({
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

    // Safety net: auto-remove after 10 seconds if still pending.
    const tempId = pendingMessage.uuid
    setTimeout(() => {
      if (messages.data.hasMessage(conversationUUID, tempId)) {
        messages.data.removeMessage(conversationUUID, tempId)
        incrementMessageVersion()
      }
    }, 10000)

    return pendingMessage.uuid
  }

  function replacePendingMessage (conversationUUID, tempUUID, realMessage) {
    if (messages.data.hasMessage(conversationUUID, realMessage.uuid)) {
      messages.data.removeMessage(conversationUUID, tempUUID)
    } else {
      messages.data.updateMessage(conversationUUID, tempUUID, realMessage)
    }
    incrementMessageVersion()
  }

  function removePendingMessage (conversationUUID, tempUUID) {
    messages.data.removeMessage(conversationUUID, tempUUID)
    incrementMessageVersion()
  }

  function addNewConversation (conversation) {
    if (!isConversationInList(conversation.uuid)) {
      refreshConversationList()
    }
  }

  function mergeMessageUpdate (data) {
    const { conversation_uuid, uuid, ...fields } = data
    if (!messages.data.hasMessage(conversation_uuid, uuid)) return
    messages.data.updateMessage(conversation_uuid, uuid, fields)
    incrementMessageVersion()
  }

  function canPushInsert (conv) {
    const matched = matchesAssignmentScope(conv)
    if (matched !== null) return matched
    return conversations.listType === CONVERSATION_LIST_TYPE.ALL
  }

  function handleConvPush (payload) {
    if (!payload || !payload.uuid) return
    if (mergeIntoList(payload.uuid, payload)) {
      if (conversation.data?.uuid === payload.uuid) {
        deepMerge(conversation.data, payload)
      }
      return
    }
    if (!canPushInsert(payload)) return
    if (conversations.status !== '' && payload.status !== conversations.status) return
    if (!conversations.data) conversations.data = []
    conversations.data.unshift(payload)
    conversations.total += 1
    trimListToCurrentPage()
  }

  function mergeConversationUpdate (update) {
    if (conversation.data?.uuid === update.uuid) {
      deepMerge(conversation.data, update)
    }
    mergeIntoList(update.uuid, update)
  }

  function mergeContactUpdate (update) {
    const { contact_id, ...fields } = update
    if (conversation.data?.contact_id === contact_id) {
      if (!conversation.data.contact) conversation.data.contact = {}
      deepMerge(conversation.data.contact, fields)
    }
    conversations?.data?.forEach(c => {
      if (c.contact_id === contact_id) {
        if (!c.contact) c.contact = {}
        deepMerge(c.contact, fields)
      }
    })
  }

  function resetConversations () {
    conversations.data = []
    conversations.page = 1
    conversations.initialized = false
    conversations.hasMore = false
    conversations.total = 0
    contextSeq++
    pendingNotificationUUIDs.clear()
    clearSelection()
  }

  function setMacro (macro, context) {
    macros.value[context] = macro
  }

  function setMacroActions (actions, context) {
    if (!macros.value[context]) {
      macros.value[context] = {}
    }
    macros.value[context].actions = actions
  }

  function getMacro (context) {
    return macros.value[context] || {}
  }

  function removeMacroAction (action, context) {
    if (!macros.value[context]) return
    macros.value[context].actions = macros.value[context].actions.filter(a => a.type !== action.type)
  }

  function resetMacro (context) {
    macros.value = { ...macros.value, [context]: {} }
  }

  function updateTypingStatus (typingData) {
    const { conversation_uuid: uuid, is_typing } = typingData

    if (conversation.data?.uuid === uuid) {
      if (typingTimeout) {
        clearTimeout(typingTimeout)
        typingTimeout = null
      }
      conversation.isTyping = is_typing
      if (is_typing) {
        typingTimeout = setTimeout(() => {
          conversation.isTyping = false
          typingTimeout = null
        }, TYPING_RECEIVE_TIMEOUT)
      }
    }

    const prev = typingTimeoutsByUUID.get(uuid)
    if (prev) clearTimeout(prev)
    if (is_typing) {
      typingByUUID[uuid] = true
      typingTimeoutsByUUID.set(uuid, setTimeout(() => {
        delete typingByUUID[uuid]
        typingTimeoutsByUUID.delete(uuid)
      }, TYPING_RECEIVE_TIMEOUT))
    } else {
      delete typingByUUID[uuid]
      typingTimeoutsByUUID.delete(uuid)
    }
  }

  function sendTyping (isTyping, otherAttributes = {}) {
    if (conversation.data?.uuid) {
      sendTypingIndicator(conversation.data.uuid, isTyping, otherAttributes.isPrivateMessage)
    }
  }

  async function fetchAllDrafts () {
    try {
      const resp = await api.getAllDrafts()
      const newDrafts = new Map()
      if (resp.data?.data) {
        for (const draft of resp.data.data) {
          newDrafts.set(draft.conversation_uuid, draft)
        }
      }
      drafts.value = newDrafts
    } catch (e) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(e).message
      })
    }
  }

  function getDraft (uuid) {
    return drafts.value.get(uuid)
  }

  function setDraft (uuid, draft) {
    drafts.value.set(uuid, draft)
    drafts.value = new Map(drafts.value)
  }

  function removeDraft (uuid) {
    drafts.value.delete(uuid)
    drafts.value = new Map(drafts.value)
  }

  function hasDraft (uuid) {
    return drafts.value.has(uuid)
  }


  function getMediaPreview (attachments) {
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
    () => conversations.data?.map(c => c.uuid).sort().join(',') ?? '',
    () => subscribeListReplace(conversations.data?.map(c => c.uuid) || [])
  )

  return {
    macros,
    conversations,
    conversation,
    messages,
    conversationsList,
    conversationMessages,
    currentConversationHasMoreMessages,
    isConversationOpen,
    current,
    currentContactName,
    currentTo,
    currentBCC,
    currentCC,
    isConversationInList,
    addPendingNotification,
    mergeConversationUpdate,
    handleConvPush,
    mergeContactUpdate,
    addNewConversation,
    getContactFullName,
    fetchNextMessages,
    fetchNextConversations,
    mergeMessageUpdate,
    updateAssigneeLastSeen,
    markAsUnread,
    incrementUnread,
    updateConversationMessage,
    snoozeConversation,
    fetchConversation,
    fetchConversationsList,
    fetchMessages,
    updateConversationTags,
    updateAssignee,
    updatePriority,
    updateStatus,
    refreshConversationList,
    resetConversations,
    updateConversationLastMessage,
    fetchFirstPageConversations,
    fetchStatuses,
    fetchPriorities,
    setListSortField,
    setListStatus,
    removeMacroAction,
    getMacro,
    setMacro,
    resetMacro,
    setMacroActions,
    removeAssignee,
    getListSortField,
    getListStatus,
    statuses,
    priorities,
    priorityOptions,
    statusOptionsNoSnooze,
    statusOptions,
    updateTypingStatus,
    typingByUUID,
    sendTyping,
    drafts,
    fetchAllDrafts,
    getDraft,
    setDraft,
    removeDraft,
    hasDraft,
    addPendingMessage,
    replacePendingMessage,
    removePendingMessage,
    selectedUUIDs,
    selectedCount,
    allSelected,
    toggleSelect,
    selectAll,
    clearSelection,
    isSelected
  }
})
