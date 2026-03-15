import { defineStore } from 'pinia'
import { computed, reactive, ref, watchEffect } from 'vue'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { computeRecipientsFromMessage } from '../utils/email-recipients'
import { useEmitter } from '../composables/useEmitter'
import { EMITTER_EVENTS } from '../constants/emitterEvents'
import { subscribeToConversation, sendTypingIndicator } from '@main/websocket'
import MessageCache from '../utils/conversation-message-cache'
import { getI18n } from '../i18n'
import { useDebounceFn } from '@vueuse/core'
import { CONVERSATION_LIST_TYPE, CONVERSATION_DEFAULT_STATUSES } from '@/constants/conversation'
import api from '../api'

export const useConversationStore = defineStore('conversation', () => {
  const CONV_LIST_PAGE_SIZE = 50
  const MESSAGE_LIST_PAGE_SIZE = 30
  const priorities = ref([])
  const statuses = ref([])
  const currentTo = ref([])
  const currentBCC = ref([])
  const currentCC = ref([])
  const macros = ref({})
  const drafts = ref(new Map())

  // Options for select fields
  const priorityOptions = computed(() => {
    return priorities.value.map(p => ({ label: p.name, value: p.id }))
  })
  const statusOptions = computed(() => {
    return statuses.value.map(s => ({ label: s.name, value: s.id }))
  })
  // Status options excluding 'Snoozed'
  const statusOptionsNoSnooze = computed(() =>
    statuses.value.filter(s => s.name !== 'Snoozed').map(s => ({
      label: s.name,
      value: s.id
    }))
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

  const conversations = reactive({
    data: [],
    listType: null,
    status: 'Open',
    sortField: 'newest',
    listFilters: [],
    viewID: 0,
    teamID: 0,
    loading: false,
    page: 1,
    hasMore: false,
    total: 0,
    errorMessage: ''
  })

  const conversation = reactive({
    data: null,
    participants: {},
    loading: false,
    errorMessage: '',
    isTyping: false
  })

  const messages = reactive({
    data: new MessageCache(),
    loading: false,
    page: 1,
    // To trigger reactivity on the messages cache, simpler than making MessageCache reactive.
    version: 0,
  })

  let seenConversationUUIDs = new Map()
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

  const conversationsList = computed(() => {
    if (!conversations.data) return []
    let filteredConversations = conversations.data
    // Filter by status if set.
    if (conversations.status !== "") {
      filteredConversations = conversations.data
        .filter(conv => {
          return conv.status === conversations.status
        })
    }

    // Sort conversations based on the selected sort field
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
    const allMessages = messages.data.getAllPagesMessages(conversation.data?.uuid)
    // Filter out continuity emails.
    return allMessages.filter(msg => !msg.meta?.continuity_email)
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

  const hasConversationOpen = computed(() => {
    return Object.keys(conversation.data || {}).length > 0
  })

  // Watch for changes in the conversation and messages and update the to, cc, and bcc
  watchEffect(async () => {
    const _ = messages.version // eslint-disable-line no-unused-vars
    const conv = conversation.data
    const msgData = messages.data
    const inboxEmail = conv?.inbox_mail

    // If the conversation is a live chat, reset recipients.
    if (conv?.inbox_channel === 'livechat') {
      currentTo.value = []
      currentCC.value = []
      currentBCC.value = []
      return
    }

    if (!conv || !msgData || !inboxEmail) return

    const latestMessage = msgData.getLatestMessage(conv.uuid, ['incoming', 'outgoing'], true)
    if (!latestMessage) {
      // Reset recipients if no latest message is found.
      currentTo.value = []
      currentCC.value = []
      currentBCC.value = []
      return
    }

    const { to, cc, bcc } = computeRecipientsFromMessage(
      latestMessage,
      conv.contact?.email || '',
      inboxEmail
    )
    currentTo.value = to
    currentCC.value = cc
    currentBCC.value = bcc
  })

  async function fetchParticipants (uuid) {
    try {
      const resp = await api.getConversationParticipants(uuid)
      const participants = resp.data.data.reduce((acc, p) => {
        acc[p.id] = p
        return acc
      }, {})
      updateParticipants(participants)
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  async function fetchConversation (uuid) {
    conversation.loading = true
    try {
      const resp = await api.getConversation(uuid)
      conversation.data = resp.data.data
      // Do a websocket subscription to the conversation.
      subscribeToConversation(uuid)
    } catch (error) {
      conversation.errorMessage = handleHTTPError(error).message
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: conversation.errorMessage
      })
    } finally {
      conversation.loading = false
    }
  }

  /**
   * Fetches messages for a conversation if not already present in the cache.
   * 
   * @param {string} uuid
   * @returns 
   */
  async function fetchMessages (uuid, fetchNextPage = false) {
    // Messages are already cached?
    let hasMessages = messages.data.getAllPagesMessages(uuid)
    if (hasMessages.length > 0 && !fetchNextPage) {
      markConversationAsRead(uuid)
      return
    }

    // Fetch messages from server.
    messages.loading = true
    // Increment page number
    let page = messages.data.getLastFetchedPage(uuid) + 1
    try {
      const response = await api.getConversationMessages(uuid, { page: page, page_size: MESSAGE_LIST_PAGE_SIZE })
      const result = response.data?.data || {}
      const newMessages = result.results || []
      markConversationAsRead(uuid)
      // Cache messages
      messages.data.addMessages(uuid, newMessages, result.page, result.total_pages)
      incrementMessageVersion()
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    } finally {
      messages.loading = false
    }
  }

  async function fetchNextMessages () {
    fetchMessages(conversation.data.uuid, true)
  }

  /**
   * Fetches a single message from the server and adds it to the message cache.
   * 
   * @param {string} conversationUUID
   * @param {string} messageUUID
   * @returns {object}
   */
  async function fetchMessage (conversationUUID, messageUUID) {
    try {
      const response = await api.getConversationMessage(conversationUUID, messageUUID)
      if (response?.data?.data) {
        const newMsg = response.data.data
        // Add message to cache.
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
    conversations.page++
    fetchConversationsList(true, conversations.listType, conversations.teamID, conversations.listFilters, conversations.viewID, conversations.page)
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
    if (listType) conversations.listType = listType
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
    if (filters) conversations.listFilters = filters
    if (showLoader) conversations.loading = true
    try {
      conversations.errorMessage = ''
      if (page === 0)
        page = conversations.page
      const response = await makeConversationListRequest(listType, teamID, viewID, filters, page)
      processConversationListResponse(response)
    } catch (error) {
      conversations.errorMessage = handleHTTPError(error).message
      conversations.total = 0
    } finally {
      conversations.loading = false
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

  function processConversationListResponse (response) {
    const apiResponse = response.data.data
    const newConversations = []
    for (const conv of apiResponse.results) {
      if (seenConversationUUIDs.has(conv.uuid)) {
        // Update existing conversation with fresh data.
        const idx = conversations.data.findIndex(c => c.uuid === conv.uuid)
        if (idx !== -1) {
          Object.assign(conversations.data[idx], conv)
        }
      } else {
        // Add to seen and new conversations list.
        seenConversationUUIDs.set(conv.uuid, true)
        newConversations.push(conv)
      }
    }
    if (apiResponse.total_pages <= conversations.page) conversations.hasMore = false
    else conversations.hasMore = true
    if (!conversations.data) conversations.data = []
    conversations.data.push(...newConversations)
    conversations.total = apiResponse.total
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
    try {
      await api.updateConversationStatus(conversation.data.uuid, { status: v })
    } catch (error) {
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

  async function upsertTags (v) {
    try {
      await api.upsertTags(conversation.data.uuid, v)
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  async function updateAssignee (type, v) {
    try {
      await api.updateAssignee(conversation.data.uuid, type, v)
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
    try {
      await api.updateAssigneeLastSeen(uuid)
    } catch (error) {
      // pass
    }
  }

  function updateParticipants (newParticipants) {
    conversation.participants = {
      ...conversation.participants,
      ...newParticipants
    }
  }

  function conversationUUIDExists (uuid) {
    return conversations.data?.find(c => c.uuid === uuid) ? true : false
  }

  // Debounced to prevent apis calls during many WS events in a short time.
  const debouncedFetchFirstPage = useDebounceFn(fetchFirstPageConversations, 1000)
  const debouncedFetchParticipants = useDebounceFn(fetchParticipants, 400)

  function refreshConversationList () {
    debouncedFetchFirstPage()
  }

  function updateConversationLastMessage (uuid, message) {
    const conv = conversations.data?.find(c => c.uuid === uuid)
    if (!conv) return
    conv.last_message = message.text_content || message.content || getMediaPreview(message.attachments)
    conv.last_message_at = message.created_at
    conv.last_message_sender = message.sender_type
  }

  /**
   * Update conversation message in the cache by fetching it from the server.
   *
   * @param {object} message - Message object with conversation_uuid field
   */
  async function updateConversationMessage (message) {
    if (conversation.data?.uuid !== message.conversation_uuid) {
      // Not the open conversation. If we have cached messages for it,
      // fetch the new message to keep the cache fresh.
      if (messages.data.getLastFetchedPage(message.conversation_uuid) > 0) {
        const fetchedMessage = await fetchMessage(message.conversation_uuid, message.uuid)
        if (fetchedMessage) {
          // Update last message in conversation list (preview)
          updateConversationLastMessage(message.conversation_uuid, fetchedMessage)
        }
      }
      return
    }

    // Open conversation and message not in cache? Fetch from server.
    if (!messages.data.hasMessage(message.conversation_uuid, message.uuid)) {
      debouncedFetchParticipants(message.conversation_uuid)
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

      updateAssigneeLastSeen(message.conversation_uuid)
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
      // WS already delivered the real message, just remove the temp.
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
    if (!conversationUUIDExists(conversation.uuid)) {
      // Fetch list of conversations again.
      fetchFirstPageConversations()
    }
  }

  /**
   * Update a single message property in the cache.
   * 
   * @param {Object} message - Message
   */
  function updateMessageProp (message) {
    const exists = messages.data.hasMessage(message.conversation_uuid, message.uuid)
    if (exists) {
      messages.data.updateMessageField(message.conversation_uuid, message.uuid, message.prop, message.value)
      incrementMessageVersion()
    }
  }

  /**
   * Update a conversation property, supports nested paths via dot notation
   * @param {Object} update - { uuid, prop, value } or { contact_id, prop, value }
   */
  function updateConversationProp (update) {
    const updateNested = (obj, prop, value) => {
      if (!prop.includes('.')) {
        obj[prop] = value
        return
      }

      const keys = prop.split('.')
      const lastKey = keys.pop()
      const target = keys.reduce((o, key) => o[key] = o[key] || {}, obj)
      target[lastKey] = value
    }

    const { uuid, contact_id, prop, value } = update

    // Handle contact-level updates (broadcast by contact_id)
    if (contact_id && prop?.startsWith('contact.')) {
      // Update current conversation if it belongs to this contact
      if (conversation.data?.contact_id === contact_id) {
        updateNested(conversation.data, prop, value)
      }
      // Update conversations in the list that belong to this contact
      conversations?.data?.forEach(c => {
        if (c.contact_id === contact_id) {
          updateNested(c, prop, value)
        }
      })
      return
    }

    // Conversation is currently open? Update it.
    if (conversation.data?.uuid === uuid) {
      updateNested(conversation.data, prop, value)
    }

    // Update conversation if it exists in the list.
    const existingConversation = conversations?.data?.find(c => c.uuid === uuid)
    if (existingConversation) {
      updateNested(existingConversation, prop, value)
    }
  }

  function resetConversations () {
    conversations.data = []
    conversations.page = 1
    seenConversationUUIDs = new Map()
  }

  /** Macros set for new conversation or an open conversation **/
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

  // Typing indicators
  function updateTypingStatus (typingData) {
    const { conversation_uuid, is_typing } = typingData

    // Only update typing status for the current conversation
    if (conversation.data?.uuid !== conversation_uuid) return

    conversation.isTyping = is_typing
  }

  function sendTyping (isTyping, otherAttributes = {}) {
    // Send typing websocket message only if a conversation is open
    if (conversation.data?.uuid) {
      sendTypingIndicator(conversation.data.uuid, isTyping, otherAttributes.isPrivateMessage)
    }
  }

  // Fetch all drafts for the current user
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

  // Get draft for a specific conversation
  function getDraft (uuid) {
    return drafts.value.get(uuid)
  }

  // Set draft for a specific conversation
  function setDraft (uuid, draft) {
    drafts.value.set(uuid, draft)
    // Trigger reactivity
    drafts.value = new Map(drafts.value)
  }

  // Remove draft for a specific conversation
  function removeDraft (uuid) {
    drafts.value.delete(uuid)
    // Trigger reactivity
    drafts.value = new Map(drafts.value)
  }

  // Check if a conversation has a draft
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

  return {
    macros,
    conversations,
    conversation,
    messages,
    conversationsList,
    conversationMessages,
    currentConversationHasMoreMessages,
    hasConversationOpen,
    current,
    currentContactName,
    currentTo,
    currentBCC,
    currentCC,
    conversationUUIDExists,
    updateConversationProp,
    addNewConversation,
    getContactFullName,
    fetchParticipants,
    fetchNextMessages,
    fetchNextConversations,
    updateMessageProp,
    updateAssigneeLastSeen,
    markAsUnread,
    updateConversationMessage,
    snoozeConversation,
    fetchConversation,
    fetchConversationsList,
    fetchMessages,
    upsertTags,
    updateAssignee,
    updatePriority,
    updateStatus,
    refreshConversationList,
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
    sendTyping,
    drafts,
    fetchAllDrafts,
    getDraft,
    setDraft,
    removeDraft,
    hasDraft,
    addPendingMessage,
    replacePendingMessage,
    removePendingMessage
  }
})
