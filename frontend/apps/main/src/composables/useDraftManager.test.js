// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { effectScope, nextTick, reactive, ref, watch } from 'vue'
const { store } = vi.hoisted(() => ({ store: {} }))
vi.mock('@main/stores/conversation', () => ({ useConversationStore: () => store }))
import { useDraftManager } from './useDraftManager'

const settle = async () => {
  for (let i = 0; i < 5; i++) await nextTick()
}
let scope, uuid, type, media, defaults, editor, drafts
beforeEach(async () => {
  vi.useFakeTimers()
  drafts = new Map()
  Object.assign(store, {
    draftsReady: Promise.resolve(),
    draftSaveStates: reactive(new Map()),
    getDraft: (id, kind) => drafts.get(`${id}::${kind}`),
    setDraft: (id, kind, draft) => drafts.set(`${id}::${kind}`, draft),
    removeDraft: (id, kind) => drafts.delete(`${id}::${kind}`),
    syncDraft: vi.fn(),
    retryDraftSave: vi.fn()
  })
  uuid = ref('A')
  type = ref('reply')
  media = ref([])
  defaults = ref({
    uuid: 'A',
    ready: true,
    recipients: { to: 'alice@example.test', cc: '', bcc: '' }
  })
  scope = effectScope()
  scope.run(() => {
    editor = useDraftManager(uuid, type, media, defaults)
    watch(editor.loadedAttachments, (files) => {
      media.value = [...files]
    })
  })
  await settle()
})
afterEach(() => {
  scope.stop()
  vi.clearAllTimers()
  vi.useRealTimers()
})

describe('UUID-keyed reply drafts', () => {
  it('preserves recipient edits across message updates, switches and reloads', async () => {
    expect(editor.recipients.value.to).toBe('alice@example.test')
    editor.htmlContent.value = '<p>Hello Alice</p>'
    editor.recipients.value.cc = 'colleague@example.test'
    editor.recipients.value.bcc = 'archive@example.test'
    defaults.value = {
      uuid: 'A',
      ready: true,
      recipients: { to: 'new@example.test', cc: '', bcc: '' }
    }
    await settle()
    expect(editor.recipients.value.to).toBe('alice@example.test')
    expect(editor.recipients.value.cc).toBe('colleague@example.test')
    uuid.value = 'B'
    defaults.value = {
      uuid: 'B',
      ready: true,
      recipients: { to: 'bob@example.test', cc: '', bcc: '' }
    }
    await settle()
    expect(editor.recipients.value.to).toBe('bob@example.test')
    uuid.value = 'A'
    await settle()
    expect(editor.htmlContent.value).toBe('<p>Hello Alice</p>')
    expect(editor.recipients.value.bcc).toBe('archive@example.test')
    expect(store.syncDraft).toHaveBeenCalledWith(
      'A',
      'reply',
      expect.objectContaining({
        meta: expect.objectContaining({
          recipients: expect.objectContaining({ cc: 'colleague@example.test' })
        })
      })
    )
  })

  it('does not clear another thread when an earlier send finishes', async () => {
    editor.htmlContent.value = '<p>A reply</p>'
    const sent = editor.captureDraft()
    uuid.value = 'B'
    defaults.value = {
      uuid: 'B',
      ready: true,
      recipients: { to: 'bob@example.test', cc: '', bcc: '' }
    }
    await settle()
    editor.htmlContent.value = '<p>B draft</p>'
    media.value = [{ id: 2, uuid: 'file-B', filename: 'B.pdf', content_type: 'application/pdf' }]
    expect(editor.completeSend(sent)).toBe(false)
    expect(editor.htmlContent.value).toBe('<p>B draft</p>')
    expect(media.value[0].uuid).toBe('file-B')
    expect(store.getDraft('A', 'reply')).toBeUndefined()
  })

  it('keeps a newer draft after the same thread send completes', () => {
    editor.htmlContent.value = '<p>First reply</p>'
    const sent = editor.captureDraft()
    editor.htmlContent.value = '<p>New unsent content</p>'
    expect(editor.completeSend(sent)).toBe(false)
    expect(editor.htmlContent.value).toBe('<p>New unsent content</p>')
  })

  it('retains failed-send content and flushes on disposal', async () => {
    editor.htmlContent.value = '<p>Keep me</p>'
    editor.captureDraft()
    uuid.value = 'B'
    await settle()
    expect(store.getDraft('A', 'reply').content).toBe('<p>Keep me</p>')
    editor.htmlContent.value = '<p>B unsaved</p>'
    scope.stop()
    expect(store.getDraft('B', 'reply').content).toBe('<p>B unsaved</p>')
  })

  it('prefills late recipients only until initialized and keeps reply/note drafts separate', async () => {
    uuid.value = 'C'
    defaults.value = { uuid: 'C', ready: false, recipients: { to: '', cc: '', bcc: '' } }
    await settle()
    defaults.value = {
      uuid: 'C',
      ready: true,
      recipients: { to: 'carol@example.test', cc: '', bcc: '' }
    }
    await settle()
    expect(editor.recipients.value.to).toBe('carol@example.test')
    editor.htmlContent.value = '<p>Reply</p>'
    type.value = 'private_note'
    await settle()
    editor.htmlContent.value = '<p>Note</p>'
    type.value = 'reply'
    await settle()
    expect(editor.htmlContent.value).toBe('<p>Reply</p>')
    expect(store.getDraft('C', 'private_note').meta.recipients).toBeUndefined()
  })
})
