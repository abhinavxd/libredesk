// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createApp, h, nextTick, reactive } from 'vue'
let store, review
const { api } = vi.hoisted(() => ({ api: { sendMessage: vi.fn(), submitReplyForReview: vi.fn() } }))
vi.mock('@main/api', () => ({ default: api }))
vi.mock('@main/stores/conversation', () => ({ useConversationStore: () => store }))
vi.mock('@main/stores/review', () => ({ useReviewStore: () => review }))
vi.mock('@main/stores/user', () => ({ useUserStore: () => ({ can: () => true, userID: 1 }) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key) => key }) }))
vi.mock('@main/composables/useEmitter', () => ({
  useEmitter: () => ({ on: vi.fn(), off: vi.fn(), emit: vi.fn() })
}))
vi.mock('@main/composables/useVisualViewportHeight', () => ({ useVisualViewportHeight: () => {} }))
vi.mock('@main/composables/useIsComposerCramped', async () => {
  const { ref } = await import('vue')
  return { useIsComposerCramped: () => ref(false) }
})
vi.mock('./ReplyBoxContent.vue', async () => {
  const { h } = await import('vue')
  return {
    default: {
      props: ['htmlContent', 'textContent'],
      emits: ['update:htmlContent', 'update:textContent', 'sendAndSetStatus'],
      setup(props, { emit }) {
        return () =>
          h('div', [
            h('textarea', {
              value: props.htmlContent,
              onInput: (event) => {
                emit('update:htmlContent', event.target.value)
                emit('update:textContent', event.target.value)
              }
            }),
            h('button', { onClick: () => emit('sendAndSetStatus', 'Closed') }, 'Send and close')
          ])
      }
    }
  }
})
import ReplyBox from './ReplyBox.vue'
const settle = async () => {
  for (let i = 0; i < 10; i++) await nextTick()
}
let app, root, drafts
beforeEach(async () => {
  vi.resetAllMocks()
  drafts = new Map()
  store = reactive({
    current: { uuid: 'A', inbox_channel: 'email', correspondent: { email: 'alice@example.test' } },
    messages: { data: { hasConversation: () => true } },
    conversationMessages: [],
    currentTo: ['alice@example.test'],
    currentCC: [],
    currentBCC: [],
    draftsReady: Promise.resolve(),
    draftSaveStates: new Map(),
    getDraft: (id, type) => drafts.get(`${id}::${type}`),
    setDraft: (id, type, draft) => drafts.set(`${id}::${type}`, draft),
    removeDraft: (id, type) => drafts.delete(`${id}::${type}`),
    syncDraft: vi.fn(),
    retryDraftSave: vi.fn(),
    resolveDraftType: () => 'reply',
    setSelectedDraftType: vi.fn(),
    addPendingMessage: vi.fn(() => 'pending-A'),
    removePendingMessage: vi.fn(),
    replacePendingMessage: vi.fn(),
    updateStatus: vi.fn()
  })
  review = reactive({
    needsReview: false,
    enabled: true,
    threads: {},
    conversationReviews(uuid) {
      return this.threads[uuid] || []
    },
    fetchConversationReviews: vi.fn(),
    fetchCounts: vi.fn(),
    withdraw: vi.fn(),
    dismiss: vi.fn()
  })
  root = document.createElement('div')
  document.body.append(root)
  app = createApp({ render: () => h(ReplyBox) })
  app.config.globalProperties.$t = (key) => key
  app.mount(root)
  await settle()
})
afterEach(() => {
  app.unmount()
  root.remove()
})
function type(value) {
  const textarea = root.querySelector('textarea')
  textarea.value = value
  textarea.dispatchEvent(new Event('input', { bubbles: true }))
}

describe('reply sends across navigation', () => {
  it('acknowledges the original thread and does not clear the new thread draft', async () => {
    let finish
    api.sendMessage.mockImplementation(
      () =>
        new Promise((resolve) => {
          finish = resolve
        })
    )
    type('A reply')
    await settle()
    root.querySelector('button').click()
    await settle()
    store.current = { uuid: 'B', inbox_channel: 'email' }
    await settle()
    type('B draft')
    await settle()
    finish({ data: { data: { uuid: 'real-A' } } })
    await settle()
    expect(store.replacePendingMessage).toHaveBeenCalledWith('A', 'pending-A', { uuid: 'real-A' })
    expect(store.updateStatus).toHaveBeenCalledWith('Closed', 'A')
    expect(root.querySelector('textarea').value).toBe('B draft')
    expect(drafts.has('A::reply')).toBe(false)
  })

  it('keeps both drafts when the old thread send fails', async () => {
    let fail
    api.sendMessage.mockImplementation(
      () =>
        new Promise((_, reject) => {
          fail = reject
        })
    )
    type('A reply')
    await settle()
    root.querySelector('button').click()
    await settle()
    store.current = { uuid: 'B', inbox_channel: 'email' }
    await settle()
    type('B draft')
    await settle()
    fail(new Error('offline'))
    await settle()
    expect(root.querySelector('textarea').value).toBe('B draft')
    expect(drafts.get('A::reply').content).toBe('A reply')
    expect(store.updateStatus).not.toHaveBeenCalled()
  })
})

describe('contributor replies', () => {
  it('holds the reply for review instead of sending it, then locks the composer', async () => {
    review.needsReview = true
    api.submitReplyForReview.mockResolvedValue({ data: { data: { uuid: 'r1' } } })
    review.fetchConversationReviews.mockImplementation(async (uuid) => {
      review.threads[uuid] = [{ uuid: 'r1', status: 'pending', author_id: 1, preview: 'A reply' }]
    })
    type('A reply')
    await settle()
    root.querySelector('button').click()
    await settle()
    expect(api.sendMessage).not.toHaveBeenCalled()
    expect(api.submitReplyForReview).toHaveBeenCalledWith(
      'A',
      expect.objectContaining({ content: 'A reply', to: ['alice@example.test'], attachments: [] })
    )
    // Contributors cannot change status, and the sent draft is cleared.
    expect(store.updateStatus).not.toHaveBeenCalled()
    expect(drafts.has('A::reply')).toBe(false)
    expect(root.querySelector('textarea')).toBeNull()
    expect(root.textContent).toContain('review.yourReplyAwaiting')
  })

  it('keeps the draft when the submission fails', async () => {
    review.needsReview = true
    api.submitReplyForReview.mockRejectedValue(new Error('offline'))
    type('A reply')
    await settle()
    root.querySelector('button').click()
    await settle()
    expect(drafts.get('A::reply').content).toBe('A reply')
    expect(root.querySelector('textarea').value).toBe('A reply')
  })

  it('shows why a reply came back', async () => {
    review.threads.A = [{ uuid: 'r2', status: 'denied', author_id: 1, reviewer_name: 'Robin', decision_note: 'Add a greeting' }]
    await settle()
    expect(root.textContent).toContain('review.returnedNoticeReason')
    expect(root.querySelector('textarea')).not.toBeNull()
  })
})
