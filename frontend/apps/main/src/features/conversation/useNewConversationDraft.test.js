// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { effectScope, nextTick } from 'vue'
import { clearNewConversationDrafts, hasNewConversationDraft, useNewConversationDraft } from './useNewConversationDraft'

let scopes
const draftFor = (channel) => {
  const scope = effectScope()
  scopes.push(scope)
  return scope.run(() => useNewConversationDraft(channel))
}
beforeEach(() => {
  localStorage.clear()
  scopes = []
})
afterEach(() => scopes.forEach((scope) => scope.stop()))

describe('new conversation draft storage', () => {
  it('round trips uploaded media records without changing the IDs', () => {
    const email = draftFor('email')
    const value = { values: { content: '<p>Message</p>' }, attachments: [
      { id: 42, uuid: 'attachment', filename: 'note.txt', size: 12 }
    ] }
    email.value = value
    expect(draftFor('email').value).toEqual(value)
    expect(hasNewConversationDraft(localStorage)).toBe(true)
  })

  it('round trips partial template parameters separately from the email draft', () => {
    draftFor('email').value = { values: { content: '<p>Email</p>' } }
    const value = { inboxId: '1', templateId: 9, templateParams: { 'body:1': 'Customer A', 'header:1': '' } }
    draftFor('whatsapp').value = value
    expect(draftFor('whatsapp').value).toEqual(value)
    expect(draftFor('email').value.values.content).toBe('<p>Email</p>')
  })

  it('clears a sent draft even if the form unmounts immediately', async () => {
    const email = draftFor('email')
    email.value = { attachments: [{ id: 42 }] }
    await nextTick()
    email.value = null
    scopes[0].stop()
    expect(draftFor('email').value).toBeNull()
    expect(hasNewConversationDraft(localStorage)).toBe(false)
  })

  it('discards both channels without recreating their drafts after unmount', async () => {
    draftFor('email').value = { values: { subject: 'Email' } }
    draftFor('whatsapp').value = { templateId: 9 }
    clearNewConversationDrafts(localStorage)
    scopes.forEach((scope) => scope.stop())
    await nextTick()
    expect(hasNewConversationDraft(localStorage)).toBe(false)
    expect(draftFor('email').value).toBeNull()
    expect(draftFor('whatsapp').value).toBeNull()
  })
})
