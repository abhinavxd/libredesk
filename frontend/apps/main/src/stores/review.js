import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import { permissions as perms } from '@main/constants/permissions'
import { WS_EVENT } from '@main/constants/websocket'
import { getI18n } from '@main/i18n'
import { useUserStore } from '@main/stores/user'
import api from '@main/api'

// The toast a live review event deserves, or null. Reviewers hear about new
// submissions; authors hear about decisions on their own emails. Nobody is
// told about their own actions, which already toast where they happened.
export function reviewEventToast(type, data, { userID, isReviewer }, t) {
  if (!data) return null
  const subject = data.subject || t('review.noSubject')
  if (type === WS_EVENT.REVIEW_CREATED && isReviewer && data.author_id !== userID) {
    return { variant: 'info', description: t('review.toast.newSubmission', { name: data.author_name, subject }) }
  }
  if (type !== WS_EVENT.REVIEW_UPDATED || data.author_id !== userID) return null
  if (data.status === 'approved') {
    return { variant: 'success', description: t('review.toast.yourApproved', { name: data.reviewer_name, subject }) }
  }
  if (data.status === 'denied') {
    const key = data.decision_note ? 'review.toast.yourDeniedWithReason' : 'review.toast.yourDenied'
    return { variant: 'warning', description: t(key, { name: data.reviewer_name, subject, reason: data.decision_note }) }
  }
  return null
}

// Contributor emails waiting for, or returned by, a reviewer.
export const useReviewStore = defineStore('review', () => {
  const userStore = useUserStore()
  const emitter = useEmitter()
  const counts = ref({ pending: 0, returned: 0 })
  const items = ref([])
  const loading = ref(false)
  const loaded = ref(false)
  // Bumped on every live review event so open threads and the queue refetch.
  const version = ref(0)
  // Review cards per conversation UUID: pending submissions and the viewer's own denials.
  const threads = ref(new Map())

  const isReviewer = computed(() => userStore.can(perms.REVIEWS_MANAGE))
  const canSubmit = computed(() => userStore.can(perms.REVIEWS_SUBMIT))
  // A Contributor has no direct send: their replies and new emails are reviewed.
  const needsReview = computed(() => !userStore.can(perms.MESSAGES_WRITE) && canSubmit.value)
  const enabled = computed(() => isReviewer.value || canSubmit.value)
  const badgeCount = computed(() =>
    isReviewer.value ? counts.value.pending : counts.value.pending + counts.value.returned
  )

  const toast = (variant, description) => emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant, description })
  const toastError = (error) => toast('destructive', handleHTTPError(error).message)
  const t = (...args) => getI18n().global.t(...args)

  async function fetchCounts() {
    if (!enabled.value) return
    try {
      const response = await api.getReviewCounts()
      counts.value = { pending: 0, returned: 0, ...(response?.data?.data || {}) }
    } catch {
      // The badge is advisory; the queue itself reports errors.
    }
  }

  async function fetchReviews() {
    if (!enabled.value) return
    loading.value = true
    try {
      const response = await api.getReviews()
      items.value = response?.data?.data || []
      loaded.value = true
    } catch (error) {
      toastError(error)
    } finally {
      loading.value = false
    }
  }

  const conversationReviews = (conversationUUID) => threads.value.get(conversationUUID) || []

  async function fetchConversationReviews(conversationUUID) {
    if (!enabled.value || !conversationUUID) return
    try {
      const response = await api.getConversationReviews(conversationUUID)
      threads.value.set(conversationUUID, response?.data?.data || [])
      threads.value = new Map(threads.value)
    } catch {
      // Cards are supplementary to the thread; the thread itself still loads.
    }
  }

  function forget(uuid) {
    items.value = items.value.filter((item) => item.uuid !== uuid)
  }

  async function decide(request, uuid, successMessage) {
    try {
      const response = await request()
      forget(uuid)
      for (const [conversationUUID, reviews] of threads.value) {
        if (reviews.some((review) => review.uuid === uuid)) fetchConversationReviews(conversationUUID)
      }
      if (successMessage) toast('success', successMessage)
      fetchCounts()
      return response?.data?.data ?? true
    } catch (error) {
      toastError(error)
      // Someone else may have acted first; show the queue as it really is.
      fetchReviews()
      fetchCounts()
      return null
    }
  }

  const approve = (uuid) => decide(() => api.approveReview(uuid), uuid, t('review.toast.approved'))
  const deny = (uuid, note) => decide(() => api.denyReview(uuid, note), uuid, t('review.toast.denied'))
  const withdraw = (uuid, kind) =>
    decide(() => api.withdrawReview(uuid), uuid, t(kind === 'new' ? 'review.toast.withdrawnNew' : 'review.toast.withdrawnReply'))
  const discard = (uuid) => decide(() => api.discardReview(uuid), uuid, t('review.toast.discarded'))

  async function dismiss(uuid) {
    try {
      await api.dismissReview(uuid)
    } catch {
      // Dismissing a notice is cosmetic.
    }
  }

  function handleLiveEvent(type, data) {
    version.value++
    fetchCounts()
    if (loaded.value) fetchReviews()
    if (data?.conversation_uuid && threads.value.has(data.conversation_uuid)) fetchConversationReviews(data.conversation_uuid)
    const message = reviewEventToast(type, data, { userID: userStore.userID, isReviewer: isReviewer.value }, t)
    if (message) toast(message.variant, message.description)
  }

  return {
    counts,
    items,
    loading,
    loaded,
    version,
    conversationReviews,
    fetchConversationReviews,
    isReviewer,
    canSubmit,
    needsReview,
    enabled,
    badgeCount,
    fetchCounts,
    fetchReviews,
    approve,
    deny,
    withdraw,
    discard,
    dismiss,
    handleLiveEvent
  }
})
