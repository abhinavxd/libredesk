import { defineStore } from 'pinia'
import { computed, reactive, ref, nextTick, watchEffect } from 'vue'
import { CONVERSATION_LIST_TYPE, CONVERSATION_DEFAULT_STATUSES } from '@/constants/conversation'
import { handleHTTPError } from '@/utils/http'
import { computeRecipientsFromMessage } from '@/utils/email-recipients'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import MessageCache from '@/utils/conversation-message-cache'
import api from '@/api'

export const useConversationStore = defineStore('conversation', () => {
  const CONV_LIST_PAGE_SIZE = 50
  const MESSAGE_LIST_PAGE_SIZE = 30
  const priorities = ref([])
  const statuses = ref([])
  const currentTo = ref([])
  const currentBCC = ref([])
  const currentCC = ref([])

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

  const sortFieldLabels = {
    oldest: 'Oldest activity',
    newest: 'Newest activity',
    started_first: 'Started first',
    started_last: 'Started last',
    waiting_longest: 'Waiting longest',
    next_sla_target: 'Next SLA target',
    priority_first: 'Priority first'
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
    mediaFiles: [],
    macro: {},
    loading: false,
    errorMessage: ''
  })

  const messages = reactive({
    data: new MessageCache(),
    loading: false,
    page: 1,
    // To trigger reactivity on the messages cache, simpler than making MessageCache reactive.
    version: 0,
  })

  let seenConversationUUIDs = new Map()
  let reRenderInterval = setInterval(() => {
    conversations.data = [...conversations.data]
  }, 120000)
  const emitter = useEmitter()

  const incrementMessageVersion = () => setTimeout(() => messages.version++, 0)

  function clearListReRenderInterval () {
    clearInterval(reRenderInterval)
  }

  async function setMacro (macro) {
    // Clear existing macro.
    conversation.macro = {}
    await nextTick()
    conversation.macro = macro
  }

  function removeMacroAction (action) {
    conversation.macro.actions = conversation.macro.actions.filter(a => a.type !== action.type)
  }

  function resetMacro () {
    conversation.macro = {}
  }

  function resetMediaFiles () {
    conversation.mediaFiles = []
  }

  function setListStatus (status, fetch = true) {
    conversations.status = status
    if (fetch) {
      resetConversations()
      reFetchConversationsList()
    }
  }

  function setListSortField (field) {
    if (conversations.sortField === field) return
    conversations.sortField = field
    resetConversations()
    reFetchConversationsList()
  }

  const getListSortField = computed(() => {
    return sortFieldLabels[conversations.sortField]
  })

  const getListStatus = computed(() => {
    return conversations.status
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
        title: 'Error',
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
        title: 'Error',
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  const conversationsList = computed(() => {
    if (!conversations.data) return []
    // Sort conversations based on the selected sort field
    return [...conversations.data].sort((a, b) => {
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
        conversations.data[index].unread_message_count = 0
      }, 3000)
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
    if (!conv || !msgData || !inboxEmail) return

    const latestMessage = msgData.getLatestMessage(conv.uuid, ['incoming', 'outgoing'], true)
    if (!latestMessage) return

    if (!["received", "sent"].includes(latestMessage.status)) {
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
        title: 'Error',
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
    } catch (error) {
      conversation.errorMessage = handleHTTPError(error).message
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        title: 'Error',
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
        title: 'Error',
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
        title: 'Error',
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
    if (conversations.listType !== listType) {
      resetCurrentConversation()
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
          order: sortFieldMap[conversations.sortField].order
        })
      case CONVERSATION_LIST_TYPE.VIEW:
        return await api.getViewConversations(viewID, {
          page: page,
          page_size: CONV_LIST_PAGE_SIZE,
          order_by: sortFieldMap[conversations.sortField].model + "." + sortFieldMap[conversations.sortField].field,
          order: sortFieldMap[conversations.sortField].order
        })
      default:
        throw new Error('Invalid conversation list type: ' + listType)
    }
  }

  function processConversationListResponse (response) {
    const apiResponse = response.data.data
    const newConversations = apiResponse.results.filter(conversation => {
      if (!seenConversationUUIDs.has(conversation.uuid)) {
        seenConversationUUIDs.set(conversation.uuid, true)
        return true
      }
      return false
    })
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
        title: 'Error',
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
        title: 'Error',
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
        title: 'Error',
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
        title: 'Error',
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
        title: 'Error',
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
        title: 'Error',
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

  function updateConversationList (message) {
    const listConversation = conversations.data.find(c => c.uuid === message.conversation_uuid)
    if (listConversation) {
      listConversation.last_message = message.content
      listConversation.last_message_at = message.created_at
      listConversation.last_message_sender = message.sender_type
      if (listConversation.uuid !== conversation?.data?.uuid) {
        listConversation.unread_message_count += 1
      }
    } else {
      // Conversation is not in the list, fetch the first page of the conversations list as this updated conversation might be at the top.
      fetchFirstPageConversations()
    }
  }

  /**
   * Update conversation message in the cache by fetching it from the server.
   * 
   * @param {object} message - Message object with conversation_uuid field
   */
  async function updateConversationMessage (message) {
    // Message does not exist in cache, fetch from server and update.
    if (!messages.data.hasMessage(message.conversation_uuid, message.uuid)) {
      fetchParticipants(message.conversation_uuid)
      const fetchedMessage = await fetchMessage(message.conversation_uuid, message.uuid)
      setTimeout(() => {
        emitter.emit(EMITTER_EVENTS.NEW_MESSAGE, {
          conversation_uuid: message.conversation_uuid,
          message: fetchedMessage
        })
      }, 100)
      updateAssigneeLastSeen(message.conversation_uuid)
    }
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

  function updateConversationProp (update) {
    // Update the current conversation if it matches the UUID.
    if (conversation.data?.uuid === update.uuid) {
      conversation.data[update.prop] = update.value
    }
    // Update the conversation in the list if it exists.
    const existingConversation = conversations?.data?.find(c => c.uuid === update.uuid)
    if (existingConversation) {
      existingConversation[update.prop] = update.value
    }
  }

  function resetCurrentConversation () {
    Object.assign(conversation, {
      data: null,
      participants: {},
      mediaFiles: [],
      macro: {},
      loading: false,
      errorMessage: ''
    })
  }

  function resetConversations () {
    conversations.data = []
    conversations.page = 1
    seenConversationUUIDs = new Map()
  }


  return {
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
    clearListReRenderInterval,
    conversationUUIDExists,
    updateConversationProp,
    addNewConversation,
    getContactFullName,
    fetchParticipants,
    fetchNextMessages,
    fetchNextConversations,
    updateMessageProp,
    updateAssigneeLastSeen,
    updateConversationMessage,
    snoozeConversation,
    fetchConversation,
    fetchConversationsList,
    fetchMessages,
    upsertTags,
    updateAssignee,
    updatePriority,
    updateStatus,
    updateConversationList,
    resetCurrentConversation,
    fetchFirstPageConversations,
    fetchStatuses,
    fetchPriorities,
    setListSortField,
    setListStatus,
    removeMacroAction,
    setMacro,
    resetMacro,
    resetMediaFiles,
    removeAssignee,
    getListSortField,
    getListStatus,
    statuses,
    priorities,
    priorityOptions,
    statusOptionsNoSnooze,
    statusOptions
  }
})
