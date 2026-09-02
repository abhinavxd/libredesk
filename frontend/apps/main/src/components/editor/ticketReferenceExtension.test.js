// @vitest-environment jsdom

import { describe, expect, it } from 'vitest'
import StarterKit from '@tiptap/starter-kit'
import { Editor } from '@tiptap/vue-3'
import { TicketReference } from './ticketReferenceExtension'

describe('ticket reference editor extension', () => {
  it('round-trips a ticket reference through HTML', () => {
    const content =
      '<p><a data-id="conversation-uuid" data-label="108" data-type="ticket-reference" href="/inboxes/all/conversation/conversation-uuid" class="ld-ticket-reference">#108</a></p>'
    const editor = new Editor({ extensions: [StarterKit, TicketReference], content })
    expect(editor.getJSON().content[0].content[0]).toMatchObject({
      type: 'ticketReference',
      attrs: {
        id: 'conversation-uuid',
        label: '108',
        href: '/inboxes/all/conversation/conversation-uuid',
        type: 'ticket-reference'
      }
    })
    expect(editor.getHTML()).toContain('href="/inboxes/all/conversation/conversation-uuid"')
    editor.destroy()
  })
})
