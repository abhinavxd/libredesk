// @vitest-environment jsdom

import { describe, expect, it } from 'vitest'
import { Editor } from '@tiptap/vue-3'
import { buildConversationExtensions } from './editorExtensions'
import { conversationReferenceHref } from './conversationReferenceExtension'

describe('conversation reference editor extension', () => {
  it('builds a root-relative conversation link', () => {
    expect(conversationReferenceHref('abc')).toBe('/inboxes/all/conversation/abc')
  })

  it('round-trips a conversation reference through HTML in the conversation editor', () => {
    const content =
      '<p><a data-id="conversation-uuid" data-label="108" href="/inboxes/all/conversation/conversation-uuid" class="ld-conversation-reference">#108</a></p>'
    const editor = new Editor({
      extensions: buildConversationExtensions({ getPlaceholder: () => '' }),
      content
    })
    expect(editor.getJSON().content[0].content[0]).toMatchObject({
      type: 'conversationReference',
      attrs: { id: 'conversation-uuid', label: '108' }
    })
    expect(editor.getHTML()).toContain('href="/inboxes/all/conversation/conversation-uuid"')
    editor.destroy()
  })

  it('derives the link from the conversation id, ignoring a stored href', () => {
    const content =
      '<p><a data-id="conversation-uuid" data-label="108" href="https://evil.example/x" class="ld-conversation-reference">#108</a></p>'
    const editor = new Editor({
      extensions: buildConversationExtensions({ getPlaceholder: () => '' }),
      content
    })
    expect(editor.getHTML()).toContain('href="/inboxes/all/conversation/conversation-uuid"')
    expect(editor.getHTML()).not.toContain('evil.example')
    editor.destroy()
  })
})
